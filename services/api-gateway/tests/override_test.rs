use api_gateway::config::{RouteConfig, RouteOverride};
use api_gateway::discovery::OverrideHandler;

fn create_test_route(path: &str, method: &str, grpc_method: &str) -> RouteConfig {
    RouteConfig {
        path: path.to_string(),
        method: method.to_string(),
        service: "test-service".to_string(),
        grpc_method: grpc_method.to_string(),
        ..Default::default()
    }
}

#[test]
fn test_no_overrides() {
    let routes = vec![create_test_route(
        "/api/users",
        "GET",
        "user.UserService/ListUsers",
    )];

    let result = OverrideHandler::apply_overrides(routes.clone(), &[]);
    assert_eq!(result.len(), 1);
    assert_eq!(result[0].path, "/api/users");
    assert_eq!(result[0].method, "GET");
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
}

#[test]
fn test_path_override() {
    let routes = vec![create_test_route(
        "/api/users",
        "GET",
        "user.UserService/ListUsers",
    )];

    let overrides = vec![RouteOverride {
        grpc_method: "user.UserService/ListUsers".to_string(),
        http_path: Some("/v1/users".to_string()),
        http_method: None,
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 1);
    assert_eq!(result[0].path, "/v1/users");
    assert_eq!(result[0].method, "GET");
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
}

#[test]
fn test_method_override() {
    let routes = vec![create_test_route(
        "/api/users",
        "GET",
        "user.UserService/ListUsers",
    )];

    let overrides = vec![RouteOverride {
        grpc_method: "user.UserService/ListUsers".to_string(),
        http_path: None,
        http_method: Some("POST".to_string()),
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 1);
    assert_eq!(result[0].path, "/api/users");
    assert_eq!(result[0].method, "POST");
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
}

#[test]
fn test_full_override() {
    let routes = vec![create_test_route(
        "/api/users",
        "GET",
        "user.UserService/ListUsers",
    )];

    let overrides = vec![RouteOverride {
        grpc_method: "user.UserService/ListUsers".to_string(),
        http_path: Some("/v1/all-users".to_string()),
        http_method: Some("POST".to_string()),
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 1);
    assert_eq!(result[0].path, "/v1/all-users");
    assert_eq!(result[0].method, "POST");
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
}

#[test]
fn test_partial_override() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
    ];

    let overrides = vec![RouteOverride {
        grpc_method: "user.UserService/ListUsers".to_string(),
        http_path: Some("/v1/users".to_string()),
        http_method: None,
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 2);

    // First route should be overridden
    assert_eq!(result[0].path, "/v1/users");
    assert_eq!(result[0].method, "GET");
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");

    // Second route should remain unchanged
    assert_eq!(result[1].path, "/api/posts");
    assert_eq!(result[1].method, "GET");
    assert_eq!(result[1].grpc_method, "post.PostService/ListPosts");
}

#[test]
fn test_multiple_overrides() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
        create_test_route(
            "/api/comments",
            "GET",
            "comment.CommentService/ListComments",
        ),
    ];

    let overrides = vec![
        RouteOverride {
            grpc_method: "user.UserService/ListUsers".to_string(),
            http_path: Some("/v1/users".to_string()),
            http_method: None,
            service: None,
            ..Default::default()
        },
        RouteOverride {
            grpc_method: "post.PostService/ListPosts".to_string(),
            http_path: None,
            http_method: Some("POST".to_string()),
            service: None,
            ..Default::default()
        },
    ];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 3);

    // First route - path overridden
    assert_eq!(result[0].path, "/v1/users");
    assert_eq!(result[0].method, "GET");

    // Second route - method overridden
    assert_eq!(result[1].path, "/api/posts");
    assert_eq!(result[1].method, "POST");

    // Third route - unchanged
    assert_eq!(result[2].path, "/api/comments");
    assert_eq!(result[2].method, "GET");
}

#[test]
fn test_override_nonexistent_route() {
    let routes = vec![create_test_route(
        "/api/users",
        "GET",
        "user.UserService/ListUsers",
    )];

    let overrides = vec![RouteOverride {
        grpc_method: "post.PostService/ListPosts".to_string(),
        http_path: Some("/v1/posts".to_string()),
        http_method: None,
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 1);

    // Route should remain unchanged since override doesn't match
    assert_eq!(result[0].path, "/api/users");
    assert_eq!(result[0].method, "GET");
}

#[test]
fn test_empty_routes_with_overrides() {
    let routes: Vec<RouteConfig> = vec![];
    let overrides = vec![RouteOverride {
        grpc_method: "user.UserService/ListUsers".to_string(),
        http_path: Some("/v1/users".to_string()),
        http_method: None,
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 0);
}

#[test]
fn test_preserve_service_name() {
    let routes = vec![create_test_route(
        "/api/users",
        "GET",
        "user.UserService/ListUsers",
    )];

    let overrides = vec![RouteOverride {
        grpc_method: "user.UserService/ListUsers".to_string(),
        http_path: Some("/v1/users".to_string()),
        http_method: Some("POST".to_string()),
        service: None,
        ..Default::default()
    }];

    let result = OverrideHandler::apply_overrides(routes, &overrides);
    assert_eq!(result.len(), 1);

    // Service name should be preserved
    assert_eq!(result[0].service, "test-service");
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
}
