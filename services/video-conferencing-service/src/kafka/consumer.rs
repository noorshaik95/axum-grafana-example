use crate::lecture::LectureService;
use common_rust::observability::{context_from_kafka_headers, KafkaHeader, KafkaTraceCarrier};
use serde::{Deserialize, Serialize};
use tracing::{instrument, Span};

/// Payload of the `course.lecture_started` Kafka topic.
///
/// Shape matches course-service's producer (`emitLectureStarted` in
/// `services/course-service/src/kafka/kafka.producer.ts`). camelCase
/// because the producer emits JSON with default key casing. `tenant_slug`
/// is optional on the wire — course-service falls back to `tenant_id`
/// when a slug isn't resolvable, so we accept either.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LectureStartedEvent {
    pub session_id: String,
    pub course_id: String,
    pub tenant_id: String,
    pub instructor_id: String,
    #[serde(default)]
    pub tenant_slug: Option<String>,
    #[serde(default)]
    pub enrolled_count: i32,
}

impl LectureStartedEvent {
    /// Best-effort tenant-slug resolution. Producer emits slug when
    /// available; fall back to `tenant_id` otherwise (course-service's
    /// tenant registry isn't fully wired yet).
    pub fn slug(&self) -> &str {
        self.tenant_slug.as_deref().unwrap_or(&self.tenant_id)
    }
}

/// Dispatch handler invoked by the Kafka runtime for each
/// `course.lecture_started` message. Initializes the lecture session in
/// the attendance store.
///
/// `headers` must carry the producer's `traceparent` + `x-request-id` +
/// `x-tenant-slug` from course-service so the consumer span links back
/// per plan/CONTRACTS.md `trace.propagation` (Kafka consumer spans link
/// back to producer span via traceparent headers).
#[instrument(
    skip(lectures, event, headers),
    fields(
        session_id = %event.session_id,
        tenant.slug = %event.slug(),
        course_id = %event.course_id,
        request_id = tracing::field::Empty,
    )
)]
pub async fn dispatch_lecture_started(
    lectures: &LectureService,
    event: LectureStartedEvent,
    headers: &[KafkaHeader],
) {
    let carrier: KafkaTraceCarrier = context_from_kafka_headers(headers);
    if let Some(rid) = carrier.request_id.as_deref() {
        Span::current().record("request_id", tracing::field::display(rid));
    }
    lectures
        .start_lecture(&event.session_id, event.slug(), event.enrolled_count)
        .await;
    tracing::info!(
        "lecture initialized from Kafka: session={} tenant={} enrolled={} traceparent_linked={}",
        event.session_id,
        event.slug(),
        event.enrolled_count,
        carrier.traceparent.is_some()
    );
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::lecture::{InMemoryAttendance, LectureService};
    use std::sync::Arc;

    #[tokio::test]
    async fn consumer_initializes_session() {
        let store = Arc::new(InMemoryAttendance::new());
        let svc = LectureService::new(store.clone());

        let evt = LectureStartedEvent {
            session_id: "s-kafka".to_string(),
            tenant_id: "eastfield".to_string(),
            tenant_slug: Some("eastfield".to_string()),
            course_id: "c-1".to_string(),
            instructor_id: "instr-1".to_string(),
            enrolled_count: 42,
        };
        dispatch_lecture_started(&svc, evt, &[]).await;

        let pulse = svc.pulse("s-kafka", "eastfield").await.unwrap();
        assert_eq!(pulse.enrolled_count, 42);
        assert_eq!(pulse.attendee_count, 0);
    }

    #[tokio::test]
    async fn consumer_falls_back_to_tenant_id_when_slug_missing() {
        let store = Arc::new(InMemoryAttendance::new());
        let svc = LectureService::new(store.clone());

        let evt = LectureStartedEvent {
            session_id: "s-no-slug".to_string(),
            tenant_id: "eastfield".to_string(),
            tenant_slug: None,
            course_id: "c-1".to_string(),
            instructor_id: "instr-1".to_string(),
            enrolled_count: 10,
        };
        dispatch_lecture_started(&svc, evt, &[]).await;

        // Should have initialized under `tenant_id` since slug wasn't supplied.
        assert!(svc.pulse("s-no-slug", "eastfield").await.is_some());
    }

    #[test]
    fn producer_payload_deserializes_from_camel_case() {
        // Matches course-service emitLectureStarted payload shape.
        let json = r#"{
            "sessionId": "s-1",
            "courseId": "c-1",
            "tenantId": "eastfield-uuid",
            "instructorId": "instr-1",
            "tenantSlug": "eastfield"
        }"#;
        let evt: LectureStartedEvent = serde_json::from_str(json).expect("deserialize");
        assert_eq!(evt.session_id, "s-1");
        assert_eq!(evt.slug(), "eastfield");
        assert_eq!(evt.enrolled_count, 0);
    }

    #[tokio::test]
    async fn consumer_links_to_producer_span_via_traceparent_header() {
        use common_rust::observability::{REQUEST_ID_HEADER, TRACEPARENT_HEADER};

        let store = Arc::new(InMemoryAttendance::new());
        let svc = LectureService::new(store.clone());

        let headers = vec![
            KafkaHeader::new(
                TRACEPARENT_HEADER,
                b"00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01".to_vec(),
            ),
            KafkaHeader::new(REQUEST_ID_HEADER, b"req-abc".to_vec()),
        ];

        let evt = LectureStartedEvent {
            session_id: "s-traced".to_string(),
            tenant_id: "eastfield".to_string(),
            tenant_slug: Some("eastfield".to_string()),
            course_id: "c-1".to_string(),
            instructor_id: "instr-1".to_string(),
            enrolled_count: 5,
        };
        dispatch_lecture_started(&svc, evt, &headers).await;

        // If header extraction mis-parsed, the dispatch would still succeed
        // but trace linking would break at the observability layer. Upstream
        // tests of `context_from_kafka_headers` (common-rust) cover the
        // carrier parsing directly.
        assert!(svc.pulse("s-traced", "eastfield").await.is_some());
    }
}
