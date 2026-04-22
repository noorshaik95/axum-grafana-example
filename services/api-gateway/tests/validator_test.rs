use api_gateway::config::RouteConfig;
use api_gateway::discovery::{DiscoveryError, RouteValidator};

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
fn test_no_duplicates() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users/:id", "GET", "user.UserService/GetUser"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
    ];

    let result = RouteValidator::check_duplicates(&routes);
    assert!(result.is_ok());
}

#[test]
fn test_duplicate_detection() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users", "GET", "user.UserService/GetAllUsers"),
    ];

    let result = RouteValidator::check_duplicates(&routes);
    assert!(result.is_err());

    if let Err(DiscoveryError::DuplicateRoute {
        method1,
        method2,
        http_route,
    }) = result
    {
        assert_eq!(method1, "user.UserService/ListUsers");
        assert_eq!(method2, "user.UserService/GetAllUsers");
        assert_eq!(http_route, "GET /api/users");
    } else {
        panic!("Expected DuplicateRoute error");
    }
}

#[test]
fn test_deduplicate_routes() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users", "GET", "user.UserService/GetAllUsers"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
    ];

    let result = RouteValidator::deduplicate_routes(routes);
    assert_eq!(result.len(), 2);
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
    assert_eq!(result[1].grpc_method, "post.PostService/ListPosts");
}

#[test]
fn test_different_methods_not_duplicate() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users", "POST", "user.UserService/CreateUser"),
    ];

    let result = RouteValidator::check_duplicates(&routes);
    assert!(result.is_ok());
}

#[test]
fn test_different_paths_not_duplicate() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users/:id", "GET", "user.UserService/GetUser"),
    ];

    let result = RouteValidator::check_duplicates(&routes);
    assert!(result.is_ok());
}

#[test]
fn test_deduplicate_keeps_first() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
        create_test_route("/api/users", "GET", "user.UserService/GetAllUsers"),
        create_test_route(
            "/api/comments",
            "GET",
            "comment.CommentService/ListComments",
        ),
    ];

    let result = RouteValidator::deduplicate_routes(routes);
    assert_eq!(result.len(), 3);

    // First occurrence of /api/users should be kept
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
    assert_eq!(result[1].grpc_method, "post.PostService/ListPosts");
    assert_eq!(result[2].grpc_method, "comment.CommentService/ListComments");
}

#[test]
fn test_multiple_duplicates() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users", "GET", "user.UserService/GetAllUsers"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
        create_test_route("/api/posts", "GET", "post.PostService/GetAllPosts"),
    ];

    let result = RouteValidator::check_duplicates(&routes);
    assert!(result.is_err());

    // Should report the first duplicate found
    if let Err(DiscoveryError::DuplicateRoute { http_route, .. }) = result {
        assert_eq!(http_route, "GET /api/users");
    } else {
        panic!("Expected DuplicateRoute error");
    }
}

#[test]
fn test_deduplicate_multiple_duplicates() {
    let routes = vec![
        create_test_route("/api/users", "GET", "user.UserService/ListUsers"),
        create_test_route("/api/users", "GET", "user.UserService/GetAllUsers"),
        create_test_route("/api/posts", "GET", "post.PostService/ListPosts"),
        create_test_route("/api/posts", "GET", "post.PostService/GetAllPosts"),
    ];

    let result = RouteValidator::deduplicate_routes(routes);
    assert_eq!(result.len(), 2);
    assert_eq!(result[0].grpc_method, "user.UserService/ListUsers");
    assert_eq!(result[1].grpc_method, "post.PostService/ListPosts");
}
