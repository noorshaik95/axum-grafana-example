#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_signaling_message_join_serialization() {
        let msg = SignalingMessage::Join {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
            display_name: "John Doe".to_string(),
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("Join"));
        assert!(json.contains("session-1"));
        assert!(json.contains("participant-1"));
        assert!(json.contains("John Doe"));
    }

    #[test]
    fn test_signaling_message_leave_serialization() {
        let msg = SignalingMessage::Leave {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("Leave"));
        assert!(json.contains("session-1"));
    }

    #[test]
    fn test_signaling_message_offer_serialization() {
        let msg = SignalingMessage::Offer {
            session_id: "session-1".to_string(),
            from: "participant-1".to_string(),
            to: "participant-2".to_string(),
            sdp: "v=0\r\no=- 123 456 IN IP4 127.0.0.1\r\n".to_string(),
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("Offer"));
        assert!(json.contains("participant-1"));
        assert!(json.contains("participant-2"));
        assert!(json.contains("v=0"));
    }

    #[test]
    fn test_signaling_message_answer_serialization() {
        let msg = SignalingMessage::Answer {
            session_id: "session-1".to_string(),
            from: "participant-2".to_string(),
            to: "participant-1".to_string(),
            sdp: "v=0\r\no=- 789 012 IN IP4 127.0.0.1\r\n".to_string(),
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("Answer"));
    }

    #[test]
    fn test_signaling_message_ice_candidate_serialization() {
        let msg = SignalingMessage::IceCandidate {
            session_id: "session-1".to_string(),
            from: "participant-1".to_string(),
            to: "participant-2".to_string(),
            candidate: "candidate:1 1 UDP 2130706431 192.168.1.1 54321 typ host".to_string(),
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("IceCandidate"));
        assert!(json.contains("candidate:1"));
    }

    #[test]
    fn test_signaling_message_screen_share_ac7() {
        let start_msg = SignalingMessage::StartScreenShare {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
        };

        let json = serde_json::to_string(&start_msg).expect("Failed to serialize");
        assert!(json.contains("StartScreenShare"));

        let stop_msg = SignalingMessage::StopScreenShare {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
        };

        let json = serde_json::to_string(&stop_msg).expect("Failed to serialize");
        assert!(json.contains("StopScreenShare"));
    }

    #[test]
    fn test_signaling_message_quality_change_ac4() {
        let msg = SignalingMessage::QualityChange {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
            quality: "QUALITY_720P".to_string(),
            bitrate: 2500,
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("QualityChange"));
        assert!(json.contains("QUALITY_720P"));
        assert!(json.contains("2500"));
    }

    #[test]
    fn test_signaling_message_stats() {
        let msg = SignalingMessage::Stats {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
            bitrate_kbps: 2500,
            packet_loss: 0.5,
            jitter_ms: 10,
            latency_ms: 50,
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("Stats"));
        assert!(json.contains("2500"));
    }

    #[test]
    fn test_signaling_message_mute_audio_ac6() {
        let msg = SignalingMessage::MuteAudio {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
            muted: true,
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("MuteAudio"));
        assert!(json.contains("true"));
    }

    #[test]
    fn test_signaling_message_mute_video_ac6() {
        let msg = SignalingMessage::MuteVideo {
            session_id: "session-1".to_string(),
            participant_id: "participant-1".to_string(),
            muted: false,
        };

        let json = serde_json::to_string(&msg).expect("Failed to serialize");
        assert!(json.contains("MuteVideo"));
        assert!(json.contains("false"));
    }

    #[test]
    fn test_participant_info_serialization() {
        let info = ParticipantInfo {
            participant_id: "participant-1".to_string(),
            display_name: "John Doe".to_string(),
            audio_muted: false,
            video_muted: false,
            is_screen_sharing: true, // AC7
        };

        let json = serde_json::to_string(&info).expect("Failed to serialize");
        assert!(json.contains("participant-1"));
        assert!(json.contains("John Doe"));
        assert!(json.contains("is_screen_sharing"));
    }

    #[test]
    fn test_signaling_server_creation() {
        let server = SignalingServer::new();
        assert_eq!(server.sessions.len(), 0);
    }

    #[test]
    fn test_signaling_message_deserialization() {
        let json = r#"{
            "type": "Join",
            "session_id": "session-1",
            "participant_id": "participant-1",
            "display_name": "John Doe"
        }"#;

        let msg: SignalingMessage =
            serde_json::from_str(json).expect("Failed to deserialize");

        match msg {
            SignalingMessage::Join {
                session_id,
                participant_id,
                display_name,
            } => {
                assert_eq!(session_id, "session-1");
                assert_eq!(participant_id, "participant-1");
                assert_eq!(display_name, "John Doe");
            }
            _ => panic!("Wrong message type"),
        }
    }

    #[test]
    fn test_signaling_message_round_trip() {
        let original = SignalingMessage::Offer {
            session_id: "session-1".to_string(),
            from: "p1".to_string(),
            to: "p2".to_string(),
            sdp: "v=0\r\n".to_string(),
        };

        let json = serde_json::to_string(&original).expect("Failed to serialize");
        let deserialized: SignalingMessage =
            serde_json::from_str(&json).expect("Failed to deserialize");

        match (original, deserialized) {
            (
                SignalingMessage::Offer {
                    session_id: s1,
                    from: f1,
                    to: t1,
                    sdp: sdp1,
                },
                SignalingMessage::Offer {
                    session_id: s2,
                    from: f2,
                    to: t2,
                    sdp: sdp2,
                },
            ) => {
                assert_eq!(s1, s2);
                assert_eq!(f1, f2);
                assert_eq!(t1, t2);
                assert_eq!(sdp1, sdp2);
            }
            _ => panic!("Round trip failed"),
        }
    }
}
