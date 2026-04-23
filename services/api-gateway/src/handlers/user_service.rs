use crate::handlers::GatewayError;
use crate::proto::user::user_service_client::UserServiceClient;
use crate::proto::user::*;
use tonic::transport::Channel;
use tracing::{debug, error};

/// Map a tonic `Status` into a `GatewayError` that preserves the gRPC code.
/// Used by every typed user-service handler so login/register/validate
/// surface NotFound→404, Unauthenticated→401, etc. instead of collapsing
/// to 502 BAD_GATEWAY (which also spuriously trips the circuit breaker).
fn status_to_gateway_error(context: &str, status: tonic::Status) -> GatewayError {
    GatewayError::GrpcStatus {
        code: status.code(),
        message: format!("{}: {}", context, status.message()),
    }
}

/// Inject trace context into gRPC request metadata
fn inject_trace_context<T>(mut request: tonic::Request<T>) -> tonic::Request<T> {
    use opentelemetry::propagation::Injector;
    use tracing_opentelemetry::OpenTelemetrySpanExt;

    struct MetadataInjector<'a>(&'a mut tonic::metadata::MetadataMap);

    impl<'a> Injector for MetadataInjector<'a> {
        fn set(&mut self, key: &str, value: String) {
            if let Ok(metadata_key) = tonic::metadata::MetadataKey::from_bytes(key.as_bytes()) {
                if let Ok(metadata_value) = tonic::metadata::MetadataValue::try_from(&value) {
                    self.0.insert(metadata_key, metadata_value);
                    debug!(key = %key, "Injected trace header into gRPC metadata");
                }
            }
        }
    }

    // CRITICAL: Extract the OpenTelemetry context from the current tracing span
    // This ensures we propagate the correct trace context across service boundaries
    let current_span = tracing::Span::current();
    let context = current_span.context();

    // Inject the context into gRPC metadata using the global propagator
    let mut injector = MetadataInjector(request.metadata_mut());
    opentelemetry::global::get_text_map_propagator(|propagator| {
        propagator.inject_context(&context, &mut injector);
    });

    request
}

/// Serialize user data to JSON.
fn serialize_user(user: Option<crate::proto::user::User>) -> Option<serde_json::Value> {
    user.map(|u| {
        serde_json::json!({
            "id": u.id,
            "email": u.email,
            "first_name": u.first_name,
            "last_name": u.last_name,
            "phone": u.phone,
            "roles": u.roles,
            "is_active": u.is_active,
        })
    })
}

/// Serialize authentication response to JSON bytes.
fn serialize_auth_response(
    access_token: String,
    refresh_token: String,
    expires_in: i64,
    user: Option<crate::proto::user::User>,
) -> Result<Vec<u8>, GatewayError> {
    let json_response = serde_json::json!({
        "access_token": access_token,
        "refresh_token": refresh_token,
        "expires_in": expires_in,
        "user": serialize_user(user)
    });

    serde_json::to_vec(&json_response)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to serialize response: {}", e)))
}

/// Handle Register method.
async fn handle_register(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = RegisterRequest {
        email: json_value["email"].as_str().unwrap_or("").to_string(),
        password: json_value["password"].as_str().unwrap_or("").to_string(),
        first_name: json_value["first_name"].as_str().unwrap_or("").to_string(),
        last_name: json_value["last_name"].as_str().unwrap_or("").to_string(),
        phone: json_value["phone"].as_str().unwrap_or("").to_string(),
    };

    debug!(email = %request.email, "Registering user");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client.register(grpc_request).await.map_err(|e| {
        error!(error = %e, "Register call failed");
        status_to_gateway_error("Register failed", e)
    })?;

    let resp = response.into_inner();
    let json_response = serde_json::json!({
        "access_token": resp.access_token,
        "refresh_token": resp.refresh_token,
        "user": serialize_user(resp.user)
    });

    serde_json::to_vec(&json_response)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to serialize response: {}", e)))
}

/// Handle Login method.
async fn handle_login(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = LoginRequest {
        email: json_value["email"].as_str().unwrap_or("").to_string(),
        password: json_value["password"].as_str().unwrap_or("").to_string(),
    };

    debug!(email = %request.email, "Logging in user");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client.login(grpc_request).await.map_err(|e| {
        error!(error = %e, "Login call failed");
        status_to_gateway_error("Login failed", e)
    })?;

    let resp = response.into_inner();
    serialize_auth_response(
        resp.access_token,
        resp.refresh_token,
        resp.expires_in,
        resp.user,
    )
}

/// Handle ValidateToken method.
async fn handle_validate_token(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = ValidateTokenRequest {
        token: json_value["token"].as_str().unwrap_or("").to_string(),
    };

    debug!("Validating token");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client.validate_token(grpc_request).await.map_err(|e| {
        error!(error = %e, "ValidateToken call failed");
        status_to_gateway_error("ValidateToken failed", e)
    })?;

    let resp = response.into_inner();
    let json_response = serde_json::json!({
        "valid": resp.valid,
        "user_id": resp.user_id,
        "roles": resp.roles,
        "error": resp.error,
    });

    serde_json::to_vec(&json_response)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to serialize response: {}", e)))
}

