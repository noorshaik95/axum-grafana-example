use once_cell::sync::Lazy;
use prometheus::{
    Counter, Gauge, Histogram, HistogramOpts, IntCounter, IntGauge, Opts, Registry,
};
use std::sync::Arc;

pub static METRICS: Lazy<Arc<Metrics>> = Lazy::new(|| Arc::new(Metrics::new()));

pub struct Metrics {
    pub registry: Registry,

    // Session metrics
    pub sessions_scheduled_total: IntCounter,
    pub sessions_active: IntGauge,
    pub sessions_completed_total: IntCounter,
    pub sessions_cancelled_total: IntCounter,

    // Participant metrics
    pub participants_joined_total: IntCounter,
    pub participants_left_total: IntCounter,
    pub participants_active: IntGauge,
    pub participants_kicked_total: IntCounter,

    // Video quality metrics (AC4)
    pub video_quality_switches_total: IntCounter,
    pub bitrate_kbps: Gauge,
    pub packet_loss_percentage: Gauge,
    pub frame_rate: Gauge,
    pub latency_ms: Histogram,

    // Recording metrics (AC8, AC9)
    pub recordings_started_total: IntCounter,
    pub recordings_completed_total: IntCounter,
    pub recordings_failed_total: IntCounter,
    pub recording_processing_duration_seconds: Histogram,
    pub recording_file_size_bytes: Histogram,

    // Screen sharing metrics (AC7)
    pub screen_shares_started_total: IntCounter,
    pub screen_shares_stopped_total: IntCounter,
    pub screen_shares_active: IntGauge,

    // Mute/unmute metrics (AC6)
    pub participants_muted_total: IntCounter,
    pub participants_unmuted_total: IntCounter,

    // WebRTC metrics
    pub webrtc_connections_total: IntCounter,
    pub webrtc_connections_failed_total: IntCounter,
    pub webrtc_ice_candidates_total: IntCounter,

    // Calendar invitation metrics (AC2)
    pub calendar_invitations_sent_total: IntCounter,
    pub calendar_invitations_failed_total: IntCounter,

    // gRPC metrics
    pub grpc_requests_total: Counter,
    pub grpc_request_duration_seconds: Histogram,
    pub grpc_errors_total: Counter,

    // Database metrics
    pub db_queries_total: IntCounter,
    pub db_query_duration_seconds: Histogram,
    pub db_errors_total: IntCounter,
}

