# Video Conferencing Service Tests

This directory contains integration tests for the video conferencing service.

## Test Coverage

### Acceptance Criteria Tests

All 9 acceptance criteria are tested:

#### AC1: Schedule video session (date, time, duration)

- `test_schedule_request_validation_ac1` - Validates scheduling parameters

#### AC2: Students receive calendar invitation

- `test_calendar_ics_generation_ac2` - Tests iCalendar format generation

#### AC3: Join button enabled 10 minutes before start

- `test_config_join_window_ac3` - Tests configuration
- `test_session_join_eligibility_timing_ac3` - Tests join timing logic

#### AC4: Video quality adapts to bandwidth (360p-1080p)

- `test_video_quality_settings_ac4` - Tests quality settings
- `test_video_quality_levels_ac4` - Tests all quality levels
- `test_update_video_quality_request_ac4` - Tests quality update
- `test_video_quality_adaptive_bitrate_ac4` - Tests adaptive logic

#### AC5: Supports 50 concurrent participants

- `test_session_settings_ac5` - Tests max participants setting
- `test_config_max_participants_ac5` - Tests configuration

#### AC6: Instructor mutes/unmutes participants

- `test_mute_participant_request_ac6` - Tests mute request structure

#### AC7: Screen sharing enabled

- `test_session_settings_ac7` - Tests screen share setting
- `test_screen_share_request_ac7` - Tests screen share request

#### AC8: Session auto-records to GCS

- `test_session_settings_ac8` - Tests auto-record setting
- `test_recording_request_ac8` - Tests recording request

#### AC9: Recording available within 30 minutes

- `test_recording_status_types_ac9` - Tests recording status types
- `test_config_recording_timeout_ac9` - Tests 30-minute timeout

## Running Tests

### Run All Tests

```bash
cargo test
```

### Run Only Unit Tests

```bash
cargo test --lib
```

### Run Only Integration Tests

```bash
cargo test --test integration_test
```

### Run Specific Test

```bash
cargo test test_schedule_request_validation_ac1
```

### Run Tests with Output

```bash
cargo test -- --nocapture
```

### Run Tests in Parallel

```bash
cargo test -- --test-threads=4
```

## Test Organization

- **Unit Tests**: Located in `src/*/tests.rs` files
  - `src/models/tests.rs` - Model and helper function tests
  - `src/config/tests.rs` - Configuration tests
  - `src/webrtc/tests.rs` - WebRTC signaling tests

- **Integration Tests**: Located in `tests/` directory
  - `integration_test.rs` - End-to-end acceptance criteria tests

## Coverage Report

To generate a coverage report:

```bash
cargo install cargo-tarpaulin
cargo tarpaulin --out Html
```

## Continuous Integration

Tests are run automatically on:

- Every commit
- Pull requests
- Before deployment

## Test Data

Tests use randomly generated UUIDs and timestamps to ensure isolation and repeatability.

## Mocking

For database tests, consider using:

- In-memory SQLite
- Testcontainers for PostgreSQL
- Mock repositories

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Tests should clean up environment variables
3. **Assertions**: Use descriptive assertion messages
4. **Naming**: Test names should clearly indicate what they test
5. **Documentation**: Include AC reference in test names
