//! In-process simulator of the Canvas migration state machine.
//!
//! The Restate workflow (`canvas_migration::CanvasMigrationImpl`) uses
//! `ctx.run`, `ctx.set`, and `ctx.awakeable` — Restate runtime primitives
//! that can't be exercised in a plain unit test. To keep the state-machine
//! covered we extract the same transitions into pure functions here, then
//! assert:
//!
//!   * `import_courses_step` checkpoints `last_imported_page` every page →
//!     a simulated crash resumes at `last + 1` without re-importing.
//!   * `import_courses_step` returns the suggested backoff from the rate
//!     limit header so the real workflow can `ctx.sleep` it durably.
//!   * `reconcile_step` matches by email via an injected predicate and
//!     counts created / matched / skipped.
//!
//! These functions mirror the body of the corresponding Restate handlers
//! one-for-one — if the real handler changes, update the simulator and a
//! failing test here is the cheap way to catch drift.

use std::time::Duration;

use crate::canvas_client::{backoff_for_remaining, CanvasApi, CanvasUser};
use crate::roster::{self, ReconcileOutcome};
use crate::state::{CanvasMigrationState, CanvasMigrationStatus};

pub struct ImportStepOutcome {
    pub pages_processed: u32,
    pub total_backoff: Duration,
    pub completed: bool,
}

/// Simulate the body of `CanvasMigration::import_courses`. Drives the fake
/// client until either all pages are consumed or `max_pages` is reached,
/// tracking backoff calls that the Restate handler would otherwise emit
/// as `ctx.sleep`.
pub async fn import_courses_step(
    canvas: &dyn CanvasApi,
    state: &mut CanvasMigrationState,
    max_pages: u32,
) -> anyhow::Result<ImportStepOutcome> {
    assert!(state.access_token.is_some(), "call connect first");
    state.status = CanvasMigrationStatus::ImportingCourses;

    let token = state.access_token.clone().unwrap();
    let base = state.canvas_base_url.clone();
    let mut page = state.last_imported_page + 1;
    let limit = if max_pages == 0 { u32::MAX } else { max_pages };

    let mut pages = 0u32;
    let mut total_backoff = Duration::ZERO;
    let mut completed = false;

    loop {
        let resp = canvas.list_courses_page(&base, &token, page).await?;
        state.total_courses_imported += resp.items.len() as u32;
        state.last_imported_page = page;

        let backoff = backoff_for_remaining(resp.rate_limit_remaining);
        total_backoff += backoff;

        page += 1;
        pages += 1;
        if !resp.has_more {
            completed = true;
            break;
        }
        if pages >= limit {
            break;
        }
    }
    if completed {
        state.status = CanvasMigrationStatus::ImportedCourses;
    }
    Ok(ImportStepOutcome {
        pages_processed: pages,
        total_backoff,
        completed,
    })
}

