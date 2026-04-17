use axum::{
    extract::{
        ws::{Message, WebSocket},
        State, WebSocketUpgrade,
    },
    response::IntoResponse,
    routing::get,
    Router,
};
use dashmap::DashMap;
use futures::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::broadcast;
use uuid::Uuid;

// WebRTC signaling messages
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum SignalingMessage {
    // Session management
    Join {
        session_id: String,
        participant_id: String,
        display_name: String,
    },
    Leave {
        session_id: String,
        participant_id: String,
    },

    // WebRTC signaling (SDP and ICE)
    Offer {
        session_id: String,
        from: String,
        to: String,
        sdp: String,
    },
    Answer {
        session_id: String,
        from: String,
        to: String,
        sdp: String,
    },
    IceCandidate {
        session_id: String,
        from: String,
        to: String,
        candidate: String,
    },

    // Media control
    MuteAudio {
        session_id: String,
        participant_id: String,
        muted: bool,
    },
    MuteVideo {
        session_id: String,
        participant_id: String,
        muted: bool,
    },

    // Screen sharing (AC7)
    StartScreenShare {
        session_id: String,
        participant_id: String,
    },
    StopScreenShare {
        session_id: String,
        participant_id: String,
    },

    // Quality adaptation (AC4: 360p-1080p)
    QualityChange {
        session_id: String,
        participant_id: String,
        quality: String,
        bitrate: i32,
    },

    // Stats reporting
    Stats {
        session_id: String,
        participant_id: String,
        bitrate_kbps: i32,
        packet_loss: f32,
        jitter_ms: i32,
        latency_ms: i32,
    },

    // Responses
    Joined {
        session_id: String,
        participant_id: String,
        participants: Vec<ParticipantInfo>,
    },
    ParticipantJoined {
        session_id: String,
        participant: ParticipantInfo,
    },
    ParticipantLeft {
        session_id: String,
        participant_id: String,
    },
    Error {
        message: String,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ParticipantInfo {
    pub participant_id: String,
    pub display_name: String,
    pub audio_muted: bool,
    pub video_muted: bool,
    pub is_screen_sharing: bool,
}

// Session state
#[derive(Clone)]
struct SessionState {
    participants: Arc<DashMap<String, ParticipantInfo>>,
    tx: broadcast::Sender<SignalingMessage>,
}

pub struct SignalingServer {
    sessions: Arc<DashMap<String, SessionState>>,
}

impl SignalingServer {
    pub fn new() -> Self {
        Self {
            sessions: Arc::new(DashMap::new()),
        }
    }

    pub fn router(self) -> Router {
        let state = Arc::new(self);
        Router::new().route("/ws", get(ws_handler)).with_state(state)
    }

    fn get_or_create_session(&self, session_id: &str) -> SessionState {
        self.sessions
            .entry(session_id.to_string())
            .or_insert_with(|| {
                let (tx, _rx) = broadcast::channel(100);
                SessionState {
                    participants: Arc::new(DashMap::new()),
                    tx,
                }
            })
            .clone()
    }

    async fn handle_message(
        &self,
        msg: SignalingMessage,
        participant_id: &str,
    ) -> Option<SignalingMessage> {
        match msg {
            SignalingMessage::Join {
                session_id,
                participant_id: pid,
                display_name,
            } => {
                let session = self.get_or_create_session(&session_id);

                // Get current participants
                let participants: Vec<ParticipantInfo> = session
                    .participants
                    .iter()
                    .map(|entry| entry.value().clone())
                    .collect();

                // Add new participant
                let participant_info = ParticipantInfo {
                    participant_id: pid.clone(),
                    display_name: display_name.clone(),
                    audio_muted: false,
                    video_muted: false,
                    is_screen_sharing: false,
                };

                session
                    .participants
                    .insert(pid.clone(), participant_info.clone());

                // Notify others about new participant
                let _ = session.tx.send(SignalingMessage::ParticipantJoined {
                    session_id: session_id.clone(),
                    participant: participant_info,
                });

                tracing::info!(
                    "Participant {} joined session {}",
                    pid,
                    session_id
                );

                // Send joined response with current participants
                Some(SignalingMessage::Joined {
                    session_id,
                    participant_id: pid,
                    participants,
                })
            }

            SignalingMessage::Leave {
                session_id,
                participant_id: pid,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    session.participants.remove(&pid);

                    // Notify others
                    let _ = session.tx.send(SignalingMessage::ParticipantLeft {
                        session_id: session_id.clone(),
                        participant_id: pid.clone(),
                    });

                    tracing::info!(
                        "Participant {} left session {}",
                        pid,
                        session_id
                    );
                }
                None
            }

            // Forward WebRTC signaling messages to target participant
            SignalingMessage::Offer {
                session_id,
                from,
                to,
                sdp,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    let _ = session.tx.send(SignalingMessage::Offer {
                        session_id,
                        from,
                        to,
                        sdp,
                    });
                }
                None
            }

            SignalingMessage::Answer {
                session_id,
                from,
                to,
                sdp,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    let _ = session.tx.send(SignalingMessage::Answer {
                        session_id,
                        from,
                        to,
                        sdp,
                    });
                }
                None
            }

            SignalingMessage::IceCandidate {
                session_id,
                from,
                to,
                candidate,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    let _ = session.tx.send(SignalingMessage::IceCandidate {
                        session_id,
                        from,
                        to,
                        candidate,
                    });

                    crate::observability::METRICS.webrtc_ice_candidates_total.inc();
                }
                None
            }

            // Media control
            SignalingMessage::MuteAudio {
                session_id,
                participant_id: pid,
                muted,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    if let Some(mut participant) = session.participants.get_mut(&pid) {
                        participant.audio_muted = muted;
                    }

                    let _ = session.tx.send(SignalingMessage::MuteAudio {
                        session_id,
                        participant_id: pid,
                        muted,
                    });
                }
                None
            }

            SignalingMessage::MuteVideo {
                session_id,
                participant_id: pid,
                muted,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    if let Some(mut participant) = session.participants.get_mut(&pid) {
                        participant.video_muted = muted;
                    }

                    let _ = session.tx.send(SignalingMessage::MuteVideo {
                        session_id,
                        participant_id: pid,
                        muted,
                    });
                }
                None
            }

            // Screen sharing (AC7)
            SignalingMessage::StartScreenShare {
                session_id,
                participant_id: pid,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    if let Some(mut participant) = session.participants.get_mut(&pid) {
                        participant.is_screen_sharing = true;
                    }

                    let _ = session.tx.send(SignalingMessage::StartScreenShare {
                        session_id,
                        participant_id: pid,
                    });

                    crate::observability::METRICS.screen_shares_started_total.inc();
                    crate::observability::METRICS.screen_shares_active.inc();
                }
                None
            }

            SignalingMessage::StopScreenShare {
                session_id,
                participant_id: pid,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    if let Some(mut participant) = session.participants.get_mut(&pid) {
                        participant.is_screen_sharing = false;
                    }

                    let _ = session.tx.send(SignalingMessage::StopScreenShare {
                        session_id,
                        participant_id: pid,
                    });

                    crate::observability::METRICS.screen_shares_stopped_total.inc();
                    crate::observability::METRICS.screen_shares_active.dec();
                }
                None
            }

            // Quality adaptation (AC4)
            SignalingMessage::QualityChange {
                session_id,
                participant_id: pid,
                quality,
                bitrate,
            } => {
                if let Some(session) = self.sessions.get(&session_id) {
                    let _ = session.tx.send(SignalingMessage::QualityChange {
                        session_id,
                        participant_id: pid,
                        quality,
                        bitrate,
                    });

                    crate::observability::METRICS.video_quality_switches_total.inc();
                    crate::observability::METRICS
                        .bitrate_kbps
                        .set(bitrate as f64);
                }
                None
            }

            // Stats reporting
            SignalingMessage::Stats {
                session_id,
                participant_id: pid,
                bitrate_kbps,
                packet_loss,
                jitter_ms,
                latency_ms,
            } => {
                crate::observability::METRICS
                    .bitrate_kbps
                    .set(bitrate_kbps as f64);
                crate::observability::METRICS
                    .packet_loss_percentage
                    .set(packet_loss as f64);
                crate::observability::METRICS
                    .latency_ms
                    .observe(latency_ms as f64);

                tracing::debug!(
                    "Stats: session={}, participant={}, bitrate={}kbps, loss={}%, latency={}ms",
                    session_id,
                    pid,
                    bitrate_kbps,
                    packet_loss,
                    latency_ms
                );
                None
            }

            _ => None,
        }
    }
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    State(server): State<Arc<SignalingServer>>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handle_socket(socket, server))
}

