# Video Conferencing Service

Epic 7: Live Video (21 points) - US-7.1: Video Conferencing

A high-performance, scalable video conferencing service built with Rust, WebRTC, and gRPC.

## Features

### User Story: Video Conferencing (21 points)

**As an instructor, I want to host live classes so that I can teach remotely in real-time**

#### Acceptance Criteria Implementation

- **AC1: Schedule video session (date, time, duration)** ✅
  - gRPC `ScheduleSession` endpoint
  - Database storage with `video_sessions` table
  - Validation for start time and duration

- **AC2: Students receive calendar invitation** ✅
  - iCalendar (ICS) format generation
  - `SendCalendarInvitation` gRPC endpoint
  - `calendar_invitations` table for tracking

- **AC3: Join button enabled 10 minutes before start** ✅
  - `SessionJoinEligibility` checker
  - Configurable join window (default: 10 minutes)
  - `JoinSession` endpoint validates timing

- **AC4: Video quality adapts to bandwidth (360p-1080p)** ✅
  - Adaptive bitrate control (300-5000 kbps)
  - Quality levels: 360p, 480p, 720p, 1080p
  - Real-time stats reporting and adjustment
  - Metrics: bitrate, packet loss, jitter, latency

- **AC5: Supports 50 concurrent participants** ✅
  - Configurable max participants (default: 50)
  - Participant count validation on join
  - Optimized SFU architecture for scalability

- **AC6: Instructor mutes/unmutes participants** ✅
  - `MuteParticipant` / `UnmuteParticipant` endpoints
  - Role-based access control (instructor only)
  - Real-time signaling to participants

- **AC7: Screen sharing enabled** ✅
  - `StartScreenShare` / `StopScreenShare` endpoints
  - WebSocket signaling for screen share tracks
  - Per-session screen sharing control

- **AC8: Session auto-records to GCS** ✅
  - Google Cloud Storage integration
  - Background recording processor
  - `StartRecording` / `StopRecording` endpoints
  - Automatic upload pipeline

- **AC9: Recording available within 30 minutes** ✅
  - Async processing with timeout (30 min)
  - Status tracking: RECORDING → PROCESSING → AVAILABLE
  - Metrics for processing duration

## Architecture

### Components

1. **gRPC Server** (Port 50052)
   - Session management
   - Participant control
   - Recording management
   - Quality settings

2. **WebSocket Signaling Server** (Port 8082)
   - WebRTC SDP/ICE exchange
   - Real-time participant events
   - Media control signaling
   - Quality adaptation

3. **Recording Processor**
   - Background task queue
   - GCS upload pipeline
   - Video processing
   - Status updates

4. **Metrics Server** (Port 9092)
   - Prometheus metrics
   - Health checks
   - Performance monitoring

### Technology Stack

- **Language**: Rust 🦀
- **Framework**: Axum + Tonic
- **WebRTC**: webrtc-rs
- **Database**: PostgreSQL (sqlx)
- **Cloud Storage**: Google Cloud Storage
- **Observability**: OpenTelemetry + Prometheus + Grafana LGTM Stack

## Database Schema

```sql
-- Sessions
video_sessions          # AC1: Schedule sessions
video_quality_settings  # AC4: Quality configurations

-- Participants
session_participants    # AC5: 50 concurrent participants
participant_quality_stats  # AC4: Bandwidth adaptation

-- Recording
session_recordings      # AC8, AC9: Auto-record to GCS
screen_shares          # AC7: Screen sharing

-- Communication
calendar_invitations    # AC2: Calendar invites
webrtc_events          # Signaling events
```

## API Documentation

### gRPC Services

#### Session Management

```protobuf
rpc ScheduleSession(ScheduleSessionRequest) returns (ScheduleSessionResponse);
rpc GetSession(GetSessionRequest) returns (SessionDetails);
rpc UpdateSession(UpdateSessionRequest) returns (SessionDetails);
rpc CancelSession(CancelSessionRequest) returns (Empty);
rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
```

#### Session Access

```protobuf
rpc JoinSession(JoinSessionRequest) returns (JoinSessionResponse);  // AC3
rpc LeaveSession(LeaveSessionRequest) returns (Empty);
rpc GetSessionStatus(GetSessionStatusRequest) returns (SessionStatusResponse);
```

#### Participant Management

```protobuf
rpc MuteParticipant(MuteParticipantRequest) returns (Empty);     // AC6
rpc UnmuteParticipant(UnmuteParticipantRequest) returns (Empty); // AC6
rpc KickParticipant(KickParticipantRequest) returns (Empty);
rpc ListParticipants(ListParticipantsRequest) returns (ListParticipantsResponse);
```

#### Screen Sharing

```protobuf
rpc StartScreenShare(ScreenShareRequest) returns (ScreenShareResponse);  // AC7
rpc StopScreenShare(StopScreenShareRequest) returns (Empty);             // AC7
```

#### Recording

```protobuf
rpc StartRecording(StartRecordingRequest) returns (StartRecordingResponse);  // AC8
rpc StopRecording(StopRecordingRequest) returns (Empty);
rpc GetRecording(GetRecordingRequest) returns (RecordingDetails);           // AC9
rpc ListRecordings(ListRecordingsRequest) returns (ListRecordingsResponse);
```

#### Video Quality

```protobuf
rpc UpdateVideoQuality(UpdateVideoQualityRequest) returns (Empty);        // AC4
rpc GetVideoQualityStats(VideoQualityStatsRequest) returns (VideoQualityStatsResponse);
```

#### Calendar

```protobuf
rpc SendCalendarInvitation(CalendarInvitationRequest) returns (CalendarInvitationResponse);  // AC2
```

### WebSocket Signaling Protocol

#### Message Types

