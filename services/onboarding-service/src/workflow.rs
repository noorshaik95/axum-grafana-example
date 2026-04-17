use restate_sdk::prelude::*;
use serde_json::json;
use tracing::info;

use crate::kafka::emit_kafka;
use crate::state::*;

const STATE_KEY: &str = "onboarding_state";

#[restate_sdk::object]
pub trait OnboardingWorkflow {
    async fn start(payload: Json<StartPayload>) -> Result<(), HandlerError>;
    async fn save_step(req: Json<SaveStepRequest>) -> Result<(), HandlerError>;
    async fn submit_for_review() -> Result<(), HandlerError>;
    async fn approve(req: Json<ApproveRequest>) -> Result<Json<String>, HandlerError>;
    async fn reject(req: Json<RejectRequest>) -> Result<(), HandlerError>;
    async fn reopen() -> Result<(), HandlerError>;

    #[shared]
    async fn get_status() -> Result<Json<OnboardingState>, HandlerError>;
}

pub struct OnboardingWorkflowImpl;

impl OnboardingWorkflow for OnboardingWorkflowImpl {
    async fn start(
        &self,
        ctx: ObjectContext<'_>,
        payload: Json<StartPayload>,
    ) -> Result<(), HandlerError> {
        let payload = payload.into_inner();
        let id = ctx.key().to_string();
        info!(onboarding_id = %id, "Starting onboarding workflow");

        let state = OnboardingState {
            id: id.clone(),
            status: OnboardingStatus::StepInProgress,
            institution_name: payload.institution_name.clone(),
            admin_email: payload.admin_email.clone(),
            ..Default::default()
        };

        ctx.set(STATE_KEY, Json(state));

        let institution_name = payload.institution_name.clone();
        let admin_email = payload.admin_email.clone();
        let id_clone = id.clone();
        ctx.run::<_, _, ()>(|| async move {
            emit_kafka(
                "onboarding.started",
                json!({
                    "onboardingId": id_clone,
                    "institutionName": institution_name,
                    "adminEmail": admin_email,
                }),
            )
            .await
            .map_err(|e| HandlerError::from(TerminalError::new(format!("Kafka emit failed: {e}"))))
        })
        .await?;

        Ok(())
    }

    async fn save_step(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<SaveStepRequest>,
    ) -> Result<(), HandlerError> {
        let req = req.into_inner();
        if !(1..=4).contains(&req.step_number) {
            return Err(TerminalError::new("Step number must be between 1 and 4").into());
        }

        let mut state: OnboardingState = ctx
            .get::<Json<OnboardingState>>(STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("Onboarding not started"))?;

        if state.status != OnboardingStatus::StepInProgress {
            return Err(
                TerminalError::new("Cannot save step: workflow not in StepInProgress state").into(),
            );
        }

        let id = state.id.clone();
        let step_number = req.step_number;

        state.steps.insert(req.step_number, req.data);
        ctx.set(STATE_KEY, Json(state));

        ctx.run::<_, _, ()>(|| async move {
            emit_kafka(
                "onboarding.step_completed",
                json!({
                    "onboardingId": id,
                    "stepNumber": step_number,
                }),
            )
            .await
            .map_err(|e| HandlerError::from(TerminalError::new(format!("Kafka emit failed: {e}"))))
        })
        .await?;

        info!(step = step_number, "Step saved");
        Ok(())
    }

    async fn submit_for_review(&self, ctx: ObjectContext<'_>) -> Result<(), HandlerError> {
        let mut state: OnboardingState = ctx
            .get::<Json<OnboardingState>>(STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("Onboarding not started"))?;

        if state.status != OnboardingStatus::StepInProgress {
            return Err(
                TerminalError::new("Cannot submit: workflow not in StepInProgress state").into(),
            );
        }

        // Validate all 4 steps are completed
        for step in 1..=4u8 {
            if !state.steps.contains_key(&step) {
                return Err(TerminalError::new(format!("Step {step} not completed")).into());
            }
        }

        state.status = OnboardingStatus::PendingReview;
        state.submitted_at = Some(chrono::Utc::now().to_rfc3339());
        let id = state.id.clone();
        ctx.set(STATE_KEY, Json(state));

        ctx.run::<_, _, ()>(|| async move {
            emit_kafka("onboarding.pending_review", json!({ "onboardingId": id }))
                .await
                .map_err(|e| {
                    HandlerError::from(TerminalError::new(format!("Kafka emit failed: {e}")))
                })
        })
        .await?;

        // TODO: In full implementation, start a 72h durable timer here:
        // ctx.sleep(Duration::from_secs(72 * 3600)).await?;
        // Then check if still PendingReview and emit onboarding.escalated

        Ok(())
    }

