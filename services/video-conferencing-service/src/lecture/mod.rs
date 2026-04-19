//! W11.1 — Live lecture mode.
//!
//! Tracks per-session attendance (Redis sorted-set semantics) and computes
//! a "pulse" = active attendees / enrolled count. Abstracted over
//! [`AttendanceStore`] so tests can run without a live Redis.

pub mod attendance;
pub mod service;

pub use attendance::{AttendanceStore, InMemoryAttendance, LectureState};
pub use service::{LectureService, LecturePulseSnapshot};
