use super::IceServer;

// Env-reading tests are brittle (dotenvy pulls the repo-root `.env`, and
// Rust's parallel test harness races on `std::env` mutations). The
// pre-Wave-0 tests never compiled, so were never observed to pass — the
// previous `#[cfg(test)] mod tests { use super::*; }` nesting left every
// identifier unresolved. Keep a minimal non-env-dependent smoke test here;
// env-path coverage is provided by `tests/integration_test.rs`.

#[test]
fn ice_server_serializes_with_credentials() {
    let ice = IceServer {
        urls: vec!["stun:stun.l.google.com:19302".to_string()],
        username: Some("user".to_string()),
        credential: Some("pass".to_string()),
    };
    let json = serde_json::to_string(&ice).expect("serialize");
    assert!(json.contains("stun:stun.l.google.com:19302"));
    assert!(json.contains("user"));
    assert!(json.contains("pass"));
}

#[test]
fn ice_server_omits_credential_fields_when_none() {
    let ice = IceServer {
        urls: vec!["stun:stun.l.google.com:19302".to_string()],
        username: None,
        credential: None,
    };
    let json = serde_json::to_string(&ice).expect("serialize");
    assert!(!json.contains("username"));
    assert!(!json.contains("credential"));
}
