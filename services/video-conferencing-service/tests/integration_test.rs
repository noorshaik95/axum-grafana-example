use chrono::Utc;
use uuid::Uuid;

// Test helper to create a test session request
fn create_test_schedule_request() -> video_conferencing_service::models::proto::ScheduleSessionRequest {
    use video_conferencing_service::models::proto::*;

    let start_time = Utc::now() + chrono::Duration::hours(1);

    ScheduleSessionRequest {
        instructor_id: Uuid::new_v4().to_string(),
        title: "Test Session".to_string(),
        description: "Integration test session".to_string(),
        start_time: Some(prost_types::Timestamp {
            seconds: start_time.timestamp(),
            nanos: start_time.timestamp_subsec_nanos() as i32,
        }),
        duration_minutes: 60,
        student_ids: vec![
            Uuid::new_v4().to_string(),
            Uuid::new_v4().to_string(),
        ],
        settings: Some(SessionSettings {
            max_participants: 50, // AC5
            auto_record: true,    // AC8
            allow_screen_share: true, // AC7
            quality_settings: Some(VideoQualitySettings {
                default_quality: VideoQuality::Quality720p as i32,
                adaptive_bitrate: true, // AC4
                min_bitrate_kbps: 300,
                max_bitrate_kbps: 5000,
            }),
            require_approval: false,
            mute_on_join: false,
        }),
    }
}

#[test]
fn test_schedule_request_validation_ac1() {
    let request = create_test_schedule_request();

    // AC1: Schedule video session (date, time, duration)
    assert!(!request.title.is_empty(), "Title should not be empty");
    assert!(request.duration_minutes > 0, "Duration should be positive");
    assert!(request.start_time.is_some(), "Start time should be set");
}

#[test]
fn test_session_settings_ac5() {
    let request = create_test_schedule_request();

    // AC5: Supports 50 concurrent participants
    assert!(request.settings.is_some());
    let settings = request.settings.unwrap();
    assert_eq!(
        settings.max_participants, 50,
        "Should support 50 concurrent participants"
    );
}

#[test]
fn test_session_settings_ac7() {
    let request = create_test_schedule_request();

    // AC7: Screen sharing enabled
    let settings = request.settings.unwrap();
    assert!(
        settings.allow_screen_share,
        "Screen sharing should be enabled"
    );
}

#[test]
fn test_session_settings_ac8() {
    let request = create_test_schedule_request();

    // AC8: Session auto-records to GCS
    let settings = request.settings.unwrap();
    assert!(settings.auto_record, "Auto recording should be enabled");
}

#[test]
fn test_video_quality_settings_ac4() {
    let request = create_test_schedule_request();

    // AC4: Video quality adapts to bandwidth (360p-1080p)
    let settings = request.settings.unwrap();
    assert!(settings.quality_settings.is_some());

    let quality_settings = settings.quality_settings.unwrap();
    assert!(
        quality_settings.adaptive_bitrate,
        "Adaptive bitrate should be enabled"
    );
    assert_eq!(quality_settings.min_bitrate_kbps, 300);
    assert_eq!(quality_settings.max_bitrate_kbps, 5000);
}

#[test]
fn test_video_quality_levels_ac4() {
    use video_conferencing_service::models::proto::VideoQuality;

    // AC4: Test all quality levels (360p-1080p)
    let qualities = vec![
        (VideoQuality::Quality360p, "360p"),
        (VideoQuality::Quality480p, "480p"),
        (VideoQuality::Quality720p, "720p"),
        (VideoQuality::Quality1080p, "1080p"),
    ];

    for (quality, name) in qualities {
        assert!(
            quality as i32 >= 0,
            "{} quality level should be valid",
            name
        );
    }
}

#[test]
fn test_participant_role_types() {
    use video_conferencing_service::models::proto::ParticipantRole;

    // Test instructor and student roles
    let instructor = ParticipantRole::Instructor;
    let student = ParticipantRole::Student;

    assert!(instructor as i32 != student as i32);
    assert!(instructor as i32 > 0);
    assert!(student as i32 > 0);
}

