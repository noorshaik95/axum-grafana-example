use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Serialize, Deserialize, Default, Clone, Debug, PartialEq)]
pub enum OnboardingStatus {
    #[default]
    Draft,
    StepInProgress,
    PendingReview,
    Approved,
    Provisioning,
    Active,
    Rejected,
    Failed,
}

#[derive(Serialize, Deserialize, Default, Clone, Debug)]
pub struct OnboardingState {
    pub id: String,
    pub status: OnboardingStatus,
    pub steps: HashMap<u8, serde_json::Value>,
    pub institution_name: String,
    pub admin_email: String,
    pub submitted_at: Option<String>,
    pub approved_by: Option<String>,
    pub tenant_id: Option<String>,
    pub error: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct StartPayload {
    pub institution_name: String,
    pub admin_email: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct SaveStepRequest {
    pub step_number: u8,
    pub data: serde_json::Value,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ApproveRequest {
    pub admin_id: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct RejectRequest {
    pub reason: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct TenantResponse {
    pub tenant_id: String,
}
