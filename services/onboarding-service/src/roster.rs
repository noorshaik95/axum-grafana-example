//! Roster reconciliation — pure logic extracted from the Canvas migration
//! workflow so it can be unit-tested without Restate or HTTP.

use crate::canvas_client::CanvasUser;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct ReconcileOutcome {
    pub matched: usize,
    pub created: usize,
    pub skipped_missing_email: usize,
}

/// Reconcile Canvas users against the Slate user base.
///
/// `is_existing_user(email) -> bool` is injected so tests can drive the
/// matching logic and the production workflow can swap in a user-auth
/// lookup. The returned `ReconcileOutcome` counts:
///   - `matched`: Canvas user whose email already exists in Slate.
///   - `created`: Canvas user whose email is new → would be registered via
///     user-auth-service. Registration happens in the production Restate
///     handler inside the same `ctx.run` that called this function; here
///     we only count.
///   - `skipped_missing_email`: Canvas users with no email — can't match
///     by email, so they're deferred to manual review.
pub fn reconcile<F>(canvas_users: &[CanvasUser], mut is_existing_user: F) -> ReconcileOutcome
where
    F: FnMut(&str) -> bool,
{
    let mut out = ReconcileOutcome::default();
    for u in canvas_users {
        match u.email.as_deref().map(str::trim).filter(|e| !e.is_empty()) {
            None => out.skipped_missing_email += 1,
            Some(email) => {
                if is_existing_user(email) {
                    out.matched += 1;
                } else {
                    out.created += 1;
                }
            }
        }
    }
    out
}

/// Placeholder matcher — always returns false so every Canvas user ends up
/// in `created`. Replace with a user-auth-service lookup once the gRPC
/// client is wired.
pub fn stub_existing(_email: &str) -> bool {
    false
}

#[cfg(test)]
mod tests {
    use super::*;

    fn u(id: u64, email: Option<&str>) -> CanvasUser {
        CanvasUser {
            id,
            email: email.map(|s| s.to_string()),
            name: None,
        }
    }

    #[test]
    fn counts_matched_and_created() {
        let users = vec![
            u(1, Some("alice@school.edu")),
            u(2, Some("bob@school.edu")),
            u(3, Some("carol@school.edu")),
        ];
        let outcome = reconcile(&users, |email| email == "bob@school.edu");
        assert_eq!(outcome.matched, 1);
        assert_eq!(outcome.created, 2);
        assert_eq!(outcome.skipped_missing_email, 0);
    }

    #[test]
    fn skips_users_missing_email() {
        let users = vec![
            u(1, Some("alice@school.edu")),
            u(2, None),
            u(3, Some("")),
        ];
        let outcome = reconcile(&users, |_| false);
        assert_eq!(outcome.matched, 0);
        assert_eq!(outcome.created, 1);
        assert_eq!(outcome.skipped_missing_email, 2);
    }

    #[test]
    fn empty_input_zero_counts() {
        let outcome = reconcile(&[], |_| true);
        assert_eq!(outcome, ReconcileOutcome::default());
    }

    #[test]
    fn whitespace_only_email_is_skipped() {
        let users = vec![u(1, Some("   "))];
        let outcome = reconcile(&users, |_| true);
        assert_eq!(outcome.skipped_missing_email, 1);
        assert_eq!(outcome.matched, 0);
    }
}
