// Integration tests for user-auth-service via the API gateway
//
// These tests verify that the gateway can successfully communicate with
// the user-auth-service for authentication and user management operations.
//
// Requirements:
//   docker-compose up api-gateway user-auth-service postgres redis
//
// Run with:
//   cargo test -- --ignored

#[cfg(test)]
mod tests {
    use serde_json::{json, Value};

    const GATEWAY_URL: &str = "http://localhost:8080";

    async fn is_gateway_available() -> bool {
        reqwest::get(format!("{}/health/live", GATEWAY_URL))
            .await
            .map(|r| r.status().is_success())
            .unwrap_or(false)
    }

    fn unique_email() -> String {
        let ts = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_millis();
        format!("rust_test_{}@example.com", ts)
    }

    /// Register a user and return the response JSON.
    async fn register_user(email: &str, password: &str) -> (u16, Value) {
        let client = reqwest::Client::new();
        let resp = client
            .post(format!("{}/api/auth/register", GATEWAY_URL))
            .json(&json!({
                "email": email,
                "password": password,
                "first_name": "Test",
                "last_name": "User",
                "phone": "+15551234567"
            }))
            .send()
            .await
            .expect("register request failed");

        let status = resp.status().as_u16();
        let body: Value = resp.json().await.unwrap_or(json!({}));
        (status, body)
    }

    /// Login and return the response JSON.
    async fn login_user(email: &str, password: &str) -> (u16, Value) {
        let client = reqwest::Client::new();
        let resp = client
            .post(format!("{}/api/auth/login", GATEWAY_URL))
            .json(&json!({
                "email": email,
                "password": password
            }))
            .send()
            .await
            .expect("login request failed");

        let status = resp.status().as_u16();
        let body: Value = resp.json().await.unwrap_or(json!({}));
        (status, body)
    }

    #[tokio::test]
    #[ignore]
    async fn test_user_registration_flow() {
        if !is_gateway_available().await {
            eprintln!("Skipping: gateway not available");
            return;
        }

        let email = unique_email();
        let (status, body) = register_user(&email, "testpassword123").await;

        assert!(
            status == 200 || status == 201,
            "registration failed with status {}: {:?}",
            status,
            body
        );
        assert!(
            body.get("access_token").is_some() || body.get("email").is_some(),
            "expected access_token or email in response: {:?}",
            body
        );
    }

    #[tokio::test]
    #[ignore]
    async fn test_user_login_flow() {
        if !is_gateway_available().await {
            eprintln!("Skipping: gateway not available");
            return;
        }

        // Register a user first, then login
        let email = unique_email();
        let (reg_status, _) = register_user(&email, "testpassword123").await;
        assert!(
            reg_status == 200 || reg_status == 201,
            "setup: registration failed with {}",
            reg_status
        );

        let (status, body) = login_user(&email, "testpassword123").await;

        assert_eq!(status, 200, "login failed with status {}: {:?}", status, body);
        assert!(
            body.get("access_token").is_some(),
            "expected access_token in login response: {:?}",
            body
        );
    }

    #[tokio::test]
    #[ignore]
    async fn test_list_users_with_auth() {
        if !is_gateway_available().await {
            eprintln!("Skipping: gateway not available");
            return;
        }

        // Login as admin
        let (status, body) = login_user("admin@example.com", "admin123").await;
        if status != 200 {
            eprintln!("Skipping: admin login failed ({})", status);
            return;
        }

        let token = body["access_token"].as_str().expect("missing access_token");

        let client = reqwest::Client::new();
        let resp = client
            .get(format!("{}/api/users", GATEWAY_URL))
            .header("Authorization", format!("Bearer {}", token))
            .send()
            .await
            .expect("list users request failed");

        let resp_status = resp.status().as_u16();
        assert_eq!(
            resp_status, 200,
            "list users returned {} (expected 200)",
            resp_status
        );
    }

