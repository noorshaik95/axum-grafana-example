use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct VideoSession {
    pub id: Uuid,
    pub instructor_id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub start_time: DateTime<Utc>,
    pub duration_minutes: i32,
    pub status: String,
    pub max_participants: i32,
    pub auto_record: bool,
    pub allow_screen_share: bool,
    pub require_approval: bool,
    pub mute_on_join: bool,
    pub join_url: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct VideoQualitySettings {
    pub id: Uuid,
    pub session_id: Uuid,
    pub default_quality: String,
    pub adaptive_bitrate: bool,
    pub min_bitrate_kbps: i32,
    pub max_bitrate_kbps: i32,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SessionParticipant {
    pub id: Uuid,
    pub session_id: Uuid,
    pub user_id: Uuid,
    pub display_name: String,
    pub role: String,
    pub audio_enabled: bool,
    pub video_enabled: bool,
    pub is_muted: bool,
    pub is_sharing_screen: bool,
    pub joined_at: DateTime<Utc>,
    pub left_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct ParticipantQualityStats {
    pub id: Uuid,
    pub participant_id: Uuid,
    pub current_quality: String,
    pub bitrate_kbps: i32,
    pub frame_rate: i32,
    pub packet_loss_percentage: i32,
    pub jitter_ms: i32,
    pub latency_ms: i32,
    pub bytes_sent: i64,
    pub bytes_received: i64,
    pub recorded_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SessionRecording {
    pub id: Uuid,
    pub session_id: Uuid,
    pub started_at: DateTime<Utc>,
    pub completed_at: Option<DateTime<Utc>>,
    pub duration_seconds: Option<i32>,
    pub file_size_bytes: Option<i64>,
    pub gcs_url: Option<String>,
    pub gcs_bucket: Option<String>,
    pub gcs_object_key: Option<String>,
    pub status: String,
    pub available_at: Option<DateTime<Utc>>,
    pub quality: String,
    pub include_audio: bool,
    pub include_screen_share: bool,
    pub error_message: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct ScreenShare {
    pub id: Uuid,
    pub session_id: Uuid,
    pub participant_id: Uuid,
    pub started_at: DateTime<Utc>,
    pub stopped_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct CalendarInvitation {
    pub id: Uuid,
    pub session_id: Uuid,
    pub recipient_user_id: Uuid,
    pub recipient_email: String,
    pub sent_at: DateTime<Utc>,
    pub calendar_ics: String,
    pub delivery_status: String,
    pub error_message: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct WebRtcEvent {
    pub id: Uuid,
    pub session_id: Uuid,
    pub participant_id: Option<Uuid>,
    pub event_type: String,
    pub event_data: Option<serde_json::Value>,
    pub created_at: DateTime<Utc>,
}