/// Handle OAuth authorization URL request.
async fn handle_oauth_auth_url(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = OAuthAuthRequest {
        provider: json_value["provider"].as_str().unwrap_or("").to_string(),
    };

    debug!(provider = %request.provider, "Getting OAuth authorization URL");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client
        .get_o_auth_authorization_url(grpc_request)
        .await
        .map_err(|e| {
            error!(error = %e, "GetOAuthAuthorizationURL call failed");
            status_to_gateway_error("GetOAuthAuthorizationURL failed", e)
        })?;

    let resp = response.into_inner();
    let json_response = serde_json::json!({
        "authorization_url": resp.authorization_url,
        "state": resp.state,
    });

    serde_json::to_vec(&json_response)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to serialize response: {}", e)))
}

/// Handle OAuth callback.
async fn handle_oauth_callback(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = OAuthCallbackRequest {
        provider: json_value["provider"].as_str().unwrap_or("").to_string(),
        code: json_value["code"].as_str().unwrap_or("").to_string(),
        state: json_value["state"].as_str().unwrap_or("").to_string(),
    };

    debug!(provider = %request.provider, "Handling OAuth callback");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client
        .handle_o_auth_callback(grpc_request)
        .await
        .map_err(|e| {
            error!(error = %e, "HandleOAuthCallback call failed");
            status_to_gateway_error("HandleOAuthCallback failed", e)
        })?;

    let resp = response.into_inner();
    serialize_auth_response(
        resp.access_token,
        resp.refresh_token,
        resp.expires_in,
        resp.user,
    )
}

/// Handle SAML auth request.
async fn handle_saml_auth_request(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = SamlAuthRequest {
        organization_id: json_value["organization_id"]
            .as_str()
            .unwrap_or("")
            .to_string(),
    };

    debug!(organization_id = %request.organization_id, "Getting SAML auth request");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client
        .get_saml_auth_request(grpc_request)
        .await
        .map_err(|e| {
            error!(error = %e, "GetSAMLAuthRequest call failed");
            status_to_gateway_error("GetSAMLAuthRequest failed", e)
        })?;

    let resp = response.into_inner();
    let json_response = serde_json::json!({
        "saml_request": resp.saml_request,
        "sso_url": resp.sso_url,
    });

    serde_json::to_vec(&json_response)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to serialize response: {}", e)))
}

/// Handle SAML assertion.
async fn handle_saml_assertion(
    client: &mut UserServiceClient<Channel>,
    json_value: &serde_json::Value,
) -> Result<Vec<u8>, GatewayError> {
    let request = SamlAssertionRequest {
        saml_response: json_value["saml_response"]
            .as_str()
            .unwrap_or("")
            .to_string(),
    };

    debug!("Handling SAML assertion");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client
        .handle_saml_assertion(grpc_request)
        .await
        .map_err(|e| {
            error!(error = %e, "HandleSAMLAssertion call failed");
            status_to_gateway_error("HandleSAMLAssertion failed", e)
        })?;

    let resp = response.into_inner();
    serialize_auth_response(
        resp.access_token,
        resp.refresh_token,
        resp.expires_in,
        resp.user,
    )
}

/// Handle SAML metadata request.
async fn handle_saml_metadata(
    client: &mut UserServiceClient<Channel>,
) -> Result<Vec<u8>, GatewayError> {
    let request = SamlMetadataRequest {};

    debug!("Getting SAML metadata");

    let grpc_request = inject_trace_context(tonic::Request::new(request));
    let response = client.get_saml_metadata(grpc_request).await.map_err(|e| {
        error!(error = %e, "GetSAMLMetadata call failed");
        status_to_gateway_error("GetSAMLMetadata failed", e)
    })?;

    let resp = response.into_inner();
    let json_response = serde_json::json!({
        "metadata_xml": resp.metadata_xml,
    });

    serde_json::to_vec(&json_response)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to serialize response: {}", e)))
}

/// Handle user service gRPC calls with proper protobuf types
#[tracing::instrument(name = "call_user_service", skip(channel, json_payload), fields(grpc.service = "user.UserService", grpc.method = %method))]
pub async fn call_user_service(
    channel: Channel,
    method: &str,
    json_payload: Vec<u8>,
) -> Result<Vec<u8>, GatewayError> {
    debug!(
        method = %method,
        payload_size = json_payload.len(),
        "Calling user service with typed client"
    );

    let mut client = UserServiceClient::new(channel);

    let json_value: serde_json::Value = serde_json::from_slice(&json_payload)
        .map_err(|e| GatewayError::ConversionError(format!("Failed to parse JSON: {}", e)))?;

    match method {
        "Register" => handle_register(&mut client, &json_value).await,
        "Login" => handle_login(&mut client, &json_value).await,
        "ValidateToken" => handle_validate_token(&mut client, &json_value).await,
        "GetOAuthAuthorizationURL" => handle_oauth_auth_url(&mut client, &json_value).await,
        "HandleOAuthCallback" => handle_oauth_callback(&mut client, &json_value).await,
        "GetSAMLAuthRequest" => handle_saml_auth_request(&mut client, &json_value).await,
        "HandleSAMLAssertion" => handle_saml_assertion(&mut client, &json_value).await,
        "GetSAMLMetadata" => handle_saml_metadata(&mut client).await,
        _ => {
            error!(method = %method, "Unsupported user service method");
            Err(GatewayError::ConversionError(format!(
                "Unsupported method: {}",
                method
            )))
        }
    }
}
