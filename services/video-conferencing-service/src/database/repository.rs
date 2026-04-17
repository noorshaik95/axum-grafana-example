use super::models::*;
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Clone)]
pub struct VideoRepository {
    pool: PgPool,
}

impl VideoRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    // Session operations
    pub async fn create_session(
        &self,
        instructor_id: Uuid,
        title: String,
        description: Option<String>,
        start_time: DateTime<Utc>,
        duration_minutes: i32,
        max_participants: i32,
        auto_record: bool,
        allow_screen_share: bool,
        require_approval: bool,
        mute_on_join: bool,
        join_url: String,
    ) -> anyhow::Result<VideoSession> {
        let session = sqlx::query_as::<_, VideoSession>(
            r#"
            INSERT INTO video_sessions (
                instructor_id, title, description, start_time, duration_minutes,
                max_participants, auto_record, allow_screen_share, require_approval,
                mute_on_join, join_url, status
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'SCHEDULED')
            RETURNING *
            "#,
        )
        .bind(instructor_id)
        .bind(title)
        .bind(description)
        .bind(start_time)
        .bind(duration_minutes)
        .bind(max_participants)
        .bind(auto_record)
        .bind(allow_screen_share)
        .bind(require_approval)
        .bind(mute_on_join)
        .bind(join_url)
        .fetch_one(&self.pool)
        .await?;

        Ok(session)
    }

    pub async fn get_session(&self, session_id: Uuid) -> anyhow::Result<Option<VideoSession>> {
        let session = sqlx::query_as::<_, VideoSession>(
            "SELECT * FROM video_sessions WHERE id = $1"
        )
        .bind(session_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(session)
    }

    pub async fn update_session_status(&self, session_id: Uuid, status: &str) -> anyhow::Result<()> {
        sqlx::query("UPDATE video_sessions SET status = $1, updated_at = NOW() WHERE id = $2")
            .bind(status)
            .bind(session_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    pub async fn list_sessions(
        &self,
        user_id: Option<Uuid>,
        status_filter: Option<String>,
        start_date: Option<DateTime<Utc>>,
        end_date: Option<DateTime<Utc>>,
        page: i32,
        page_size: i32,
    ) -> anyhow::Result<(Vec<VideoSession>, i64)> {
        let offset = (page - 1) * page_size;

        // Build dynamic query
        let mut query = String::from("SELECT * FROM video_sessions WHERE 1=1");
        let mut count_query = String::from("SELECT COUNT(*) FROM video_sessions WHERE 1=1");

        if user_id.is_some() {
            query.push_str(" AND instructor_id = $1");
            count_query.push_str(" AND instructor_id = $1");
        }
        if status_filter.is_some() {
            query.push_str(&format!(" AND status = ${}", if user_id.is_some() { 2 } else { 1 }));
            count_query.push_str(&format!(" AND status = ${}", if user_id.is_some() { 2 } else { 1 }));
        }

        query.push_str(" ORDER BY start_time DESC LIMIT $");
        query.push_str(&(if user_id.is_some() && status_filter.is_some() { 3 } else if user_id.is_some() || status_filter.is_some() { 2 } else { 1 }).to_string());
        query.push_str(" OFFSET $");
        query.push_str(&(if user_id.is_some() && status_filter.is_some() { 4 } else if user_id.is_some() || status_filter.is_some() { 3 } else { 2 }).to_string());

        let mut query_builder = sqlx::query_as::<_, VideoSession>(&query);
        let mut count_builder = sqlx::query_scalar::<_, i64>(&count_query);

        if let Some(uid) = user_id {
            query_builder = query_builder.bind(uid);
            count_builder = count_builder.bind(uid);
        }
        if let Some(ref status) = status_filter {
            query_builder = query_builder.bind(status);
            count_builder = count_builder.bind(status);
        }

        query_builder = query_builder.bind(page_size).bind(offset);

        let sessions = query_builder.fetch_all(&self.pool).await?;
        let total_count = count_builder.fetch_one(&self.pool).await?;

        Ok((sessions, total_count))
    }

    // Participant operations
    pub async fn add_participant(
        &self,
        session_id: Uuid,
        user_id: Uuid,
        display_name: String,
        role: String,
        audio_enabled: bool,
        video_enabled: bool,
    ) -> anyhow::Result<SessionParticipant> {
        let participant = sqlx::query_as::<_, SessionParticipant>(
            r#"
            INSERT INTO session_participants (
                session_id, user_id, display_name, role, audio_enabled, video_enabled
            )
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING *
            "#,
        )
        .bind(session_id)
        .bind(user_id)
        .bind(display_name)
        .bind(role)
        .bind(audio_enabled)
        .bind(video_enabled)
        .fetch_one(&self.pool)
        .await?;

        Ok(participant)
    }

    pub async fn get_participant(&self, participant_id: Uuid) -> anyhow::Result<Option<SessionParticipant>> {
        let participant = sqlx::query_as::<_, SessionParticipant>(
            "SELECT * FROM session_participants WHERE id = $1"
        )
        .bind(participant_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(participant)
    }

    pub async fn get_active_participants(&self, session_id: Uuid) -> anyhow::Result<Vec<SessionParticipant>> {
        let participants = sqlx::query_as::<_, SessionParticipant>(
            "SELECT * FROM session_participants WHERE session_id = $1 AND left_at IS NULL"
        )
        .bind(session_id)
        .fetch_all(&self.pool)
        .await?;

        Ok(participants)
    }

    pub async fn count_active_participants(&self, session_id: Uuid) -> anyhow::Result<i64> {
        let count = sqlx::query_scalar::<_, i64>(
            "SELECT COUNT(*) FROM session_participants WHERE session_id = $1 AND left_at IS NULL"
        )
        .bind(session_id)
        .fetch_one(&self.pool)
        .await?;

        Ok(count)
    }

    pub async fn update_participant_mute(&self, participant_id: Uuid, is_muted: bool) -> anyhow::Result<()> {
        sqlx::query("UPDATE session_participants SET is_muted = $1 WHERE id = $2")
            .bind(is_muted)
            .bind(participant_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    pub async fn update_participant_screen_share(&self, participant_id: Uuid, is_sharing: bool) -> anyhow::Result<()> {
        sqlx::query("UPDATE session_participants SET is_sharing_screen = $1 WHERE id = $2")
            .bind(is_sharing)
            .bind(participant_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    pub async fn mark_participant_left(&self, participant_id: Uuid) -> anyhow::Result<()> {
        sqlx::query("UPDATE session_participants SET left_at = NOW() WHERE id = $1")
            .bind(participant_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    // Quality settings operations
    pub async fn create_quality_settings(
        &self,
        session_id: Uuid,
        default_quality: String,
        adaptive_bitrate: bool,
        min_bitrate_kbps: i32,
        max_bitrate_kbps: i32,
    ) -> anyhow::Result<VideoQualitySettings> {
        let settings = sqlx::query_as::<_, VideoQualitySettings>(
            r#"
            INSERT INTO video_quality_settings (
                session_id, default_quality, adaptive_bitrate, min_bitrate_kbps, max_bitrate_kbps
            )
            VALUES ($1, $2, $3, $4, $5)
            RETURNING *
            "#,
        )
        .bind(session_id)
        .bind(default_quality)
        .bind(adaptive_bitrate)
        .bind(min_bitrate_kbps)
        .bind(max_bitrate_kbps)
        .fetch_one(&self.pool)
        .await?;

        Ok(settings)
    }

    pub async fn get_quality_settings(&self, session_id: Uuid) -> anyhow::Result<Option<VideoQualitySettings>> {
        let settings = sqlx::query_as::<_, VideoQualitySettings>(
            "SELECT * FROM video_quality_settings WHERE session_id = $1"
        )
        .bind(session_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(settings)
    }

    // Quality stats operations
    pub async fn save_quality_stats(
        &self,
        participant_id: Uuid,
        current_quality: String,
        bitrate_kbps: i32,
        frame_rate: i32,
        packet_loss_percentage: i32,
        jitter_ms: i32,
        latency_ms: i32,
        bytes_sent: i64,
        bytes_received: i64,
    ) -> anyhow::Result<ParticipantQualityStats> {
        let stats = sqlx::query_as::<_, ParticipantQualityStats>(
            r#"
            INSERT INTO participant_quality_stats (
                participant_id, current_quality, bitrate_kbps, frame_rate,
                packet_loss_percentage, jitter_ms, latency_ms, bytes_sent, bytes_received
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            RETURNING *
            "#,
        )
        .bind(participant_id)
        .bind(current_quality)
        .bind(bitrate_kbps)
        .bind(frame_rate)
        .bind(packet_loss_percentage)
        .bind(jitter_ms)
        .bind(latency_ms)
        .bind(bytes_sent)
        .bind(bytes_received)
        .fetch_one(&self.pool)
        .await?;

        Ok(stats)
    }

    // Recording operations
    pub async fn start_recording(
        &self,
        session_id: Uuid,
        quality: String,
        include_audio: bool,
        include_screen_share: bool,
    ) -> anyhow::Result<SessionRecording> {
        let recording = sqlx::query_as::<_, SessionRecording>(
            r#"
            INSERT INTO session_recordings (
                session_id, started_at, quality, include_audio, include_screen_share, status
            )
            VALUES ($1, NOW(), $2, $3, $4, 'RECORDING')
            RETURNING *
            "#,
        )
        .bind(session_id)
        .bind(quality)
        .bind(include_audio)
        .bind(include_screen_share)
        .fetch_one(&self.pool)
        .await?;

        Ok(recording)
    }

    pub async fn complete_recording(
        &self,
        recording_id: Uuid,
        duration_seconds: i32,
        file_size_bytes: i64,
        gcs_url: String,
        gcs_bucket: String,
        gcs_object_key: String,
    ) -> anyhow::Result<()> {
        sqlx::query(
            r#"
            UPDATE session_recordings
            SET completed_at = NOW(),
                duration_seconds = $1,
                file_size_bytes = $2,
                gcs_url = $3,
                gcs_bucket = $4,
                gcs_object_key = $5,
                status = 'AVAILABLE',
                available_at = NOW()
            WHERE id = $6
            "#,
        )
        .bind(duration_seconds)
        .bind(file_size_bytes)
        .bind(gcs_url)
        .bind(gcs_bucket)
        .bind(gcs_object_key)
        .bind(recording_id)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn update_recording_status(&self, recording_id: Uuid, status: &str, error_message: Option<String>) -> anyhow::Result<()> {
        sqlx::query("UPDATE session_recordings SET status = $1, error_message = $2 WHERE id = $3")
            .bind(status)
            .bind(error_message)
            .bind(recording_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    pub async fn get_recording(&self, recording_id: Uuid) -> anyhow::Result<Option<SessionRecording>> {
        let recording = sqlx::query_as::<_, SessionRecording>(
            "SELECT * FROM session_recordings WHERE id = $1"
        )
        .bind(recording_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(recording)
    }

    pub async fn list_recordings(&self, session_id: Option<Uuid>, instructor_id: Option<Uuid>, page: i32, page_size: i32) -> anyhow::Result<(Vec<SessionRecording>, i64)> {
        let offset = (page - 1) * page_size;

        let recordings = if let Some(sid) = session_id {
            sqlx::query_as::<_, SessionRecording>(
                "SELECT * FROM session_recordings WHERE session_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3"
            )
            .bind(sid)
            .bind(page_size)
            .bind(offset)
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as::<_, SessionRecording>(
                "SELECT * FROM session_recordings ORDER BY started_at DESC LIMIT $1 OFFSET $2"
            )
            .bind(page_size)
            .bind(offset)
            .fetch_all(&self.pool)
            .await?
        };

        let total_count = if let Some(sid) = session_id {
            sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM session_recordings WHERE session_id = $1")
                .bind(sid)
                .fetch_one(&self.pool)
                .await?
        } else {
            sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM session_recordings")
                .fetch_one(&self.pool)
                .await?
        };

        Ok((recordings, total_count))
    }

    // Screen share operations
    pub async fn start_screen_share(&self, session_id: Uuid, participant_id: Uuid) -> anyhow::Result<ScreenShare> {
        let screen_share = sqlx::query_as::<_, ScreenShare>(
            "INSERT INTO screen_shares (session_id, participant_id) VALUES ($1, $2) RETURNING *"
        )
        .bind(session_id)
        .bind(participant_id)
        .fetch_one(&self.pool)
        .await?;

        Ok(screen_share)
    }

    pub async fn stop_screen_share(&self, screen_share_id: Uuid) -> anyhow::Result<()> {
        sqlx::query("UPDATE screen_shares SET stopped_at = NOW() WHERE id = $1")
            .bind(screen_share_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    // Calendar invitation operations
    pub async fn save_calendar_invitation(
        &self,
        session_id: Uuid,
        recipient_user_id: Uuid,
        recipient_email: String,
        calendar_ics: String,
    ) -> anyhow::Result<CalendarInvitation> {
        let invitation = sqlx::query_as::<_, CalendarInvitation>(
            r#"
            INSERT INTO calendar_invitations (
                session_id, recipient_user_id, recipient_email, calendar_ics
            )
            VALUES ($1, $2, $3, $4)
            RETURNING *
            "#,
        )
        .bind(session_id)
        .bind(recipient_user_id)
        .bind(recipient_email)
        .bind(calendar_ics)
        .fetch_one(&self.pool)
        .await?;

        Ok(invitation)
    }

    // WebRTC event logging
    pub async fn log_webrtc_event(
        &self,
        session_id: Uuid,
        participant_id: Option<Uuid>,
        event_type: String,
        event_data: Option<serde_json::Value>,
    ) -> anyhow::Result<()> {
        sqlx::query(
            "INSERT INTO webrtc_events (session_id, participant_id, event_type, event_data) VALUES ($1, $2, $3, $4)"
        )
        .bind(session_id)
        .bind(participant_id)
        .bind(event_type)
        .bind(event_data)
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
