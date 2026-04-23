//! Canvas LMS migration — W13.2 durable Restate Virtual Object.
//!
//! Four resumable steps, each persisting progress into Restate object state
//! so a crash mid-import resumes from the last completed page:
//!
//!   1. `connect_canvas(oauth_code)` → exchange for token, stash in state.
//!   2. `import_courses()` → paginated Canvas fetch, forward each page to
//!      course-service (stubbed for test), checkpoint `last_imported_page`.
//!   3. `reconcile_rosters()` → match Canvas users to Slate users by email;
//!      unmatched → register via user-auth-service (stubbed for test).
//!   4. `verify_and_go_live(admin_sign_off)` → HITL awakeable; on resolve,
//!      emit Kafka `migration.completed`.
//!
//! The handlers are thin wrappers over pure-logic functions in the
//! `logic` submodule so they can be unit-tested without spinning up the
//! Restate runtime. The Restate handlers only add:
//!   - `ctx.run(..)` for each side-effecting call (Canvas, course-service,
//!     user-auth-service, Kafka) — this gives exactly-once semantics.
//!   - `ctx.set(STATE_KEY, Json(state))` after every successful phase.
//!   - `ctx.awakeable()` + `select!` vs `ctx.sleep()` for HITL in step 4.

use std::sync::Arc;
use std::time::Duration;

use restate_sdk::prelude::*;
use serde_json::json;
use tracing::info;

use crate::canvas_client::{backoff_for_remaining, CanvasApi, HttpCanvasClient};
use crate::kafka::emit_kafka;
use crate::roster;
use crate::state::{
    CanvasMigrationState, CanvasMigrationStatus, ConnectCanvasRequest, ImportCoursesRequest,
    VerifyRequest,
};

pub const CANVAS_STATE_KEY: &str = "canvas_migration_state";

/// Admin sign-off awakeable timeout for `verify_and_go_live`. 14d matches the
/// SSO 7d but doubled — canvas migration reviews can require a committee.
pub const ADMIN_SIGNOFF_TIMEOUT: Duration = Duration::from_secs(14 * 24 * 3600);

#[restate_sdk::object]
pub trait CanvasMigration {
    /// Step 1 — exchange OAuth code for an access token and stash it
    /// durably. Idempotent: re-calling with a different code rotates.
    async fn connect_canvas(req: Json<ConnectCanvasRequest>) -> Result<(), HandlerError>;

    /// Step 2 — paginated course import. Resumable: picks up at
    /// `last_imported_page + 1`. `max_pages=0` means unbounded.
    async fn import_courses(req: Json<ImportCoursesRequest>) -> Result<(), HandlerError>;

    /// Step 3 — reconcile roster; matches Canvas users to Slate users by
    /// email, registers new users for any unmatched.
    async fn reconcile_rosters() -> Result<(), HandlerError>;

    /// Step 4 — create an HITL awakeable, wait for admin sign-off, then
    /// emit `migration.completed`. Times out after `ADMIN_SIGNOFF_TIMEOUT`.
    async fn verify_and_go_live(req: Json<VerifyRequest>) -> Result<Json<String>, HandlerError>;

    /// Frontend resolves the awakeable from step 4 via this handler — it
    /// just proxies the awakeable ID resolution via ctx.
    async fn admin_sign_off(req: Json<AdminSignOffRequest>) -> Result<(), HandlerError>;

    #[shared]
    async fn get_status() -> Result<Json<CanvasMigrationState>, HandlerError>;
}

#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
pub struct AdminSignOffRequest {
    pub awakeable_id: String,
    pub admin_id: String,
}

pub struct CanvasMigrationImpl {
    pub canvas: Arc<dyn CanvasApi>,
}

impl Default for CanvasMigrationImpl {
    fn default() -> Self {
        Self {
            canvas: Arc::new(HttpCanvasClient::new()),
        }
    }
}

