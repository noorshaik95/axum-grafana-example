//! Kafka integration for video-conferencing-service.
//!
//! Consumes `course.lecture_started` events (W11.1) and initializes a
//! live-lecture session in the attendance store. Uses the
//! `libs/common-rust` propagation helpers to link consumer spans back to
//! the producer.

pub mod consumer;

pub use consumer::{LectureStartedEvent, dispatch_lecture_started};