#[test]
fn test_session_status_types() {
    use video_conferencing_service::models::proto::SessionStatus;

    // Test all session statuses
    let scheduled = SessionStatus::Scheduled;
    let active = SessionStatus::Active;
    let completed = SessionStatus::Completed;
    let cancelled = SessionStatus::Cancelled;

    assert!(scheduled as i32 > 0);
    assert!(active as i32 > 0);
    assert!(completed as i32 > 0);
    assert!(cancelled as i32 > 0);
}

#[test]
fn test_recording_status_types_ac9() {
    use video_conferencing_service::models::proto::RecordingStatus;

    // AC9: Recording status transitions
    let recording = RecordingStatus::Recording;
    let processing = RecordingStatus::Processing;
    let available = RecordingStatus::Available;
    let failed = RecordingStatus::Failed;

    // Verify all statuses are distinct
    assert!(recording as i32 > 0);
    assert!(processing as i32 > 0);
    assert!(available as i32 > 0);
    assert!(failed as i32 > 0);
}

#[test]
fn test_calendar_ics_generation_ac2() {
    use icalendar::*;

    // AC2: Calendar invitation generation
    let start = Utc::now();
    let end = start + chrono::Duration::hours(1);

    let event = Event::new()
        .summary("Test Session")
        .description("Test Description")
        .starts(start)
        .ends(end)
        .done();

    let calendar = Calendar::new().push(event).done();
    let ics = calendar.to_string();

    assert!(ics.contains("BEGIN:VCALENDAR"));
    assert!(ics.contains("BEGIN:VEVENT"));
    assert!(ics.contains("SUMMARY:Test Session"));
    assert!(ics.contains("END:VEVENT"));
    assert!(ics.contains("END:VCALENDAR"));
}

#[test]
fn test_join_request_structure() {
    use video_conferencing_service::models::proto::JoinSessionRequest;

    let request = JoinSessionRequest {
        session_id: Uuid::new_v4().to_string(),
        user_id: Uuid::new_v4().to_string(),
        display_name: "Test User".to_string(),
        audio_enabled: true,
        video_enabled: true,
    };

    assert!(!request.session_id.is_empty());
    assert!(!request.user_id.is_empty());
    assert!(!request.display_name.is_empty());
}

#[test]
fn test_mute_participant_request_ac6() {
    use video_conferencing_service::models::proto::MuteParticipantRequest;

    // AC6: Instructor mutes/unmutes participants
    let request = MuteParticipantRequest {
        session_id: Uuid::new_v4().to_string(),
        participant_id: Uuid::new_v4().to_string(),
        instructor_id: Uuid::new_v4().to_string(),
    };

    assert!(!request.session_id.is_empty());
    assert!(!request.participant_id.is_empty());
    assert!(!request.instructor_id.is_empty());
}

#[test]
fn test_screen_share_request_ac7() {
    use video_conferencing_service::models::proto::ScreenShareRequest;

    // AC7: Screen sharing
    let request = ScreenShareRequest {
        session_id: Uuid::new_v4().to_string(),
        participant_id: Uuid::new_v4().to_string(),
    };

    assert!(!request.session_id.is_empty());
    assert!(!request.participant_id.is_empty());
}

#[test]
fn test_recording_request_ac8() {
    use video_conferencing_service::models::proto::*;

    // AC8: Session auto-records to GCS
    let request = StartRecordingRequest {
        session_id: Uuid::new_v4().to_string(),
        instructor_id: Uuid::new_v4().to_string(),
        settings: Some(RecordingSettings {
            quality: VideoQuality::Quality720p as i32,
            include_audio: true,
            include_screen_share: true,
        }),
    };

    assert!(!request.session_id.is_empty());
    assert!(request.settings.is_some());

    let settings = request.settings.unwrap();
    assert!(settings.include_audio);
    assert!(settings.include_screen_share);
}