impl Metrics {
    pub fn new() -> Self {
        let registry = Registry::new();

        // Session metrics
        let sessions_scheduled_total = IntCounter::with_opts(
            Opts::new("sessions_scheduled_total", "Total number of sessions scheduled")
        ).unwrap();

        let sessions_active = IntGauge::with_opts(
            Opts::new("sessions_active", "Number of currently active sessions")
        ).unwrap();

        let sessions_completed_total = IntCounter::with_opts(
            Opts::new("sessions_completed_total", "Total number of completed sessions")
        ).unwrap();

        let sessions_cancelled_total = IntCounter::with_opts(
            Opts::new("sessions_cancelled_total", "Total number of cancelled sessions")
        ).unwrap();

        // Participant metrics
        let participants_joined_total = IntCounter::with_opts(
            Opts::new("participants_joined_total", "Total number of participants joined")
        ).unwrap();

        let participants_left_total = IntCounter::with_opts(
            Opts::new("participants_left_total", "Total number of participants left")
        ).unwrap();

        let participants_active = IntGauge::with_opts(
            Opts::new("participants_active", "Number of currently active participants")
        ).unwrap();

        let participants_kicked_total = IntCounter::with_opts(
            Opts::new("participants_kicked_total", "Total number of participants kicked")
        ).unwrap();

        // Video quality metrics
        let video_quality_switches_total = IntCounter::with_opts(
            Opts::new("video_quality_switches_total", "Total number of video quality switches (AC4)")
        ).unwrap();

        let bitrate_kbps = Gauge::with_opts(
            Opts::new("bitrate_kbps", "Current bitrate in kbps (AC4)")
        ).unwrap();

        let packet_loss_percentage = Gauge::with_opts(
            Opts::new("packet_loss_percentage", "Current packet loss percentage")
        ).unwrap();

        let frame_rate = Gauge::with_opts(
            Opts::new("frame_rate", "Current frame rate")
        ).unwrap();

        let latency_ms = Histogram::with_opts(
            HistogramOpts::new("latency_ms", "Latency in milliseconds")
                .buckets(vec![10.0, 25.0, 50.0, 100.0, 250.0, 500.0, 1000.0])
        ).unwrap();

        // Recording metrics
        let recordings_started_total = IntCounter::with_opts(
            Opts::new("recordings_started_total", "Total number of recordings started (AC8)")
        ).unwrap();

        let recordings_completed_total = IntCounter::with_opts(
            Opts::new("recordings_completed_total", "Total number of recordings completed (AC8)")
        ).unwrap();

        let recordings_failed_total = IntCounter::with_opts(
            Opts::new("recordings_failed_total", "Total number of recordings failed")
        ).unwrap();

        let recording_processing_duration_seconds = Histogram::with_opts(
            HistogramOpts::new(
                "recording_processing_duration_seconds",
                "Time to process and upload recording (AC9: should be < 30 min)"
            ).buckets(vec![60.0, 300.0, 600.0, 900.0, 1200.0, 1500.0, 1800.0])
        ).unwrap();

        let recording_file_size_bytes = Histogram::with_opts(
            HistogramOpts::new("recording_file_size_bytes", "Recording file size in bytes")
                .buckets(prometheus::exponential_buckets(1024.0 * 1024.0, 2.0, 10).unwrap())
        ).unwrap();

        // Screen sharing metrics
        let screen_shares_started_total = IntCounter::with_opts(
            Opts::new("screen_shares_started_total", "Total screen shares started (AC7)")
        ).unwrap();

        let screen_shares_stopped_total = IntCounter::with_opts(
            Opts::new("screen_shares_stopped_total", "Total screen shares stopped (AC7)")
        ).unwrap();

        let screen_shares_active = IntGauge::with_opts(
            Opts::new("screen_shares_active", "Number of active screen shares (AC7)")
        ).unwrap();

        // Mute/unmute metrics
        let participants_muted_total = IntCounter::with_opts(
            Opts::new("participants_muted_total", "Total participants muted by instructor (AC6)")
        ).unwrap();

        let participants_unmuted_total = IntCounter::with_opts(
            Opts::new("participants_unmuted_total", "Total participants unmuted by instructor (AC6)")
        ).unwrap();

        // WebRTC metrics
        let webrtc_connections_total = IntCounter::with_opts(
            Opts::new("webrtc_connections_total", "Total WebRTC connections established")
        ).unwrap();

        let webrtc_connections_failed_total = IntCounter::with_opts(
            Opts::new("webrtc_connections_failed_total", "Total WebRTC connections failed")
        ).unwrap();

        let webrtc_ice_candidates_total = IntCounter::with_opts(
            Opts::new("webrtc_ice_candidates_total", "Total ICE candidates exchanged")
        ).unwrap();

        // Calendar invitation metrics
        let calendar_invitations_sent_total = IntCounter::with_opts(
            Opts::new("calendar_invitations_sent_total", "Total calendar invitations sent (AC2)")
        ).unwrap();

        let calendar_invitations_failed_total = IntCounter::with_opts(
            Opts::new("calendar_invitations_failed_total", "Total calendar invitations failed (AC2)")
        ).unwrap();

        // gRPC metrics
        let grpc_requests_total = Counter::with_opts(
            Opts::new("grpc_requests_total", "Total gRPC requests")
        ).unwrap();

        let grpc_request_duration_seconds = Histogram::with_opts(
            HistogramOpts::new("grpc_request_duration_seconds", "gRPC request duration")
                .buckets(vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0])
        ).unwrap();

        let grpc_errors_total = Counter::with_opts(
            Opts::new("grpc_errors_total", "Total gRPC errors")
        ).unwrap();

        // Database metrics
        let db_queries_total = IntCounter::with_opts(
            Opts::new("db_queries_total", "Total database queries")
        ).unwrap();

