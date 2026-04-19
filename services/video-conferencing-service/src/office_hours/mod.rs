//! W11.4 — Office hours rooms.
//!
//! Creates a video room keyed to a `booking_id` from scheduling-service.
//! Backend is either native WebRTC (default) or Zoom depending on tenant
//! config. Returns a join URL + TTL.

use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OhBackend {
    Native,
    Zoom,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OhRoom {
    pub room_id: String,
    pub booking_id: String,
    pub tenant_slug: String,
    pub instructor_id: String,
    pub student_id: String,
    pub join_url: String,
    pub backend: OhBackend,
    pub starts_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct CreateOhRoomInput<'a> {
    pub booking_id: &'a str,
    pub tenant_slug: &'a str,
    pub instructor_id: &'a str,
    pub student_id: &'a str,
    pub starts_at: DateTime<Utc>,
    pub duration_minutes: i32,
    pub backend: OhBackend,
}

/// Native join URL shape. Mirrors the shape used by `generate_join_url` in
/// the gRPC service so the frontend can resolve either consistently.
fn native_join_url(host: &str, room_id: &str) -> String {
    format!("https://{}/oh/{}", host, room_id)
}

/// Build a room for an office-hours booking. For Zoom backend we return a
/// placeholder URL — a real Zoom meeting creation requires the W11.3 token
/// flow to be ratified. The room still has a valid `expires_at` so the
/// caller can safely gate access.
pub fn create_room(host: &str, input: CreateOhRoomInput<'_>) -> OhRoom {
    let room_id = Uuid::new_v4().to_string();
    let expires_at = input.starts_at + Duration::minutes(input.duration_minutes as i64 + 15);
    let join_url = match input.backend {
        OhBackend::Native => native_join_url(host, &room_id),
        OhBackend::Zoom => {
            // TODO(W11.3): issue a real Zoom meeting via `zoom::create_meeting`.
            format!("https://zoom.us/j/pending-{}", room_id)
        }
    };
    OhRoom {
        room_id,
        booking_id: input.booking_id.to_string(),
        tenant_slug: input.tenant_slug.to_string(),
        instructor_id: input.instructor_id.to_string(),
        student_id: input.student_id.to_string(),
        join_url,
        backend: input.backend,
        starts_at: input.starts_at,
        expires_at,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn input(backend: OhBackend) -> CreateOhRoomInput<'static> {
        CreateOhRoomInput {
            booking_id: "b-1",
            tenant_slug: "eastfield",
            instructor_id: "instr-1",
            student_id: "stu-1",
            starts_at: Utc::now(),
            duration_minutes: 30,
            backend,
        }
    }

    #[test]
    fn native_room_join_url_uses_oh_prefix() {
        let room = create_room("video.slate.local", input(OhBackend::Native));
        assert_eq!(room.backend, OhBackend::Native);
        assert!(room.join_url.contains("/oh/"));
        assert_eq!(room.booking_id, "b-1");
    }

    #[test]
    fn zoom_room_uses_zoom_url_shape() {
        let room = create_room("video.slate.local", input(OhBackend::Zoom));
        assert_eq!(room.backend, OhBackend::Zoom);
        assert!(room.join_url.starts_with("https://zoom.us/"));
    }

    #[test]
    fn expires_at_extends_past_scheduled_end() {
        let starts = Utc::now();
        let room = create_room(
            "video.slate.local",
            CreateOhRoomInput {
                starts_at: starts,
                duration_minutes: 30,
                ..input(OhBackend::Native)
            },
        );
        // 15-minute grace buffer.
        assert!(room.expires_at > starts + Duration::minutes(30));
    }
}
