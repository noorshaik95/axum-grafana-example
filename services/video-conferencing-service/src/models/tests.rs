#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_session_join_eligibility_scheduled_before_window() {
        use chrono::{Duration, Utc};

        let start_time = Utc::now() + Duration::hours(1);
        let eligibility = SessionJoinEligibility::check(start_time, "SCHEDULED", 10);

        assert!(!eligibility.can_join);
        assert!(eligibility.message.contains("minutes"));
        assert!(eligibility.join_enabled_at.is_some());
    }

    #[test]
    fn test_session_join_eligibility_within_window() {
        use chrono::{Duration, Utc};

        let start_time = Utc::now() + Duration::minutes(5); // 5 minutes from now
        let eligibility = SessionJoinEligibility::check(start_time, "SCHEDULED", 10);

        assert!(eligibility.can_join); // Within 10-minute window
        assert_eq!(eligibility.message, "Session is ready to join");
        assert!(eligibility.join_enabled_at.is_some());
    }

    #[test]
    fn test_session_join_eligibility_active() {
        use chrono::Utc;

        let start_time = Utc::now();
        let eligibility = SessionJoinEligibility::check(start_time, "ACTIVE", 10);

        assert!(eligibility.can_join);
        assert_eq!(eligibility.message, "Session is active and ready to join");
    }

    #[test]
    fn test_session_join_eligibility_completed() {
        use chrono::Utc;

        let start_time = Utc::now();
        let eligibility = SessionJoinEligibility::check(start_time, "COMPLETED", 10);

        assert!(!eligibility.can_join);
        assert_eq!(eligibility.message, "Session has already completed");
    }

    #[test]
    fn test_session_join_eligibility_cancelled() {
        use chrono::Utc;

        let start_time = Utc::now();
        let eligibility = SessionJoinEligibility::check(start_time, "CANCELLED", 10);

        assert!(!eligibility.can_join);
        assert_eq!(eligibility.message, "Session has been cancelled");
    }

    #[test]
    fn test_video_quality_from_bandwidth_360p() {
        let quality = VideoQualityHelper::from_bandwidth(400);
        assert_eq!(quality as i32, VideoQuality::Quality360p as i32);
    }

    #[test]
    fn test_video_quality_from_bandwidth_480p() {
        let quality = VideoQualityHelper::from_bandwidth(800);
        assert_eq!(quality as i32, VideoQuality::Quality480p as i32);
    }

    #[test]
    fn test_video_quality_from_bandwidth_720p() {
        let quality = VideoQualityHelper::from_bandwidth(2000);
        assert_eq!(quality as i32, VideoQuality::Quality720p as i32);
    }

    #[test]
    fn test_video_quality_from_bandwidth_1080p() {
        let quality = VideoQualityHelper::from_bandwidth(5000);
        assert_eq!(quality as i32, VideoQuality::Quality1080p as i32);
    }

    #[test]
    fn test_video_quality_bitrate_ranges() {
        let (min, max) = VideoQualityHelper::get_bitrate_range(VideoQuality::Quality360p);
        assert_eq!(min, 300);
        assert_eq!(max, 800);

        let (min, max) = VideoQualityHelper::get_bitrate_range(VideoQuality::Quality480p);
        assert_eq!(min, 500);
        assert_eq!(max, 1200);

        let (min, max) = VideoQualityHelper::get_bitrate_range(VideoQuality::Quality720p);
        assert_eq!(min, 1000);
        assert_eq!(max, 2500);

        let (min, max) = VideoQualityHelper::get_bitrate_range(VideoQuality::Quality1080p);
        assert_eq!(min, 2500);
        assert_eq!(max, 5000);
    }

    #[test]
    fn test_video_quality_string_conversion() {
        let quality_str = VideoQualityHelper::quality_to_string(VideoQuality::Quality720p);
        assert_eq!(quality_str, "QUALITY_720P");

        let quality = VideoQualityHelper::string_to_quality("QUALITY_720P");
        assert_eq!(quality as i32, VideoQuality::Quality720p as i32);
    }

    #[test]
    fn test_video_quality_string_conversion_round_trip() {
        let original = VideoQuality::Quality1080p;
        let as_string = VideoQualityHelper::quality_to_string(original);
        let back_to_enum = VideoQualityHelper::string_to_quality(&as_string);
        assert_eq!(original as i32, back_to_enum as i32);
    }

    #[test]
    fn test_video_quality_default_for_invalid_string() {
        let quality = VideoQualityHelper::string_to_quality("INVALID");
        assert_eq!(quality as i32, VideoQuality::Quality720p as i32); // Should default to 720p
    }

    #[test]
    fn test_video_quality_adaptive_bandwidth_boundaries() {
        // Test boundary conditions
        assert_eq!(
            VideoQualityHelper::from_bandwidth(500) as i32,
            VideoQuality::Quality360p as i32
        );
        assert_eq!(
            VideoQualityHelper::from_bandwidth(501) as i32,
            VideoQuality::Quality480p as i32
        );
        assert_eq!(
            VideoQualityHelper::from_bandwidth(1000) as i32,
            VideoQuality::Quality480p as i32
        );
        assert_eq!(
            VideoQualityHelper::from_bandwidth(1001) as i32,
            VideoQuality::Quality720p as i32
        );
        assert_eq!(
            VideoQualityHelper::from_bandwidth(2500) as i32,
            VideoQuality::Quality720p as i32
        );
        assert_eq!(
            VideoQualityHelper::from_bandwidth(2501) as i32,
            VideoQuality::Quality1080p as i32
        );
    }
}