    async fn approve(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<ApproveRequest>,
    ) -> Result<Json<String>, HandlerError> {
        let req = req.into_inner();
        let mut state: OnboardingState = ctx
            .get::<Json<OnboardingState>>(STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("Onboarding not started"))?;

        if state.status != OnboardingStatus::PendingReview {
            return Err(
                TerminalError::new("Cannot approve: workflow not in PendingReview state").into(),
            );
        }

        state.status = OnboardingStatus::Provisioning;
        state.approved_by = Some(req.admin_id.clone());
        ctx.set(STATE_KEY, Json(state.clone()));

        info!(onboarding_id = %state.id, admin = %req.admin_id, "Provisioning tenant");

        // Provisioning saga — each step uses ctx.run() for exactly-once semantics

        // Step 1: Provision tenant via tenant-service REST API
        let institution_name = state.institution_name.clone();
        let admin_email = state.admin_email.clone();
        let onboarding_id = state.id.clone();

        let tenant_id: Json<String> = ctx
            .run(|| async move {
                let client = reqwest::Client::new();
                let resp = client
                    .post("http://tenant-service:8083/tenants")
                    .json(&json!({
                        "name": institution_name,
                        "adminEmail": admin_email,
                        "onboardingId": onboarding_id,
                    }))
                    .send()
                    .await
                    .map_err(|e| {
                        HandlerError::from(TerminalError::new(format!(
                            "Tenant provisioning failed: {e}"
                        )))
                    })?;

                if !resp.status().is_success() {
                    let status = resp.status();
                    let body = resp.text().await.unwrap_or_default();
                    return Err(TerminalError::new(format!(
                        "Tenant service returned {status}: {body}"
                    ))
                    .into());
                }

                let tenant_resp: TenantResponse = resp.json().await.map_err(|e| {
                    HandlerError::from(TerminalError::new(format!(
                        "Failed to parse tenant response: {e}"
                    )))
                })?;

                Ok(Json(tenant_resp.tenant_id))
            })
            .await?;

        let tenant_id = tenant_id.into_inner();

        // Step 2: Emit Kafka approval event
        let onboarding_id = state.id.clone();
        let institution_name = state.institution_name.clone();
        let tid = tenant_id.clone();

        ctx.run::<_, _, ()>(|| async move {
            emit_kafka(
                "onboarding.approved",
                json!({
                    "onboardingId": onboarding_id,
                    "tenantId": tid,
                    "institutionName": institution_name,
                }),
            )
            .await
            .map_err(|e| HandlerError::from(TerminalError::new(format!("Kafka emit failed: {e}"))))
        })
        .await?;

        // Update state to Active
        state.status = OnboardingStatus::Active;
        state.tenant_id = Some(tenant_id.clone());
        ctx.set(STATE_KEY, Json(state));

        info!(tenant_id = %tenant_id, "Onboarding approved and provisioned");
        Ok(Json(tenant_id))
    }

    async fn reject(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<RejectRequest>,
    ) -> Result<(), HandlerError> {
        let req = req.into_inner();
        let mut state: OnboardingState = ctx
            .get::<Json<OnboardingState>>(STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("Onboarding not started"))?;

        if state.status != OnboardingStatus::PendingReview {
            return Err(
                TerminalError::new("Cannot reject: workflow not in PendingReview state").into(),
            );
        }

        state.status = OnboardingStatus::Rejected;
        state.error = Some(req.reason.clone());
        let id = state.id.clone();
        ctx.set(STATE_KEY, Json(state));

        let reason = req.reason.clone();
        ctx.run::<_, _, ()>(|| async move {
            emit_kafka(
                "onboarding.rejected",
                json!({
                    "onboardingId": id,
                    "reason": reason,
                }),
            )
            .await
            .map_err(|e| HandlerError::from(TerminalError::new(format!("Kafka emit failed: {e}"))))
        })
        .await?;

        info!("Onboarding rejected");
        Ok(())
    }

    async fn reopen(&self, ctx: ObjectContext<'_>) -> Result<(), HandlerError> {
        let mut state: OnboardingState = ctx
            .get::<Json<OnboardingState>>(STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("Onboarding not started"))?;

        match state.status {
            OnboardingStatus::Rejected | OnboardingStatus::PendingReview => {}
            _ => {
                return Err(TerminalError::new(
                    "Cannot reopen: workflow must be in Rejected or PendingReview state",
                )
                .into());
            }
        }

        state.status = OnboardingStatus::StepInProgress;
        state.error = None;
        ctx.set(STATE_KEY, Json(state));

        info!("Onboarding reopened");
        Ok(())
    }

    async fn get_status(
        &self,
        ctx: SharedObjectContext<'_>,
    ) -> Result<Json<OnboardingState>, HandlerError> {
        let state: OnboardingState = ctx
            .get::<Json<OnboardingState>>(STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .unwrap_or_default();
        Ok(Json(state))
    }
}
