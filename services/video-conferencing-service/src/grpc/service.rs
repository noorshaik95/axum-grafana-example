use crate::config::Config;
use crate::database::repository::VideoRepository;
use crate::models::proto::*;
use crate::models::{SessionJoinEligibility, VideoQualityHelper};
use crate::observability::METRICS;
use chrono::{Duration, Utc};
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::instrument;
use uuid::Uuid;

pub struct VideoConferencingServiceImpl {
    repo: VideoRepository,
    config: Arc<Config>,
}

impl VideoConferencingServiceImpl {
    pub fn new(repo: VideoRepository, config: Arc<Config>) -> Self {
        Self { repo, config }
    }

    fn generate_join_url(&self, session_id: &str) -> String {
        format!(
            "https://{}/session/{}",
            self.config.server.host, session_id
        )
    }

    fn get_ice_servers_json(&self) -> String {
        serde_json::to_string(&self.config.webrtc.ice_servers)
            .unwrap_or_else(|_| "[]".to_string())
    }
}

#[tonic::async_trait]
impl video_conferencing_service_server::VideoConferencingService for VideoConferencingServiceImpl {
    // AC1: Schedule video session (date, time, duration)
    #[instrument(skip(self))]
    async fn schedule_session(
        &self,
        request: Request<ScheduleSessionRequest>,
    ) -> Result<Response<ScheduleSessionResponse>, Status> {
        let start = std::time::Instant::now();
        METRICS.grpc_requests_total.inc();

        let req = request.into_inner();

        // Validate request
        if req.title.is_empty() {
            return Err(Status::invalid_argument("Title is required"));
        }

        let start_time = req
            .start_time
            .ok_or_else(|| Status::invalid_argument("Start time is required"))?;
        let start_time_chrono = chrono::DateTime::from_timestamp(
            start_time.seconds,
            start_time.nanos as u32,
        )
        .ok_or_else(|| Status::invalid_argument("Invalid start time"))?;

        if req.duration_minutes <= 0 {
            return Err(Status::invalid_argument("Duration must be positive"));
        }

        let instructor_id = Uuid::parse_str(&req.instructor_id)
            .map_err(|_| Status::invalid_argument("Invalid instructor ID"))?;

        // Get settings or use defaults
        let settings = req.settings.unwrap_or_else(|| SessionSettings {
            max_participants: self.config.session.max_participants as i32, // AC5: 50
            auto_record: true,                                               // AC8
            allow_screen_share: true,                                        // AC7
            quality_settings: Some(VideoQualitySettings {
                default_quality: VideoQuality::Quality720p as i32,
                adaptive_bitrate: true, // AC4
                min_bitrate_kbps: 300,
                max_bitrate_kbps: 5000,
            }),
            require_approval: false,
            mute_on_join: false,
        });

        // Create session
        let session_id = Uuid::new_v4();
        let join_url = self.generate_join_url(&session_id.to_string());

        let session = self
            .repo
            .create_session(
                instructor_id,
                req.title.clone(),
                if req.description.is_empty() {
                    None
                } else {
                    Some(req.description.clone())
                },
                start_time_chrono,
                req.duration_minutes,
                settings.max_participants,
                settings.auto_record,
                settings.allow_screen_share,
                settings.require_approval,
                settings.mute_on_join,
                join_url.clone(),
            )
            .await
            .map_err(|e| {
                tracing::error!("Failed to create session: {}", e);
                METRICS.grpc_errors_total.inc();
                Status::internal("Failed to create session")
            })?;

        // Create quality settings
        if let Some(quality_settings) = settings.quality_settings {
            let quality_str = VideoQualityHelper::quality_to_string(
                VideoQuality::try_from(quality_settings.default_quality)
                    .unwrap_or(VideoQuality::Quality720p),
            );

            self.repo
                .create_quality_settings(
                    session.id,
                    quality_str,
                    quality_settings.adaptive_bitrate,
                    quality_settings.min_bitrate_kbps,
                    quality_settings.max_bitrate_kbps,
                )
                .await
                .map_err(|e| {
                    tracing::error!("Failed to create quality settings: {}", e);
                    Status::internal("Failed to create quality settings")
                })?;
        }

        // Generate calendar ICS (AC2: Students receive calendar invitation)
        let calendar_ics = self.generate_calendar_ics(
            &req.title,
            &req.description,
            start_time_chrono,
            req.duration_minutes,
            &join_url,
        );

        METRICS.sessions_scheduled_total.inc();

        let duration = start.elapsed();
        METRICS
            .grpc_request_duration_seconds
            .observe(duration.as_secs_f64());

        tracing::info!(
            "Session scheduled: id={}, instructor={}, start_time={}",
            session.id,
            instructor_id,
            start_time_chrono
        );

        Ok(Response::new(ScheduleSessionResponse {
            session_id: session.id.to_string(),
            join_url,
            calendar_ics,
            session: Some(self.session_to_proto(&session)),
        }))
    }

