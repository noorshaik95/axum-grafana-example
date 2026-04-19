//! W11.2 — Q&A queue.
//!
//! Redis list `tenant:{slug}:live:{session_id}:questions` (in prod) or an
//! in-memory equivalent in dev/tests. Questions are pushed with zero
//! upvotes; `upvote` atomically INCRs the upvote counter; `list` returns
//! questions sorted by upvotes desc, ties broken by submitted_at asc.

pub mod service;
pub mod store;

pub use service::QaService;
pub use store::{InMemoryQaStore, QaStore, Question};