    #[tokio::test]
    #[ignore]
    async fn test_create_user_requires_auth() {
        if !is_gateway_available().await {
            eprintln!("Skipping: gateway not available");
            return;
        }

        let create_payload = json!({
            "email": unique_email(),
            "password": "password123",
            "first_name": "New",
            "last_name": "User",
            "phone": "+15559876543",
            "roles": ["user"]
        });

        let client = reqwest::Client::new();

        // Without auth -- should get 401
        let resp = client
            .post(format!("{}/api/users", GATEWAY_URL))
            .json(&create_payload)
            .send()
            .await
            .expect("create user (no auth) request failed");

        let no_auth_status = resp.status().as_u16();
        assert!(
            no_auth_status == 401 || no_auth_status == 403,
            "create user without auth returned {} (expected 401/403)",
            no_auth_status
        );

        // With admin auth -- should succeed
        let (login_status, login_body) = login_user("admin@example.com", "admin123").await;
        if login_status != 200 {
            eprintln!("Skipping authed create: admin login failed ({})", login_status);
            return;
        }

        let token = login_body["access_token"].as_str().unwrap();
        let resp = client
            .post(format!("{}/api/users", GATEWAY_URL))
            .header("Authorization", format!("Bearer {}", token))
            .json(&create_payload)
            .send()
            .await
            .expect("create user (authed) request failed");

        let authed_status = resp.status().as_u16();
        assert!(
            authed_status == 200 || authed_status == 201,
            "create user with admin token returned {} (expected 200/201)",
            authed_status
        );
    }

    #[tokio::test]
    #[ignore]
    async fn test_jwt_token_validation() {
        if !is_gateway_available().await {
            eprintln!("Skipping: gateway not available");
            return;
        }

        // Register + login to get a valid token
        let email = unique_email();
        let (reg_status, _) = register_user(&email, "testpassword123").await;
        assert!(reg_status == 200 || reg_status == 201);

        let (_, login_body) = login_user(&email, "testpassword123").await;
        let token = login_body["access_token"]
            .as_str()
            .expect("missing access_token");

        let client = reqwest::Client::new();

        // Valid token should work
        let resp = client
            .get(format!("{}/api/users/profile", GATEWAY_URL))
            .header("Authorization", format!("Bearer {}", token))
            .send()
            .await
            .expect("profile request failed");

        assert_eq!(resp.status().as_u16(), 200, "valid token should grant access");

        // Invalid token should be rejected
        let resp = client
            .get(format!("{}/api/users/profile", GATEWAY_URL))
            .header("Authorization", "Bearer invalid.token.data")
            .send()
            .await
            .expect("invalid token request failed");

        let invalid_status = resp.status().as_u16();
        assert!(
            invalid_status == 401 || invalid_status == 403,
            "invalid token returned {} (expected 401/403)",
            invalid_status
        );
    }

    #[tokio::test]
    #[ignore]
    async fn test_grpc_error_mapping() {
        if !is_gateway_available().await {
            eprintln!("Skipping: gateway not available");
            return;
        }

        let client = reqwest::Client::new();

        // UNAUTHENTICATED -> 401: access protected endpoint without token
        let resp = client
            .get(format!("{}/api/users/profile", GATEWAY_URL))
            .send()
            .await
            .expect("no-auth request failed");
        let status = resp.status().as_u16();
        assert!(
            status == 401 || status == 403,
            "no-auth should map to 401/403, got {}",
            status
        );

        // NOT_FOUND -> 404: non-existent resource
        // Login first to avoid 401
        let (login_status, login_body) = login_user("admin@example.com", "admin123").await;
        if login_status == 200 {
            let token = login_body["access_token"].as_str().unwrap();
            let resp = client
                .get(format!(
                    "{}/api/users/00000000-0000-0000-0000-000000000000",
                    GATEWAY_URL
                ))
                .header("Authorization", format!("Bearer {}", token))
                .send()
                .await
                .expect("not-found request failed");
            let status = resp.status().as_u16();
            assert!(
                status == 404 || status == 400,
                "non-existent user should map to 404, got {}",
                status
            );
        }

        // INVALID_ARGUMENT -> 400: malformed request
        let resp = client
            .post(format!("{}/api/auth/login", GATEWAY_URL))
            .header("Content-Type", "application/json")
            .body("not-json")
            .send()
            .await
            .expect("bad-json request failed");
        let status = resp.status().as_u16();
        assert!(
            status == 400 || status == 422,
            "bad JSON should map to 400, got {}",
            status
        );
    }
}