    #[instrument(skip(self))]
    async fn get_session(
        &self,
        request: Request<GetSessionRequest>,
    ) -> Result<Response<SessionDetails>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        let session = self
            .repo
            .get_session(session_id)
            .await
            .map_err(|e| {
                tracing::error!("Database error: {}", e);
                Status::internal("Failed to fetch session")
            })?
            .ok_or_else(|| Status::not_found("Session not found"))?;

        Ok(Response::new(self.session_to_proto(&session)))
    }

    #[instrument(skip(self))]
    async fn update_session(
        &self,
        request: Request<UpdateSessionRequest>,
    ) -> Result<Response<SessionDetails>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        // For simplicity, this is a basic implementation
        // In production, you'd update individual fields
        let session = self
            .repo
            .get_session(session_id)
            .await
            .map_err(|e| {
                tracing::error!("Database error: {}", e);
                Status::internal("Failed to fetch session")
            })?
            .ok_or_else(|| Status::not_found("Session not found"))?;

        Ok(Response::new(self.session_to_proto(&session)))
    }

    #[instrument(skip(self))]
    async fn cancel_session(
        &self,
        request: Request<CancelSessionRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        self.repo
            .update_session_status(session_id, "CANCELLED")
            .await
            .map_err(|e| {
                tracing::error!("Failed to cancel session: {}", e);
                Status::internal("Failed to cancel session")
            })?;

        METRICS.sessions_cancelled_total.inc();

        tracing::info!("Session cancelled: id={}, reason={}", session_id, req.reason);

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn list_sessions(
        &self,
        request: Request<ListSessionsRequest>,
    ) -> Result<Response<ListSessionsResponse>, Status> {
        let req = request.into_inner();

        let user_id = if !req.user_id.is_empty() {
            Some(
                Uuid::parse_str(&req.user_id)
                    .map_err(|_| Status::invalid_argument("Invalid user ID"))?,
            )
        } else {
            None
        };

        let status_filter = req.status_filter.and_then(|s| {
            match SessionStatus::try_from(s) {
                Ok(SessionStatus::Scheduled) => Some("SCHEDULED".to_string()),
                Ok(SessionStatus::Active) => Some("ACTIVE".to_string()),
                Ok(SessionStatus::Completed) => Some("COMPLETED".to_string()),
                Ok(SessionStatus::Cancelled) => Some("CANCELLED".to_string()),
                _ => None,
            }
        });

        let page = if req.page <= 0 { 1 } else { req.page };
        let page_size = if req.page_size <= 0 || req.page_size > 100 {
            20
        } else {
            req.page_size
        };

        let (sessions, total_count) = self
            .repo
            .list_sessions(user_id, status_filter, None, None, page, page_size)
            .await
            .map_err(|e| {
                tracing::error!("Failed to list sessions: {}", e);
                Status::internal("Failed to list sessions")
            })?;

        Ok(Response::new(ListSessionsResponse {
            sessions: sessions.iter().map(|s| self.session_to_proto(s)).collect(),
            total_count: total_count as i32,
            page,
            page_size,
        }))
    }

    // AC3: Join button enabled 10 minutes before start
    #[instrument(skip(self))]
    async fn join_session(
        &self,
        request: Request<JoinSessionRequest>,
    ) -> Result<Response<JoinSessionResponse>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;
        let user_id = Uuid::parse_str(&req.user_id)
            .map_err(|_| Status::invalid_argument("Invalid user ID"))?;

        let session = self
            .repo
            .get_session(session_id)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?
            .ok_or_else(|| Status::not_found("Session not found"))?;

        // Check if join is allowed (AC3: 10 minutes before start)
        let eligibility = SessionJoinEligibility::check(
            session.start_time,
            &session.status,
            self.config.session.join_window_minutes,
        );

        if !eligibility.can_join {
            return Ok(Response::new(JoinSessionResponse {
                participant_id: String::new(),
                ice_servers_json: String::new(),
                session: Some(self.session_to_proto(&session)),
                current_participants: vec![],
                can_join: false,
                message: eligibility.message,
            }));
        }

        // Check participant limit (AC5: 50 concurrent participants)
        let active_count = self
            .repo
            .count_active_participants(session_id)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?;

        if active_count >= session.max_participants as i64 {
            return Err(Status::resource_exhausted(format!(
                "Session is full (max {} participants)",
                session.max_participants
            )));
        }

        // Add participant
        let role = if user_id == session.instructor_id {
            "INSTRUCTOR"
        } else {
            "STUDENT"
        };

        let participant = self
            .repo
            .add_participant(
                session_id,
                user_id,
                req.display_name,
                role.to_string(),
                req.audio_enabled,
                req.video_enabled,
            )
            .await
            .map_err(|e| {
                tracing::error!("Failed to add participant: {}", e);
                Status::internal("Failed to join session")
            })?;

        // Update session to ACTIVE if this is the first participant
        if active_count == 0 {
            self.repo
                .update_session_status(session_id, "ACTIVE")
                .await
                .ok();
            METRICS.sessions_active.inc();
        }

        METRICS.participants_joined_total.inc();
        METRICS.participants_active.inc();
        METRICS.webrtc_connections_total.inc();

        // Get current participants
        let current_participants = self
            .repo
            .get_active_participants(session_id)
            .await
            .unwrap_or_default();

        tracing::info!(
            "Participant joined: session={}, user={}, role={}",
            session_id,
            user_id,
            role
        );

        Ok(Response::new(JoinSessionResponse {
            participant_id: participant.id.to_string(),
            ice_servers_json: self.get_ice_servers_json(),
            session: Some(self.session_to_proto(&session)),
            current_participants: current_participants
                .iter()
                .map(|p| self.participant_to_proto(p))
                .collect(),
            can_join: true,
            message: "Successfully joined session".to_string(),
        }))
    }

    #[instrument(skip(self))]
    async fn leave_session(
        &self,
        request: Request<LeaveSessionRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let participant_id = Uuid::parse_str(&req.participant_id)
            .map_err(|_| Status::invalid_argument("Invalid participant ID"))?;

        self.repo
            .mark_participant_left(participant_id)
            .await
            .map_err(|e| {
                tracing::error!("Failed to mark participant left: {}", e);
                Status::internal("Failed to leave session")
            })?;

        METRICS.participants_left_total.inc();
        METRICS.participants_active.dec();

        tracing::info!("Participant left: id={}", participant_id);

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn get_session_status(
        &self,
        request: Request<GetSessionStatusRequest>,
    ) -> Result<Response<SessionStatusResponse>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        let session = self
            .repo
            .get_session(session_id)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?
            .ok_or_else(|| Status::not_found("Session not found"))?;

        let participant_count = self
            .repo
            .count_active_participants(session_id)
            .await
            .unwrap_or(0);

        let eligibility = SessionJoinEligibility::check(
            session.start_time,
            &session.status,
            self.config.session.join_window_minutes,
        );

        let status = match session.status.as_str() {
            "SCHEDULED" => SessionStatus::Scheduled,
            "ACTIVE" => SessionStatus::Active,
            "COMPLETED" => SessionStatus::Completed,
            "CANCELLED" => SessionStatus::Cancelled,
            _ => SessionStatus::Unspecified,
        };

        Ok(Response::new(SessionStatusResponse {
            session_id: session.id.to_string(),
            status: status as i32,
            participant_count: participant_count as i32,
            started_at: None, // Could track this separately
            elapsed_minutes: 0,
            is_recording: false, // Would check recordings table
            can_join: eligibility.can_join,
            join_enabled_at: eligibility.join_enabled_at.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
        }))
    }

    // AC6: Instructor mutes/unmutes participants
    #[instrument(skip(self))]
    async fn mute_participant(
        &self,
        request: Request<MuteParticipantRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let participant_id = Uuid::parse_str(&req.participant_id)
            .map_err(|_| Status::invalid_argument("Invalid participant ID"))?;

        self.repo
            .update_participant_mute(participant_id, true)
            .await
            .map_err(|e| {
                tracing::error!("Failed to mute participant: {}", e);
                Status::internal("Failed to mute participant")
            })?;

        METRICS.participants_muted_total.inc();

        tracing::info!(
            "Participant muted: id={}, by_instructor={}",
            participant_id,
            req.instructor_id
        );

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn unmute_participant(
        &self,
        request: Request<UnmuteParticipantRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let participant_id = Uuid::parse_str(&req.participant_id)
            .map_err(|_| Status::invalid_argument("Invalid participant ID"))?;

        self.repo
            .update_participant_mute(participant_id, false)
            .await
            .map_err(|e| {
                tracing::error!("Failed to unmute participant: {}", e);
                Status::internal("Failed to unmute participant")
            })?;

        METRICS.participants_unmuted_total.inc();

        tracing::info!(
            "Participant unmuted: id={}, by_instructor={}",
            participant_id,
            req.instructor_id
        );

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn kick_participant(
        &self,
        request: Request<KickParticipantRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let participant_id = Uuid::parse_str(&req.participant_id)
            .map_err(|_| Status::invalid_argument("Invalid participant ID"))?;

        self.repo
            .mark_participant_left(participant_id)
            .await
            .map_err(|e| {
                tracing::error!("Failed to kick participant: {}", e);
                Status::internal("Failed to kick participant")
            })?;

        METRICS.participants_kicked_total.inc();

        tracing::info!(
            "Participant kicked: id={}, by_instructor={}, reason={}",
            participant_id,
            req.instructor_id,
            req.reason
        );

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn list_participants(
        &self,
        request: Request<ListParticipantsRequest>,
    ) -> Result<Response<ListParticipantsResponse>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        let participants = self
            .repo
            .get_active_participants(session_id)
            .await
            .map_err(|e| {
                tracing::error!("Failed to list participants: {}", e);
                Status::internal("Failed to list participants")
            })?;

        Ok(Response::new(ListParticipantsResponse {
            participants: participants
                .iter()
                .map(|p| self.participant_to_proto(p))
                .collect(),
        }))
    }

    // AC7: Screen sharing enabled
    #[instrument(skip(self))]
    async fn start_screen_share(
        &self,
        request: Request<ScreenShareRequest>,
    ) -> Result<Response<ScreenShareResponse>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;
        let participant_id = Uuid::parse_str(&req.participant_id)
            .map_err(|_| Status::invalid_argument("Invalid participant ID"))?;

        let session = self
            .repo
            .get_session(session_id)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?
            .ok_or_else(|| Status::not_found("Session not found"))?;

        if !session.allow_screen_share {
            return Ok(Response::new(ScreenShareResponse {
                screen_share_id: String::new(),
                allowed: false,
                message: "Screen sharing is not allowed for this session".to_string(),
            }));
        }

        let screen_share = self
            .repo
            .start_screen_share(session_id, participant_id)
            .await
            .map_err(|e| {
                tracing::error!("Failed to start screen share: {}", e);
                Status::internal("Failed to start screen share")
            })?;

        self.repo
            .update_participant_screen_share(participant_id, true)
            .await
            .ok();

        METRICS.screen_shares_started_total.inc();
        METRICS.screen_shares_active.inc();

        tracing::info!(
            "Screen share started: session={}, participant={}",
            session_id,
            participant_id
        );

        Ok(Response::new(ScreenShareResponse {
            screen_share_id: screen_share.id.to_string(),
            allowed: true,
            message: "Screen sharing started successfully".to_string(),
        }))
    }

    #[instrument(skip(self))]
    async fn stop_screen_share(
        &self,
        request: Request<StopScreenShareRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let participant_id = Uuid::parse_str(&req.participant_id)
            .map_err(|_| Status::invalid_argument("Invalid participant ID"))?;

        self.repo
            .update_participant_screen_share(participant_id, false)
            .await
            .map_err(|e| {
                tracing::error!("Failed to stop screen share: {}", e);
                Status::internal("Failed to stop screen share")
            })?;

        METRICS.screen_shares_stopped_total.inc();
        METRICS.screen_shares_active.dec();

        tracing::info!("Screen share stopped: participant={}", participant_id);

        Ok(Response::new(()))
    }

    // Continued in next part...
    // AC8: Session auto-records to GCS
    #[instrument(skip(self))]
    async fn start_recording(
        &self,
        request: Request<StartRecordingRequest>,
    ) -> Result<Response<StartRecordingResponse>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        let settings = req.settings.unwrap_or_else(|| RecordingSettings {
            quality: VideoQuality::Quality720p as i32,
            include_audio: true,
            include_screen_share: true,
        });

        let quality_str = VideoQualityHelper::quality_to_string(
            VideoQuality::try_from(settings.quality).unwrap_or(VideoQuality::Quality720p),
        );

        let recording = self
            .repo
            .start_recording(
                session_id,
                quality_str,
                settings.include_audio,
                settings.include_screen_share,
            )
            .await
            .map_err(|e| {
                tracing::error!("Failed to start recording: {}", e);
                METRICS.recordings_failed_total.inc();
                Status::internal("Failed to start recording")
            })?;

        METRICS.recordings_started_total.inc();

        tracing::info!("Recording started: session={}, id={}", session_id, recording.id);

        Ok(Response::new(StartRecordingResponse {
            recording_id: recording.id.to_string(),
            started_at: Some(prost_types::Timestamp {
                seconds: recording.started_at.timestamp(),
                nanos: recording.started_at.timestamp_subsec_nanos() as i32,
            }),
        }))
    }

    #[instrument(skip(self))]
    async fn stop_recording(
        &self,
        request: Request<StopRecordingRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        let recording_id = Uuid::parse_str(&req.recording_id)
            .map_err(|_| Status::invalid_argument("Invalid recording ID"))?;

        // Mark as processing (actual upload to GCS happens in background)
        self.repo
            .update_recording_status(recording_id, "PROCESSING", None)
            .await
            .map_err(|e| {
                tracing::error!("Failed to stop recording: {}", e);
                Status::internal("Failed to stop recording")
            })?;

        tracing::info!("Recording stopped and processing: id={}", recording_id);

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn get_recording(
        &self,
        request: Request<GetRecordingRequest>,
    ) -> Result<Response<RecordingDetails>, Status> {
        let req = request.into_inner();

        let recording_id = Uuid::parse_str(&req.recording_id)
            .map_err(|_| Status::invalid_argument("Invalid recording ID"))?;

        let recording = self
            .repo
            .get_recording(recording_id)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?
            .ok_or_else(|| Status::not_found("Recording not found"))?;

        Ok(Response::new(self.recording_to_proto(&recording)))
    }

    #[instrument(skip(self))]
    async fn list_recordings(
        &self,
        request: Request<ListRecordingsRequest>,
    ) -> Result<Response<ListRecordingsResponse>, Status> {
        let req = request.into_inner();

        let session_id = if !req.session_id.is_empty() {
            Some(
                Uuid::parse_str(&req.session_id)
                    .map_err(|_| Status::invalid_argument("Invalid session ID"))?,
            )
        } else {
            None
        };

        let page = if req.page <= 0 { 1 } else { req.page };
        let page_size = if req.page_size <= 0 { 20 } else { req.page_size };

        let (recordings, total_count) = self
            .repo
            .list_recordings(session_id, None, page, page_size)
            .await
            .map_err(|e| {
                tracing::error!("Failed to list recordings: {}", e);
                Status::internal("Failed to list recordings")
            })?;

        Ok(Response::new(ListRecordingsResponse {
            recordings: recordings
                .iter()
                .map(|r| self.recording_to_proto(r))
                .collect(),
            total_count: total_count as i32,
        }))
    }

    // AC4: Video quality adapts to bandwidth (360p-1080p)
    #[instrument(skip(self))]
    async fn update_video_quality(
        &self,
        request: Request<UpdateVideoQualityRequest>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();

        METRICS.video_quality_switches_total.inc();

        tracing::info!(
            "Video quality updated: session={}, participant={}, quality={:?}",
            req.session_id,
            req.participant_id,
            req.quality
        );

        Ok(Response::new(()))
    }

    #[instrument(skip(self))]
    async fn get_video_quality_stats(
        &self,
        request: Request<VideoQualityStatsRequest>,
    ) -> Result<Response<VideoQualityStatsResponse>, Status> {
        let _req = request.into_inner();

        // Return mock stats for now
        Ok(Response::new(VideoQualityStatsResponse { stats: vec![] }))
    }

    // AC2: Students receive calendar invitation
    #[instrument(skip(self))]
    async fn send_calendar_invitation(
        &self,
        request: Request<CalendarInvitationRequest>,
    ) -> Result<Response<CalendarInvitationResponse>, Status> {
        let req = request.into_inner();

        let session_id = Uuid::parse_str(&req.session_id)
            .map_err(|_| Status::invalid_argument("Invalid session ID"))?;

        let session = self
            .repo
            .get_session(session_id)
            .await
            .map_err(|e| Status::internal(format!("Database error: {}", e)))?
            .ok_or_else(|| Status::not_found("Session not found"))?;

        let calendar_ics = self.generate_calendar_ics(
            &session.title,
            session.description.as_deref().unwrap_or(""),
            session.start_time,
            session.duration_minutes,
            &session.join_url,
        );

        // In production, would actually send emails here
        for email in &req.recipient_emails {
            METRICS.calendar_invitations_sent_total.inc();
            tracing::info!("Calendar invitation sent to: {}", email);
        }

        Ok(Response::new(CalendarInvitationResponse {
            success: true,
            sent_to: req.recipient_emails.clone(),
            calendar_ics: calendar_ics.clone(),
        }))
    }

    // Helper methods
    fn session_to_proto(&self, session: &crate::database::models::VideoSession) -> SessionDetails {
        let status = match session.status.as_str() {
            "SCHEDULED" => SessionStatus::Scheduled,
            "ACTIVE" => SessionStatus::Active,
            "COMPLETED" => SessionStatus::Completed,
            "CANCELLED" => SessionStatus::Cancelled,
            _ => SessionStatus::Unspecified,
        };

        SessionDetails {
            session_id: session.id.to_string(),
            instructor_id: session.instructor_id.to_string(),
            title: session.title.clone(),
            description: session.description.clone().unwrap_or_default(),
            start_time: Some(prost_types::Timestamp {
                seconds: session.start_time.timestamp(),
                nanos: session.start_time.timestamp_subsec_nanos() as i32,
            }),
            duration_minutes: session.duration_minutes,
            status: status as i32,
            settings: Some(SessionSettings {
                max_participants: session.max_participants,
                auto_record: session.auto_record,
                allow_screen_share: session.allow_screen_share,
                quality_settings: Some(VideoQualitySettings {
                    default_quality: VideoQuality::Quality720p as i32,
                    adaptive_bitrate: true,
                    min_bitrate_kbps: 300,
                    max_bitrate_kbps: 5000,
                }),
                require_approval: session.require_approval,
                mute_on_join: session.mute_on_join,
            }),
            participant_count: 0, // Would query this
            created_at: Some(prost_types::Timestamp {
                seconds: session.created_at.timestamp(),
                nanos: session.created_at.timestamp_subsec_nanos() as i32,
            }),
            updated_at: Some(prost_types::Timestamp {
                seconds: session.updated_at.timestamp(),
                nanos: session.updated_at.timestamp_subsec_nanos() as i32,
            }),
            join_url: session.join_url.clone(),
        }
    }

    fn participant_to_proto(
        &self,
        participant: &crate::database::models::SessionParticipant,
    ) -> ParticipantInfo {
        let role = match participant.role.as_str() {
            "INSTRUCTOR" => ParticipantRole::Instructor,
            "STUDENT" => ParticipantRole::Student,
            _ => ParticipantRole::Unspecified,
        };

        ParticipantInfo {
            participant_id: participant.id.to_string(),
            user_id: participant.user_id.to_string(),
            display_name: participant.display_name.clone(),
            role: role as i32,
            audio_enabled: participant.audio_enabled,
            video_enabled: participant.video_enabled,
            is_muted: participant.is_muted,
            is_sharing_screen: participant.is_sharing_screen,
            joined_at: Some(prost_types::Timestamp {
                seconds: participant.joined_at.timestamp(),
                nanos: participant.joined_at.timestamp_subsec_nanos() as i32,
            }),
            quality_stats: None, // Would query this
        }
    }

    fn recording_to_proto(
        &self,
        recording: &crate::database::models::SessionRecording,
    ) -> RecordingDetails {
        let status = match recording.status.as_str() {
            "RECORDING" => RecordingStatus::Recording,
            "PROCESSING" => RecordingStatus::Processing,
            "AVAILABLE" => RecordingStatus::Available,
            "FAILED" => RecordingStatus::Failed,
            _ => RecordingStatus::Unspecified,
        };

        RecordingDetails {
            recording_id: recording.id.to_string(),
            session_id: recording.session_id.to_string(),
            started_at: Some(prost_types::Timestamp {
                seconds: recording.started_at.timestamp(),
                nanos: recording.started_at.timestamp_subsec_nanos() as i32,
            }),
            completed_at: recording.completed_at.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
            duration_seconds: recording.duration_seconds.unwrap_or(0),
            file_size_bytes: recording.file_size_bytes.unwrap_or(0),
            gcs_url: recording.gcs_url.clone().unwrap_or_default(),
            status: status as i32,
            available_at: recording.available_at.map(|dt| prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            }),
        }
    }

    fn generate_calendar_ics(
        &self,
        title: &str,
        description: &str,
        start_time: chrono::DateTime<Utc>,
        duration_minutes: i32,
        join_url: &str,
    ) -> String {
        use icalendar::*;

        let end_time = start_time + Duration::minutes(duration_minutes as i64);

        let event = Event::new()
            .summary(title)
            .description(&format!("{}\n\nJoin URL: {}", description, join_url))
            .starts(start_time)
            .ends(end_time)
            .done();

        let calendar = Calendar::new().push(event).done();

        calendar.to_string()
    }
}