/// Simulate the body of `CanvasMigration::reconcile_rosters`. Pages through
/// the fake client, then runs `roster::reconcile` with the provided
/// existence predicate.
pub async fn reconcile_step(
    canvas: &dyn CanvasApi,
    state: &mut CanvasMigrationState,
    is_existing_user: impl FnMut(&str) -> bool,
) -> anyhow::Result<ReconcileOutcome> {
    assert_eq!(
        state.status,
        CanvasMigrationStatus::ImportedCourses,
        "reconcile requires courses imported"
    );
    state.status = CanvasMigrationStatus::ReconcilingRosters;

    let token = state.access_token.clone().unwrap_or_default();
    let base = state.canvas_base_url.clone();

    let mut users: Vec<CanvasUser> = Vec::new();
    let mut page = 1u32;
    loop {
        let p = canvas.list_users_page(&base, &token, page).await?;
        users.extend(p.items);
        if !p.has_more {
            break;
        }
        page += 1;
    }

    let outcome = roster::reconcile(&users, is_existing_user);
    state.total_users_matched = outcome.matched as u32;
    state.total_users_created = outcome.created as u32;
    state.status = CanvasMigrationStatus::ReconciledRosters;
    Ok(outcome)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::canvas_client::tests::FakeCanvas;
    use crate::canvas_client::{CanvasCourse, CanvasUser};
    use std::sync::atomic::AtomicU32;
    use std::sync::Arc;

    fn state_with_token() -> CanvasMigrationState {
        CanvasMigrationState {
            id: "mig-1".into(),
            tenant_id: Some("t-1".into()),
            canvas_base_url: "https://canvas.example".into(),
            access_token: Some("token".into()),
            status: CanvasMigrationStatus::Connected,
            ..Default::default()
        }
    }

    fn courses(n: usize) -> Vec<CanvasCourse> {
        (1..=n as u64)
            .map(|i| CanvasCourse {
                id: i,
                name: format!("Course {i}"),
                course_code: Some(format!("C{i:03}")),
            })
            .collect()
    }

    #[tokio::test]
    async fn import_checkpoints_every_page_and_completes() {
        let fake = FakeCanvas {
            courses: courses(125),
            users: vec![],
            per_page: 50,
            course_page_rate_limits: vec![Some(5000.0), Some(4950.0), Some(4900.0)],
            calls: Arc::new(AtomicU32::new(0)),
        };
        let mut state = state_with_token();
        let outcome = import_courses_step(&fake, &mut state, 0).await.unwrap();
        assert!(outcome.completed);
        assert_eq!(outcome.pages_processed, 3);
        assert_eq!(state.total_courses_imported, 125);
        assert_eq!(state.last_imported_page, 3);
        assert_eq!(state.status, CanvasMigrationStatus::ImportedCourses);
        assert_eq!(outcome.total_backoff, Duration::ZERO);
    }

    #[tokio::test]
    async fn import_resumes_mid_import_after_simulated_crash() {
        // First invocation runs 1 page then "crashes" (capped via max_pages=1).
        let fake = FakeCanvas {
            courses: courses(150),
            users: vec![],
            per_page: 50,
            course_page_rate_limits: vec![],
            calls: Arc::new(AtomicU32::new(0)),
        };
        let mut state = state_with_token();
        let first = import_courses_step(&fake, &mut state, 1).await.unwrap();
        assert_eq!(first.pages_processed, 1);
        assert!(!first.completed);
        assert_eq!(state.last_imported_page, 1);
        assert_eq!(state.total_courses_imported, 50);

        // Resume — fresh invocation picks up at page 2, no duplicates.
        let resume = import_courses_step(&fake, &mut state, 0).await.unwrap();
        assert!(resume.completed);
        assert_eq!(resume.pages_processed, 2); // pages 2, 3
        assert_eq!(state.last_imported_page, 3);
        assert_eq!(
            state.total_courses_imported, 150,
            "resume must continue from checkpoint without re-importing earlier pages"
        );
    }

    #[tokio::test]
    async fn import_accumulates_backoff_on_low_rate_limit() {
        let fake = FakeCanvas {
            courses: courses(100),
            users: vec![],
            per_page: 50,
            // First page reports low remaining → workflow should request backoff;
            // second page is fine.
            course_page_rate_limits: vec![Some(42.0), Some(5000.0)],
            calls: Arc::new(AtomicU32::new(0)),
        };
        let mut state = state_with_token();
        let outcome = import_courses_step(&fake, &mut state, 0).await.unwrap();
        assert!(outcome.completed);
        assert!(
            outcome.total_backoff > Duration::ZERO,
            "low rate-limit remaining must produce backoff"
        );
    }

    #[tokio::test]
    async fn reconcile_matches_existing_and_creates_unmatched() {
        let users = vec![
            CanvasUser {
                id: 1,
                email: Some("alice@x.edu".into()),
                name: None,
            },
            CanvasUser {
                id: 2,
                email: Some("bob@x.edu".into()),
                name: None,
            },
            CanvasUser {
                id: 3,
                email: None,
                name: None,
            },
        ];
        let fake = FakeCanvas {
            courses: vec![],
            users,
            per_page: 100,
            course_page_rate_limits: vec![],
            calls: Arc::new(AtomicU32::new(0)),
        };
        let mut state = state_with_token();
        state.status = CanvasMigrationStatus::ImportedCourses;
        let outcome = reconcile_step(&fake, &mut state, |email| email == "alice@x.edu")
            .await
            .unwrap();
        assert_eq!(outcome.matched, 1);
        assert_eq!(outcome.created, 1);
        assert_eq!(outcome.skipped_missing_email, 1);
        assert_eq!(state.total_users_matched, 1);
        assert_eq!(state.total_users_created, 1);
        assert_eq!(state.status, CanvasMigrationStatus::ReconciledRosters);
    }
}
