// Re-export proto types
pub mod proto {
    tonic::include_proto!("video_conferencing");
}

pub use proto::*;

// Additional helper types
use chrono::{DateTime, Duration, Utc};
use uuid::Uuid;

pub struct SessionJoinEligibility {
    pub can_join: bool,
    pub message: String,
    pub join_enabled_at: Option<DateTime<Utc>>,
}

impl SessionJoinEligibility {
    pub fn check(start_time: DateTime<Utc>, status: &str, join_window_minutes: i64) -> Self {
        let now = Utc::now();
        let join_enabled_at = start_time - Duration::minutes(join_window_minutes); // AC3: 10 minutes before

        match status {
            "SCHEDULED" => {
                if now >= join_enabled_at {
                    Self {
                        can_join: true,
                        message: "Session is ready to join".to_string(),
                        join_enabled_at: Some(join_enabled_at),
                    }
                } else {
                    let minutes_until = (join_enabled_at - now).num_minutes();
                    Self {
                        can_join: false,
                        message: format!("Session starts in {} minutes. Join will be enabled {} minutes before start.",
                            (start_time - now).num_minutes(), join_window_minutes),
                        join_enabled_at: Some(join_enabled_at),
                    }
                }
            }
            "ACTIVE" => Self {
                can_join: true,
                message: "Session is active and ready to join".to_string(),
                join_enabled_at: Some(join_enabled_at),
            },
            "COMPLETED" => Self {
                can_join: false,
                message: "Session has already completed".to_string(),
                join_enabled_at: Some(join_enabled_at),
            },
            "CANCELLED" => Self {
                can_join: false,
                message: "Session has been cancelled".to_string(),
                join_enabled_at: Some(join_enabled_at),
            },
            _ => Self {
                can_join: false,
                message: format!("Unknown session status: {}", status),
                join_enabled_at: None,
            },
        }
    }
}

// Video quality helpers (AC4: 360p-1080p adaptive bitrate)
pub struct VideoQualityHelper;

impl VideoQualityHelper {
    pub fn from_bandwidth(bandwidth_kbps: i32) -> VideoQuality {
        match bandwidth_kbps {
            0..=500 => VideoQuality::Quality360p,
            501..=1000 => VideoQuality::Quality480p,
            1001..=2500 => VideoQuality::Quality720p,
            _ => VideoQuality::Quality1080p,
        }
    }

    pub fn get_bitrate_range(quality: VideoQuality) -> (i32, i32) {
        match quality {
            VideoQuality::Quality360p => (300, 800),
            VideoQuality::Quality480p => (500, 1200),
            VideoQuality::Quality720p => (1000, 2500),
            VideoQuality::Quality1080p => (2500, 5000),
            _ => (500, 1200), // default to 480p
        }
    }

    pub fn quality_to_string(quality: VideoQuality) -> String {
        match quality {
            VideoQuality::Quality360p => "QUALITY_360P".to_string(),
            VideoQuality::Quality480p => "QUALITY_480P".to_string(),
            VideoQuality::Quality720p => "QUALITY_720P".to_string(),
            VideoQuality::Quality1080p => "QUALITY_1080P".to_string(),
            _ => "QUALITY_720P".to_string(), // default
        }
    }

    pub fn string_to_quality(s: &str) -> VideoQuality {
        match s {
            "QUALITY_360P" => VideoQuality::Quality360p,
            "QUALITY_480P" => VideoQuality::Quality480p,
            "QUALITY_720P" => VideoQuality::Quality720p,
            "QUALITY_1080P" => VideoQuality::Quality1080p,
            _ => VideoQuality::Quality720p, // default
        }
    }
}

#[cfg(test)]
mod tests;