async fn handle_socket(socket: WebSocket, server: Arc<SignalingServer>) {
    let (mut sender, mut receiver) = socket.split();
    let participant_id = Uuid::new_v4().to_string();

    let mut session_id: Option<String> = None;
    let mut rx: Option<broadcast::Receiver<SignalingMessage>> = None;

    tracing::info!("WebSocket connection established for participant {}", participant_id);

    loop {
        tokio::select! {
            // Receive from WebSocket client
            msg = receiver.next() => {
                match msg {
                    Some(Ok(Message::Text(text))) => {
                        match serde_json::from_str::<SignalingMessage>(&text) {
                            Ok(signaling_msg) => {
                                // Store session info on first join
                                if let SignalingMessage::Join { ref session_id: sid, .. } = signaling_msg {
                                    session_id = Some(sid.clone());
                                    let session = server.get_or_create_session(sid);
                                    rx = Some(session.tx.subscribe());
                                }

                                // Handle message
                                if let Some(response) = server.handle_message(signaling_msg, &participant_id).await {
                                    let response_text = serde_json::to_string(&response).unwrap();
                                    if sender.send(Message::Text(response_text)).await.is_err() {
                                        break;
                                    }
                                }
                            }
                            Err(e) => {
                                tracing::error!("Failed to parse signaling message: {}", e);
                                let error_msg = SignalingMessage::Error {
                                    message: format!("Invalid message format: {}", e),
                                };
                                let error_text = serde_json::to_string(&error_msg).unwrap();
                                let _ = sender.send(Message::Text(error_text)).await;
                            }
                        }
                    }
                    Some(Ok(Message::Close(_))) => {
                        tracing::info!("WebSocket closed by client: {}", participant_id);
                        break;
                    }
                    Some(Err(e)) => {
                        tracing::error!("WebSocket error: {}", e);
                        break;
                    }
                    None => break,
                    _ => {}
                }
            }

            // Broadcast to WebSocket client
            broadcast_msg = async {
                if let Some(ref mut receiver) = rx {
                    receiver.recv().await.ok()
                } else {
                    None
                }
            } => {
                if let Some(msg) = broadcast_msg {
                    let text = serde_json::to_string(&msg).unwrap();
                    if sender.send(Message::Text(text)).await.is_err() {
                        break;
                    }
                }
            }
        }
    }

    // Clean up on disconnect
    if let Some(sid) = session_id {
        let leave_msg = SignalingMessage::Leave {
            session_id: sid,
            participant_id: participant_id.clone(),
        };
        server.handle_message(leave_msg, &participant_id).await;
    }

    tracing::info!("WebSocket connection closed for participant {}", participant_id);
}

impl Default for SignalingServer {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests;
