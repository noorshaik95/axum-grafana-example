use super::attendance::{AttendanceStore, LectureState};
use chrono::Utc;
use std::sync::Arc;

/// Snapshot returned by `GetLecturePulse` — `attendee_count / enrolled_count`.
#[derive(Debug, Clone, PartialEq)]
pub struct LecturePulseSnapshot {
    pub session_id: String,
    pub attendee_count: i32,
    pub enrolled_count: i32,
    pub pulse: f64,
}

impl LecturePulseSnapshot {
    /// Compute pulse. Returns 0.0 when `enrolled_count <= 0` (no division by
    /// zero; no NaN sneaking into proto floats).
    pub fn compute(session_id: String, attendee_count: i32, enrolled_count: i32) -> Self {
        let pulse = if enrolled_count > 0 {
            (attendee_count as f64) / (enrolled_count as f64)
        } else {
            0.0
        };
        Self {
            session_id,
            attendee_count,
            enrolled_count,
            pulse,
        }
    }
}

/// High-level lecture lifecycle service. Wraps an attendance store and owns
/// the session -> pulse calculation.
#[derive(Clone)]
pub struct LectureService {
    store: Arc<dyn AttendanceStore>,
}

impl LectureService {
    pub fn new(store: Arc<dyn AttendanceStore>) -> Self {
        Self { store }
    }

    pub async fn start_lecture(
        &self,
        session_id: &str,
        tenant_slug: &str,
        enrolled_count: i32,
    ) -> LectureState {
        let state = LectureState {
            session_id: session_id.to_string(),
            tenant_slug: tenant_slug.to_string(),
            enrolled_count,
            started_at_unix: Utc::now().timestamp(),
            active: true,
        };
        self.store.init_session(state.clone()).await;
        state
    }

    pub async fn end_lecture(&self, session_id: &str, tenant_slug: &str) {
        self.store.end_session(tenant_slug, session_id).await;
    }

    /// Record a participant join. Returns (join_timestamp, attendee_count).
    /// `accepted = false` when the lecture has not been initialized.
    pub async fn join_lecture(
        &self,
        session_id: &str,
        tenant_slug: &str,
        user_id: &str,
    ) -> JoinOutcome {
        let state = self.store.get_state(tenant_slug, session_id).await;
        match state {
            Some(s) if s.active => {
                let ts = Utc::now().timestamp();
                self.store
                    .record_join(tenant_slug, session_id, user_id, ts)
                    .await;
                let count = self.store.attendee_count(tenant_slug, session_id).await;
                JoinOutcome {
                    accepted: true,
                    join_timestamp_unix: ts,
                    attendee_count: count,
                }
            }
            _ => JoinOutcome {
                accepted: false,
                join_timestamp_unix: 0,
                attendee_count: 0,
            },
        }
    }

    pub async fn pulse(&self, session_id: &str, tenant_slug: &str) -> Option<LecturePulseSnapshot> {
        let state = self.store.get_state(tenant_slug, session_id).await?;
        let count = self.store.attendee_count(tenant_slug, session_id).await;
        Some(LecturePulseSnapshot::compute(
            session_id.to_string(),
            count,
            state.enrolled_count,
        ))
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct JoinOutcome {
    pub accepted: bool,
    pub join_timestamp_unix: i64,
    pub attendee_count: i32,
}

#[cfg(test)]
mod tests {
    use super::super::attendance::InMemoryAttendance;
    use super::*;

    fn svc() -> LectureService {
        LectureService::new(Arc::new(InMemoryAttendance::new()))
    }

    #[test]
    fn pulse_is_attendee_over_enrolled() {
        let s = LecturePulseSnapshot::compute("s".to_string(), 20, 30);
        assert!((s.pulse - (20.0 / 30.0)).abs() < f64::EPSILON);
    }

    #[test]
    fn pulse_zero_enrolled_avoids_nan() {
        let s = LecturePulseSnapshot::compute("s".to_string(), 5, 0);
        assert_eq!(s.pulse, 0.0);
    }

    #[tokio::test]
    async fn lecture_lifecycle_start_join_pulse_end() {
        let svc = svc();
        svc.start_lecture("s1", "eastfield", 10).await;

        let j1 = svc.join_lecture("s1", "eastfield", "u1").await;
        let j2 = svc.join_lecture("s1", "eastfield", "u2").await;
        assert!(j1.accepted && j2.accepted);
        assert_eq!(j2.attendee_count, 2);

        let pulse = svc.pulse("s1", "eastfield").await.unwrap();
        assert_eq!(pulse.attendee_count, 2);
        assert_eq!(pulse.enrolled_count, 10);
        assert!((pulse.pulse - 0.2).abs() < f64::EPSILON);

        svc.end_lecture("s1", "eastfield").await;
        let after = svc.pulse("s1", "eastfield").await.unwrap();
        assert_eq!(after.attendee_count, 0);
    }

    #[tokio::test]
    async fn join_without_start_is_rejected() {
        let svc = svc();
        let out = svc.join_lecture("ghost", "eastfield", "u1").await;
        assert!(!out.accepted);
        assert_eq!(out.attendee_count, 0);
    }

    #[tokio::test]
    async fn pulse_math_vs_enrolled() {
        // Explicit acceptance: pulse calc vs enrolled.
        let svc = svc();
        svc.start_lecture("s2", "eastfield", 100).await;
        for i in 0..25 {
            svc.join_lecture("s2", "eastfield", &format!("u{}", i)).await;
        }
        let p = svc.pulse("s2", "eastfield").await.unwrap();
        assert_eq!(p.attendee_count, 25);
        assert!((p.pulse - 0.25).abs() < f64::EPSILON);
    }
}
