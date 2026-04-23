//! W10.1 — unit tests for the video-position surface that don't require a
//! running Postgres/Redis.
//!
//! Covers:
//! - Redis key layout (`tenant:{slug}:video_pos:{user}:{content}`)
//! - TTL helper for video vs non-video content (W10.3)

use content_management_service::handlers::content_handlers::{
    video_pos_redis_key, VIDEO_POS_REDIS_TTL_SECS,
};
use content_management_service::storage::minio::{
    get_url_ttl, DEFAULT_URL_TTL_SECS, VIDEO_URL_TTL_SECS,
};
use std::time::Duration;
use uuid::Uuid;

#[test]
fn redis_key_shape_matches_w10_spec() {
    let user = Uuid::parse_str("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa").unwrap();
    let content = Uuid::parse_str("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb").unwrap();
    let key = video_pos_redis_key("eastfield", user, content);
    assert_eq!(
        key,
        "tenant:eastfield:video_pos:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
    );
}

#[test]
fn redis_ttl_is_thirty_minutes() {
    assert_eq!(VIDEO_POS_REDIS_TTL_SECS, 30 * 60);
}

#[test]
fn url_ttl_video_is_short() {
    assert_eq!(get_url_ttl("video/mp4"), Duration::from_secs(VIDEO_URL_TTL_SECS));
    assert_eq!(VIDEO_URL_TTL_SECS, 900);
}

#[test]
fn url_ttl_non_video_is_default() {
    assert_eq!(
        get_url_ttl("application/pdf"),
        Duration::from_secs(DEFAULT_URL_TTL_SECS)
    );
    assert_eq!(DEFAULT_URL_TTL_SECS, 3600);
}

#[test]
fn url_ttl_is_prefix_based() {
    // "videostream/..." must not be mistaken for "video/*".
    assert_eq!(
        get_url_ttl("videostream/foo"),
        Duration::from_secs(DEFAULT_URL_TTL_SECS)
    );
    // Only the video/ prefix qualifies.
    assert_eq!(
        get_url_ttl("video/quicktime"),
        Duration::from_secs(VIDEO_URL_TTL_SECS)
    );
}