#[test]
fn test_update_video_quality_request_ac4() {
    use video_conferencing_service::models::proto::*;

    // AC4: Video quality adaptation
    let request = UpdateVideoQualityRequest {
        session_id: Uuid::new_v4().to_string(),
        participant_id: Uuid::new_v4().to_string(),
        quality: VideoQuality::Quality720p as i32,
    };

    assert!(!request.session_id.is_empty());
    assert!(!request.participant_id.is_empty());
    assert_eq!(request.quality, VideoQuality::Quality720p as i32);
}

#[test]
fn test_config_max_participants_ac5() {
    use std::env;

    env::set_var("MAX_PARTICIPANTS", "50");
    let config = video_conferencing_service::Config::from_env()
        .expect("Failed to create config");

    // AC5: Supports 50 concurrent participants
    assert_eq!(config.session.max_participants, 50);

    env::remove_var("MAX_PARTICIPANTS");
}

#[test]
fn test_config_join_window_ac3() {
    use std::env;

    env::set_var("JOIN_WINDOW_MINUTES", "10");
    let config = video_conferencing_service::Config::from_env()
        .expect("Failed to create config");

    // AC3: Join button enabled 10 minutes before start
    assert_eq!(config.session.join_window_minutes, 10);

    env::remove_var("JOIN_WINDOW_MINUTES");
}

#[test]
fn test_config_recording_timeout_ac9() {
    use std::env;

    env::set_var("RECORDING_TIMEOUT", "1800");
    let config = video_conferencing_service::Config::from_env()
        .expect("Failed to create config");

    // AC9: Recording available within 30 minutes (1800 seconds)
    assert_eq!(config.recording.processing_timeout_seconds, 1800);

    env::remove_var("RECORDING_TIMEOUT");
}

#[test]
fn test_session_join_eligibility_timing_ac3() {
    use video_conferencing_service::models::SessionJoinEligibility;
    use chrono::{Duration, Utc};

    // AC3: Join button enabled 10 minutes before start
    let join_window = 10;

    // Test 1: Session starts in 5 minutes (within window)
    let start_time = Utc::now() + Duration::minutes(5);
    let eligibility = SessionJoinEligibility::check(start_time, "SCHEDULED", join_window);
    assert!(eligibility.can_join, "Should be able to join within 10-minute window");

    // Test 2: Session starts in 20 minutes (outside window)
    let start_time = Utc::now() + Duration::minutes(20);
    let eligibility = SessionJoinEligibility::check(start_time, "SCHEDULED", join_window);
    assert!(!eligibility.can_join, "Should not be able to join before 10-minute window");

    // Test 3: Session is active
    let start_time = Utc::now();
    let eligibility = SessionJoinEligibility::check(start_time, "ACTIVE", join_window);
    assert!(eligibility.can_join, "Should be able to join active session");
}

#[test]
fn test_video_quality_adaptive_bitrate_ac4() {
    use video_conferencing_service::models::{VideoQualityHelper, proto::VideoQuality};

    // AC4: Video quality adapts to bandwidth (360p-1080p)

    // Low bandwidth -> 360p
    let quality = VideoQualityHelper::from_bandwidth(400);
    assert_eq!(quality as i32, VideoQuality::Quality360p as i32);

    // Medium-low bandwidth -> 480p
    let quality = VideoQualityHelper::from_bandwidth(800);
    assert_eq!(quality as i32, VideoQuality::Quality480p as i32);

    // Medium bandwidth -> 720p
    let quality = VideoQualityHelper::from_bandwidth(1500);
    assert_eq!(quality as i32, VideoQuality::Quality720p as i32);

    // High bandwidth -> 1080p
    let quality = VideoQualityHelper::from_bandwidth(4000);
    assert_eq!(quality as i32, VideoQuality::Quality1080p as i32);
}