**Session Management**

```json
{
  "type": "Join",
  "session_id": "uuid",
  "participant_id": "uuid",
  "display_name": "John Doe"
}
```

**WebRTC Signaling**

```json
{
  "type": "Offer",
  "session_id": "uuid",
  "from": "participant-1",
  "to": "participant-2",
  "sdp": "v=0\r\no=..."
}
```

**Quality Adaptation** (AC4)

```json
{
  "type": "QualityChange",
  "session_id": "uuid",
  "participant_id": "uuid",
  "quality": "QUALITY_720P",
  "bitrate": 2500
}
```

**Screen Sharing** (AC7)

```json
{
  "type": "StartScreenShare",
  "session_id": "uuid",
  "participant_id": "uuid"
}
```

## Observability

### Prometheus Metrics

#### Session Metrics (AC1)

- `sessions_scheduled_total` - Total scheduled sessions
- `sessions_active` - Currently active sessions
- `sessions_completed_total` - Completed sessions
- `sessions_cancelled_total` - Cancelled sessions

#### Participant Metrics (AC5, AC6)

- `participants_joined_total` - Total participants joined
- `participants_left_total` - Total participants left
- `participants_active` - Currently active participants
- `participants_muted_total` - Participants muted by instructor
- `participants_unmuted_total` - Participants unmuted

#### Video Quality Metrics (AC4)

- `video_quality_switches_total` - Quality adaptation events
- `bitrate_kbps` - Current bitrate
- `packet_loss_percentage` - Network packet loss
- `frame_rate` - Video frame rate
- `latency_ms` - Network latency histogram

#### Recording Metrics (AC8, AC9)

- `recordings_started_total` - Recordings started
- `recordings_completed_total` - Recordings completed
- `recordings_failed_total` - Recording failures
- `recording_processing_duration_seconds` - Processing time (should be < 30min)
- `recording_file_size_bytes` - Recording file sizes

#### Screen Sharing Metrics (AC7)

- `screen_shares_started_total` - Screen shares started
- `screen_shares_stopped_total` - Screen shares stopped
- `screen_shares_active` - Active screen shares

#### WebRTC Metrics

- `webrtc_connections_total` - Total WebRTC connections
- `webrtc_connections_failed_total` - Failed connections
- `webrtc_ice_candidates_total` - ICE candidates exchanged

### OpenTelemetry Tracing

All gRPC methods are instrumented with distributed tracing:

- Request/response spans
- Database query spans
- External service calls (GCS)
- Error tracking

### Structured Logging

JSON-formatted logs with:

- Trace IDs for correlation
- Session and participant context
- Error details
- Performance metrics

## Configuration

See `.env.example` for full configuration options.

### Key Settings

```bash
# AC5: 50 concurrent participants
MAX_PARTICIPANTS=50

# AC3: Join enabled 10 minutes before start
JOIN_WINDOW_MINUTES=10

# AC8: Auto-recording to GCS
RECORDING_ENABLED=true
GCS_BUCKET=slate-recordings

# AC9: Processing timeout (30 minutes)
RECORDING_TIMEOUT=1800

# AC4: Video quality range
# Adaptive bitrate will select quality based on bandwidth:
# - 360p: 300-800 kbps
# - 480p: 500-1200 kbps
# - 720p: 1000-2500 kbps
# - 1080p: 2500-5000 kbps
```

## Development

### Prerequisites

- Rust 1.82+
- PostgreSQL 15+
- Google Cloud credentials (for GCS)
- protoc (Protocol Buffers compiler)

### Build

```bash
cd services/video-conferencing-service
cargo build --release
```

### Run Migrations

```bash
# Migrations run automatically on startup
# Or manually with sqlx:
sqlx migrate run
```

### Run Tests

```bash
cargo test
```

### Run Service

```bash
# Set environment variables
cp .env.example .env
# Edit .env with your configuration

# Run service
cargo run
```

## Docker Deployment

### Build Image

```bash
docker build -f services/video-conferencing-service/Dockerfile -t video-conferencing-service .
```

### Run with Docker Compose

```bash
docker-compose up -d video-conferencing-service
```

### Ports

- **50052**: gRPC server
- **8082**: WebSocket/HTTP server
- **9092**: Prometheus metrics
- **50000-50100/udp**: WebRTC media ports

## Production Considerations

### GCS Setup (AC8, AC9)

1. Create GCS bucket:

```bash
gsutil mb gs://slate-recordings
```

2. Set up service account with permissions:
   - `storage.objects.create`
   - `storage.objects.get`
   - `storage.objects.delete`

3. Download credentials and set:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
```

### Scaling for 50+ Participants (AC5)

- Use SFU architecture (implemented)
- Deploy multiple instances behind load balancer
- Use Redis for session state sharing
- Configure appropriate resource limits

### Recording Performance (AC9)

- Use SSD storage for temporary recordings
- Optimize GCS upload bandwidth
- Monitor `recording_processing_duration_seconds` metric
- Set up alerts for processing > 25 minutes

### Video Quality Optimization (AC4)

- Monitor `packet_loss_percentage` and `latency_ms`
- Adjust quality thresholds based on network conditions
- Use TURN servers for firewall traversal
- Configure appropriate ICE servers

## Monitoring

### Grafana Dashboards

Access Grafana at http://localhost:3000 (default: admin/admin)

Recommended panels:

- Active sessions and participants
- Video quality distribution (360p/480p/720p/1080p)
- Recording processing time (AC9 compliance)
- WebRTC connection success rate
- Screen sharing usage (AC7)

### Health Checks

```bash
# Service health
curl http://localhost:8082/health

# Metrics
curl http://localhost:9092/metrics
```

## License

Copyright © 2025 Slate Education Platform
