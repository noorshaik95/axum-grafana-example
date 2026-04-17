#[cfg(test)]
mod tests {
    use super::*;
    use std::env;

    #[test]
    fn test_config_defaults() {
        // Clear environment variables
        env::remove_var("SERVER_HOST");
        env::remove_var("GRPC_PORT");
        env::remove_var("MAX_PARTICIPANTS");

        let config = Config::from_env().expect("Failed to create config");

        // Test defaults
        assert_eq!(config.server.host, "0.0.0.0");
        assert_eq!(config.server.grpc_port, 50052);
        assert_eq!(config.server.ws_port, 8082);
        assert_eq!(config.session.max_participants, 50); // AC5
        assert_eq!(config.session.join_window_minutes, 10); // AC3
        assert_eq!(config.recording.processing_timeout_seconds, 1800); // AC9: 30 minutes
    }

    #[test]
    fn test_config_from_env() {
        env::set_var("SERVER_HOST", "127.0.0.1");
        env::set_var("GRPC_PORT", "50053");
        env::set_var("WS_PORT", "8083");
        env::set_var("MAX_PARTICIPANTS", "100");
        env::set_var("JOIN_WINDOW_MINUTES", "15");

        let config = Config::from_env().expect("Failed to create config");

        assert_eq!(config.server.host, "127.0.0.1");
        assert_eq!(config.server.grpc_port, 50053);
        assert_eq!(config.server.ws_port, 8083);
        assert_eq!(config.session.max_participants, 100);
        assert_eq!(config.session.join_window_minutes, 15);

        // Cleanup
        env::remove_var("SERVER_HOST");
        env::remove_var("GRPC_PORT");
        env::remove_var("WS_PORT");
        env::remove_var("MAX_PARTICIPANTS");
        env::remove_var("JOIN_WINDOW_MINUTES");
    }

    #[test]
    fn test_database_url_generation() {
        let config = Config::from_env().expect("Failed to create config");

        let url = config.database_url();

        assert!(url.contains("postgresql://"));
        assert!(url.contains(&config.database.user));
        assert!(url.contains(&config.database.host));
        assert!(url.contains(&config.database.port.to_string()));
        assert!(url.contains(&config.database.name));
    }

    #[test]
    fn test_recording_config_ac8_ac9() {
        env::set_var("RECORDING_ENABLED", "true");
        env::set_var("GCS_BUCKET", "test-bucket");
        env::set_var("RECORDING_TIMEOUT", "1800"); // AC9: 30 minutes

        let config = Config::from_env().expect("Failed to create config");

        assert!(config.recording.enabled); // AC8
        assert_eq!(config.recording.gcs_bucket, "test-bucket");
        assert_eq!(config.recording.processing_timeout_seconds, 1800); // AC9

        // Cleanup
        env::remove_var("RECORDING_ENABLED");
        env::remove_var("GCS_BUCKET");
        env::remove_var("RECORDING_TIMEOUT");
    }

    #[test]
    fn test_webrtc_config() {
        env::set_var("WEBRTC_PORT_MIN", "40000");
        env::set_var("WEBRTC_PORT_MAX", "40100");

        let config = Config::from_env().expect("Failed to create config");

        assert_eq!(config.webrtc.port_range_min, 40000);
        assert_eq!(config.webrtc.port_range_max, 40100);
        assert!(!config.webrtc.ice_servers.is_empty());

        // Cleanup
        env::remove_var("WEBRTC_PORT_MIN");
        env::remove_var("WEBRTC_PORT_MAX");
    }

    #[test]
    fn test_session_config_ac3_ac5() {
        env::set_var("MAX_PARTICIPANTS", "50"); // AC5
        env::set_var("JOIN_WINDOW_MINUTES", "10"); // AC3

        let config = Config::from_env().expect("Failed to create config");

        assert_eq!(config.session.max_participants, 50); // AC5
        assert_eq!(config.session.join_window_minutes, 10); // AC3

        // Cleanup
        env::remove_var("MAX_PARTICIPANTS");
        env::remove_var("JOIN_WINDOW_MINUTES");
    }

    #[test]
    fn test_observability_config() {
        env::set_var("TEMPO_ENDPOINT", "http://tempo:4317");
        env::set_var("SERVICE_NAME", "video-conferencing-service");
        env::set_var("PROMETHEUS_PORT", "9092");

        let config = Config::from_env().expect("Failed to create config");

        assert_eq!(config.observability.tempo_endpoint, "http://tempo:4317");
        assert_eq!(
            config.observability.service_name,
            "video-conferencing-service"
        );
        assert_eq!(config.observability.prometheus_port, 9092);

        // Cleanup
        env::remove_var("TEMPO_ENDPOINT");
        env::remove_var("SERVICE_NAME");
        env::remove_var("PROMETHEUS_PORT");
    }

    #[test]
    fn test_ice_server_serialization() {
        let ice_server = IceServer {
            urls: vec!["stun:stun.l.google.com:19302".to_string()],
            username: Some("user".to_string()),
            credential: Some("pass".to_string()),
        };

        let json = serde_json::to_string(&ice_server).expect("Failed to serialize");
        assert!(json.contains("stun:stun.l.google.com:19302"));
        assert!(json.contains("user"));
        assert!(json.contains("pass"));
    }

    #[test]
    fn test_ice_server_without_credentials() {
        let ice_server = IceServer {
            urls: vec!["stun:stun.l.google.com:19302".to_string()],
            username: None,
            credential: None,
        };

        let json = serde_json::to_string(&ice_server).expect("Failed to serialize");
        assert!(json.contains("stun:stun.l.google.com:19302"));
        // Should not include username/credential fields when None
        assert!(!json.contains("username"));
        assert!(!json.contains("credential"));
    }
}
