-- Video Conferencing Schema Migration
-- Epic 7: Live Video (21 points)

-- Sessions table (AC1: Schedule video session)
CREATE TABLE IF NOT EXISTS video_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instructor_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_minutes INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'SCHEDULED',
    -- AC5: Supports 50 concurrent participants
    max_participants INTEGER NOT NULL DEFAULT 50,
    auto_record BOOLEAN NOT NULL DEFAULT true, -- AC8
    allow_screen_share BOOLEAN NOT NULL DEFAULT true, -- AC7
    require_approval BOOLEAN NOT NULL DEFAULT false,
    mute_on_join BOOLEAN NOT NULL DEFAULT false,
    join_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_status CHECK (status IN ('SCHEDULED', 'ACTIVE', 'COMPLETED', 'CANCELLED')),
    CONSTRAINT positive_duration CHECK (duration_minutes > 0),
    CONSTRAINT positive_max_participants CHECK (max_participants > 0 AND max_participants <= 100)
);

-- Video quality settings (AC4: 360p-1080p adaptive)
CREATE TABLE IF NOT EXISTS video_quality_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES video_sessions(id) ON DELETE CASCADE,
    default_quality VARCHAR(50) NOT NULL DEFAULT 'QUALITY_720P',
    adaptive_bitrate BOOLEAN NOT NULL DEFAULT true, -- AC4: adapts to bandwidth
    min_bitrate_kbps INTEGER NOT NULL DEFAULT 300,
    max_bitrate_kbps INTEGER NOT NULL DEFAULT 5000,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_quality CHECK (default_quality IN ('QUALITY_360P', 'QUALITY_480P', 'QUALITY_720P', 'QUALITY_1080P')),
    CONSTRAINT valid_bitrate_range CHECK (min_bitrate_kbps < max_bitrate_kbps)
);

-- Session participants (AC5: 50 concurrent participants)
CREATE TABLE IF NOT EXISTS session_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES video_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'STUDENT',
    audio_enabled BOOLEAN NOT NULL DEFAULT true,
    video_enabled BOOLEAN NOT NULL DEFAULT true,
    is_muted BOOLEAN NOT NULL DEFAULT false, -- AC6: Instructor mutes/unmutes
    is_sharing_screen BOOLEAN NOT NULL DEFAULT false, -- AC7: Screen sharing
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT valid_role CHECK (role IN ('INSTRUCTOR', 'STUDENT')),
    UNIQUE(session_id, user_id)
);

-- Video quality stats per participant (AC4: quality adapts to bandwidth)
CREATE TABLE IF NOT EXISTS participant_quality_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID NOT NULL REFERENCES session_participants(id) ON DELETE CASCADE,
    current_quality VARCHAR(50) NOT NULL,
    bitrate_kbps INTEGER NOT NULL,
    frame_rate INTEGER NOT NULL,
    packet_loss_percentage INTEGER NOT NULL DEFAULT 0,
    jitter_ms INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_quality_stats CHECK (current_quality IN ('QUALITY_360P', 'QUALITY_480P', 'QUALITY_720P', 'QUALITY_1080P'))
);

-- Session recordings (AC8: Auto-records to GCS, AC9: Available within 30 minutes)
CREATE TABLE IF NOT EXISTS session_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES video_sessions(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    file_size_bytes BIGINT,
    gcs_url TEXT, -- AC8: stored in Google Cloud Storage
    gcs_bucket VARCHAR(255),
    gcs_object_key TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'RECORDING',
    -- AC9: Recording available within 30 minutes
    available_at TIMESTAMP WITH TIME ZONE,
    quality VARCHAR(50) NOT NULL DEFAULT 'QUALITY_720P',
    include_audio BOOLEAN NOT NULL DEFAULT true,
    include_screen_share BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_recording_status CHECK (status IN ('RECORDING', 'PROCESSING', 'AVAILABLE', 'FAILED')),
    CONSTRAINT valid_recording_quality CHECK (quality IN ('QUALITY_360P', 'QUALITY_480P', 'QUALITY_720P', 'QUALITY_1080P'))
);