        let db_query_duration_seconds = Histogram::with_opts(
            HistogramOpts::new("db_query_duration_seconds", "Database query duration")
                .buckets(vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0])
        ).unwrap();

        let db_errors_total = IntCounter::with_opts(
            Opts::new("db_errors_total", "Total database errors")
        ).unwrap();

        // Register all metrics
        registry.register(Box::new(sessions_scheduled_total.clone())).unwrap();
        registry.register(Box::new(sessions_active.clone())).unwrap();
        registry.register(Box::new(sessions_completed_total.clone())).unwrap();
        registry.register(Box::new(sessions_cancelled_total.clone())).unwrap();

        registry.register(Box::new(participants_joined_total.clone())).unwrap();
        registry.register(Box::new(participants_left_total.clone())).unwrap();
        registry.register(Box::new(participants_active.clone())).unwrap();
        registry.register(Box::new(participants_kicked_total.clone())).unwrap();

        registry.register(Box::new(video_quality_switches_total.clone())).unwrap();
        registry.register(Box::new(bitrate_kbps.clone())).unwrap();
        registry.register(Box::new(packet_loss_percentage.clone())).unwrap();
        registry.register(Box::new(frame_rate.clone())).unwrap();
        registry.register(Box::new(latency_ms.clone())).unwrap();

        registry.register(Box::new(recordings_started_total.clone())).unwrap();
        registry.register(Box::new(recordings_completed_total.clone())).unwrap();
        registry.register(Box::new(recordings_failed_total.clone())).unwrap();
        registry.register(Box::new(recording_processing_duration_seconds.clone())).unwrap();
        registry.register(Box::new(recording_file_size_bytes.clone())).unwrap();

        registry.register(Box::new(screen_shares_started_total.clone())).unwrap();
        registry.register(Box::new(screen_shares_stopped_total.clone())).unwrap();
        registry.register(Box::new(screen_shares_active.clone())).unwrap();

        registry.register(Box::new(participants_muted_total.clone())).unwrap();
        registry.register(Box::new(participants_unmuted_total.clone())).unwrap();

        registry.register(Box::new(webrtc_connections_total.clone())).unwrap();
        registry.register(Box::new(webrtc_connections_failed_total.clone())).unwrap();
        registry.register(Box::new(webrtc_ice_candidates_total.clone())).unwrap();

        registry.register(Box::new(calendar_invitations_sent_total.clone())).unwrap();
        registry.register(Box::new(calendar_invitations_failed_total.clone())).unwrap();

        registry.register(Box::new(grpc_requests_total.clone())).unwrap();
        registry.register(Box::new(grpc_request_duration_seconds.clone())).unwrap();
        registry.register(Box::new(grpc_errors_total.clone())).unwrap();

        registry.register(Box::new(db_queries_total.clone())).unwrap();
        registry.register(Box::new(db_query_duration_seconds.clone())).unwrap();
        registry.register(Box::new(db_errors_total.clone())).unwrap();

        Self {
            registry,
            sessions_scheduled_total,
            sessions_active,
            sessions_completed_total,
            sessions_cancelled_total,
            participants_joined_total,
            participants_left_total,
            participants_active,
            participants_kicked_total,
            video_quality_switches_total,
            bitrate_kbps,
            packet_loss_percentage,
            frame_rate,
            latency_ms,
            recordings_started_total,
            recordings_completed_total,
            recordings_failed_total,
            recording_processing_duration_seconds,
            recording_file_size_bytes,
            screen_shares_started_total,
            screen_shares_stopped_total,
            screen_shares_active,
            participants_muted_total,
            participants_unmuted_total,
            webrtc_connections_total,
            webrtc_connections_failed_total,
            webrtc_ice_candidates_total,
            calendar_invitations_sent_total,
            calendar_invitations_failed_total,
            grpc_requests_total,
            grpc_request_duration_seconds,
            grpc_errors_total,
            db_queries_total,
            db_query_duration_seconds,
            db_errors_total,
        }
    }
}

impl Default for Metrics {
    fn default() -> Self {
        Self::new()
    }
}