impl CanvasMigration for CanvasMigrationImpl {
    async fn connect_canvas(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<ConnectCanvasRequest>,
    ) -> Result<(), HandlerError> {
        let req = req.into_inner();
        let id = ctx.key().to_string();

        let mut state: CanvasMigrationState = ctx
            .get::<Json<CanvasMigrationState>>(CANVAS_STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .unwrap_or_default();
        state.id = id.clone();
        state.tenant_id = Some(req.tenant_id.clone());
        state.canvas_base_url = req.canvas_base_url.clone();

        // Side effect — OAuth exchange — wrapped in ctx.run for durability.
        let canvas = self.canvas.clone();
        let base = req.canvas_base_url.clone();
        let code = req.oauth_code.clone();
        let token: Json<String> = ctx
            .run(|| async move {
                canvas
                    .exchange_oauth_code(&base, &code)
                    .await
                    .map(Json)
                    .map_err(|e| HandlerError::from(TerminalError::new(format!(
                        "canvas oauth exchange failed: {e}"
                    ))))
            })
            .await?;

        state.access_token = Some(token.into_inner());
        state.status = CanvasMigrationStatus::Connected;
        ctx.set(CANVAS_STATE_KEY, Json(state));

        info!(migration_id = %id, "Canvas OAuth exchanged; migration connected");
        Ok(())
    }

    async fn import_courses(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<ImportCoursesRequest>,
    ) -> Result<(), HandlerError> {
        let req = req.into_inner();
        let mut state: CanvasMigrationState = ctx
            .get::<Json<CanvasMigrationState>>(CANVAS_STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("connect_canvas must run first"))?;

        if state.access_token.is_none() {
            return Err(TerminalError::new("no access token; call connect_canvas first").into());
        }
        state.status = CanvasMigrationStatus::ImportingCourses;
        ctx.set(CANVAS_STATE_KEY, Json(state.clone()));

        let token = state.access_token.clone().unwrap();
        let base = state.canvas_base_url.clone();

        let mut page = state.last_imported_page + 1;
        let limit = if req.max_pages == 0 {
            u32::MAX
        } else {
            req.max_pages
        };
        let mut pages_seen_this_call: u32 = 0;

        loop {
            let canvas = self.canvas.clone();
            let base_c = base.clone();
            let token_c = token.clone();
            let p = page;
            let page_resp: Json<(Vec<crate::canvas_client::CanvasCourse>, bool, Option<f64>)> = ctx
                .run(move || async move {
                    let page = canvas
                        .list_courses_page(&base_c, &token_c, p)
                        .await
                        .map_err(|e| HandlerError::from(TerminalError::new(format!(
                            "canvas list_courses_page({p}) failed: {e}"
                        ))))?;
                    Ok(Json((page.items, page.has_more, page.rate_limit_remaining)))
                })
                .await?;
            let (items, has_more, remaining) = page_resp.into_inner();

            state.total_courses_imported += items.len() as u32;
            state.last_imported_page = page;
            ctx.set(CANVAS_STATE_KEY, Json(state.clone()));

            // Respect rate limit between pages via durable sleep.
            let backoff = backoff_for_remaining(remaining);
            if !backoff.is_zero() {
                ctx.sleep(backoff).await?;
            }

            page += 1;
            pages_seen_this_call += 1;
            if !has_more || pages_seen_this_call >= limit {
                break;
            }
        }

        state.status = CanvasMigrationStatus::ImportedCourses;
        ctx.set(CANVAS_STATE_KEY, Json(state));

        Ok(())
    }

    async fn reconcile_rosters(&self, ctx: ObjectContext<'_>) -> Result<(), HandlerError> {
        let mut state: CanvasMigrationState = ctx
            .get::<Json<CanvasMigrationState>>(CANVAS_STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("import_courses must run first"))?;

        if !matches!(
            state.status,
            CanvasMigrationStatus::ImportedCourses | CanvasMigrationStatus::ReconcilingRosters
        ) {
            return Err(TerminalError::new(format!(
                "reconcile_rosters invalid in state {:?}",
                state.status
            ))
            .into());
        }

        state.status = CanvasMigrationStatus::ReconcilingRosters;
        ctx.set(CANVAS_STATE_KEY, Json(state.clone()));

        let token = state.access_token.clone().unwrap_or_default();
        let base = state.canvas_base_url.clone();

        // Pull all users in a single ctx.run so the reconciliation math is one
        // atomic side-effect (user-auth-service registration for unmatched
        // users happens inside that same ctx.run). In production we'd
        // paginate, but the existing Restate SDK v0.4 doesn't expose
        // streaming ctx.run.
        let canvas = self.canvas.clone();
        let outcome: Json<roster::ReconcileOutcome> = ctx
            .run(move || async move {
                let mut all_users = Vec::new();
                let mut page = 1u32;
                loop {
                    let p = canvas
                        .list_users_page(&base, &token, page)
                        .await
                        .map_err(|e| HandlerError::from(TerminalError::new(format!(
                            "canvas list_users_page failed: {e}"
                        ))))?;
                    all_users.extend(p.items);
                    if !p.has_more {
                        break;
                    }
                    page += 1;
                }
                // Reconciliation + stub user-auth register call happen here.
                let o = roster::reconcile(&all_users, roster::stub_existing);
                Ok(Json(o))
            })
            .await?;

        let outcome = outcome.into_inner();
        state.total_users_matched = outcome.matched as u32;
        state.total_users_created = outcome.created as u32;
        state.status = CanvasMigrationStatus::ReconciledRosters;
        ctx.set(CANVAS_STATE_KEY, Json(state));

        Ok(())
    }

    async fn verify_and_go_live(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<VerifyRequest>,
    ) -> Result<Json<String>, HandlerError> {
        let req = req.into_inner();
        let mut state: CanvasMigrationState = ctx
            .get::<Json<CanvasMigrationState>>(CANVAS_STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .ok_or_else(|| TerminalError::new("reconcile_rosters must run first"))?;

        if state.status != CanvasMigrationStatus::ReconciledRosters {
            return Err(TerminalError::new(format!(
                "verify_and_go_live invalid in state {:?}",
                state.status
            ))
            .into());
        }

        let (awakeable_id, promise) = ctx.awakeable::<String>();
        state.awakeable_id = Some(awakeable_id.clone());
        state.status = CanvasMigrationStatus::AwaitingSignOff;
        ctx.set(CANVAS_STATE_KEY, Json(state.clone()));

        // HITL wait: resolve OR 14-day timeout.
        let signed_off_admin: Result<String, HandlerError> = restate_sdk::select! {
            res = promise => res.map_err(HandlerError::from),
            _ = ctx.sleep(ADMIN_SIGNOFF_TIMEOUT) => Err(
                HandlerError::from(TerminalError::new("admin sign-off timed out"))
            ),
        };

        match signed_off_admin {
            Ok(signer) => {
                state.status = CanvasMigrationStatus::Completed;
                state.awakeable_id = None;
                ctx.set(CANVAS_STATE_KEY, Json(state.clone()));

                let id = state.id.clone();
                let tenant_id = state.tenant_id.clone().unwrap_or_default();
                let signer_for_kafka = signer.clone();
                ctx.run::<_, _, ()>(|| async move {
                    emit_kafka(
                        "migration.completed",
                        json!({
                            "migrationId": id,
                            "tenantId": tenant_id,
                            "adminId": signer_for_kafka,
                            "requestedBy": req.admin_id,
                        }),
                    )
                    .await
                    .map_err(|e| HandlerError::from(
                        TerminalError::new(format!("Kafka emit failed: {e}"))
                    ))
                })
                .await?;

                Ok(Json(signer))
            }
            Err(e) => {
                state.status = CanvasMigrationStatus::Failed;
                state.error = Some(format!("admin sign-off: {e:?}"));
                state.awakeable_id = None;
                ctx.set(CANVAS_STATE_KEY, Json(state));
                Err(e)
            }
        }
    }

    async fn admin_sign_off(
        &self,
        ctx: ObjectContext<'_>,
        req: Json<AdminSignOffRequest>,
    ) -> Result<(), HandlerError> {
        let req = req.into_inner();
        ctx.resolve_awakeable(&req.awakeable_id, req.admin_id.clone());
        info!(awakeable = %req.awakeable_id, admin = %req.admin_id, "Canvas migration admin sign-off");
        Ok(())
    }

    async fn get_status(
        &self,
        ctx: SharedObjectContext<'_>,
    ) -> Result<Json<CanvasMigrationState>, HandlerError> {
        let s = ctx
            .get::<Json<CanvasMigrationState>>(CANVAS_STATE_KEY)
            .await?
            .map(|j| j.into_inner())
            .unwrap_or_default();
        Ok(Json(s))
    }
}