-- Screen sharing sessions (AC7: Screen sharing enabled)
CREATE TABLE IF NOT EXISTS screen_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES video_sessions(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES session_participants(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(session_id, participant_id, started_at)
);

-- Calendar invitations (AC2: Students receive calendar invitation)
CREATE TABLE IF NOT EXISTS calendar_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES video_sessions(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    calendar_ics TEXT NOT NULL, -- iCalendar format
    delivery_status VARCHAR(50) NOT NULL DEFAULT 'SENT',
    error_message TEXT,
    CONSTRAINT valid_delivery_status CHECK (delivery_status IN ('SENT', 'DELIVERED', 'FAILED'))
);

-- WebRTC signaling events (for debugging and analytics)
CREATE TABLE IF NOT EXISTS webrtc_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES video_sessions(id) ON DELETE CASCADE,
    participant_id UUID REFERENCES session_participants(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance

-- Session queries
CREATE INDEX idx_video_sessions_instructor ON video_sessions(instructor_id);
CREATE INDEX idx_video_sessions_status ON video_sessions(status);
CREATE INDEX idx_video_sessions_start_time ON video_sessions(start_time);
CREATE INDEX idx_video_sessions_status_start_time ON video_sessions(status, start_time);

-- Participant queries
CREATE INDEX idx_session_participants_session ON session_participants(session_id);
CREATE INDEX idx_session_participants_user ON session_participants(user_id);
CREATE INDEX idx_session_participants_joined_at ON session_participants(joined_at);
CREATE INDEX idx_session_participants_active ON session_participants(session_id) WHERE left_at IS NULL;

-- Quality stats queries
CREATE INDEX idx_quality_stats_participant ON participant_quality_stats(participant_id);
CREATE INDEX idx_quality_stats_recorded_at ON participant_quality_stats(recorded_at);

-- Recording queries
CREATE INDEX idx_recordings_session ON session_recordings(session_id);
CREATE INDEX idx_recordings_status ON session_recordings(status);
CREATE INDEX idx_recordings_available_at ON session_recordings(available_at);

-- Screen share queries
CREATE INDEX idx_screen_shares_session ON screen_shares(session_id);
CREATE INDEX idx_screen_shares_participant ON screen_shares(participant_id);
CREATE INDEX idx_screen_shares_active ON screen_shares(session_id) WHERE stopped_at IS NULL;

-- Calendar invitation queries
CREATE INDEX idx_calendar_invitations_session ON calendar_invitations(session_id);
CREATE INDEX idx_calendar_invitations_recipient ON calendar_invitations(recipient_user_id);

-- WebRTC event queries
CREATE INDEX idx_webrtc_events_session ON webrtc_events(session_id);
CREATE INDEX idx_webrtc_events_participant ON webrtc_events(participant_id);
CREATE INDEX idx_webrtc_events_created_at ON webrtc_events(created_at);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_video_sessions_updated_at BEFORE UPDATE ON video_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE video_sessions IS 'AC1: Video session scheduling with date, time, and duration';
COMMENT ON TABLE session_participants IS 'AC5: Supports 50 concurrent participants per session';
COMMENT ON TABLE participant_quality_stats IS 'AC4: Video quality adapts to bandwidth (360p-1080p)';
COMMENT ON TABLE session_recordings IS 'AC8: Session auto-records to GCS, AC9: Available within 30 minutes';
COMMENT ON TABLE screen_shares IS 'AC7: Screen sharing enabled for participants';
COMMENT ON TABLE calendar_invitations IS 'AC2: Students receive calendar invitation';
COMMENT ON COLUMN session_participants.is_muted IS 'AC6: Instructor can mute/unmute participants';
