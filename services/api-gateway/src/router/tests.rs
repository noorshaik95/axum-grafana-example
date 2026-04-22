//! Tests for the router module.

use crate::config::RouteConfig;

use super::matcher::RouteMatcher;
use super::pattern::{PathSegment, RoutePattern};
use super::{RequestRouter, RouterError};

// ============================================================================
// Pattern Tests
// ============================================================================

#[test]
fn test_parse_static_pattern() {
    let pattern = RoutePattern::parse("/api/users", "GET");
    assert_eq!(pattern.segments.len(), 2);
    assert!(matches!(pattern.segments[0], PathSegment::Static(_)));
    assert!(matches!(pattern.segments[1], PathSegment::Static(_)));
    assert_eq!(pattern.method, "GET");
}

#[test]
fn test_parse_dynamic_pattern() {
    let pattern = RoutePattern::parse("/api/users/:id", "GET");
    assert_eq!(pattern.segments.len(), 3);
    assert!(matches!(pattern.segments[0], PathSegment::Static(_)));
    assert!(matches!(pattern.segments[1], PathSegment::Static(_)));
    assert!(matches!(pattern.segments[2], PathSegment::Dynamic(_)));
}

#[test]
fn test_is_static() {
    let static_pattern = RoutePattern::parse("/api/users", "GET");
    assert!(static_pattern.is_static());

    let dynamic_pattern = RoutePattern::parse("/api/users/:id", "GET");
    assert!(!dynamic_pattern.is_static());
}

#[test]
fn test_matches_static() {
    let pattern = RoutePattern::parse("/api/users", "GET");

    assert!(pattern.matches("/api/users").is_some());
    assert!(pattern.matches("/api/posts").is_none());
    assert!(pattern.matches("/api/users/123").is_none());
}

#[test]
fn test_matches_dynamic_single_param() {
    let pattern = RoutePattern::parse("/api/users/:id", "GET");

    let params = pattern.matches("/api/users/123").unwrap();
    assert_eq!(params.get("id"), Some(&"123".to_string()));

    assert!(pattern.matches("/api/users").is_none());
    assert!(pattern.matches("/api/posts/123").is_none());
}

#[test]
fn test_matches_dynamic_multiple_params() {
    let pattern = RoutePattern::parse("/api/posts/:post_id/comments/:comment_id", "GET");

    let params = pattern.matches("/api/posts/456/comments/789").unwrap();
    assert_eq!(params.get("post_id"), Some(&"456".to_string()));
    assert_eq!(params.get("comment_id"), Some(&"789".to_string()));
}

#[test]
fn test_method_uppercase() {
    let pattern = RoutePattern::parse("/api/users", "get");
    assert_eq!(pattern.method, "GET");
}

// ============================================================================
// Matcher Tests
// ============================================================================

fn create_matcher_test_routes() -> Vec<RouteConfig> {
    vec![
        RouteConfig {
            path: "/api/users".to_string(),
            method: "GET".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/ListUsers".to_string(),
            ..Default::default()
        },
        RouteConfig {
            path: "/api/users/:id".to_string(),
            method: "GET".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/GetUser".to_string(),
            ..Default::default()
        },
    ]
}

#[test]
fn test_matcher_static_route_matching() {
    let mut matcher = RouteMatcher::new();
    matcher.add_routes(create_matcher_test_routes());

    let result = matcher.match_route("/api/users", "GET").unwrap();
    assert_eq!(result.service.as_ref(), "user-service");
    assert_eq!(result.grpc_method.as_ref(), "user.UserService/ListUsers");
}

#[test]
fn test_matcher_dynamic_route_matching() {
    let mut matcher = RouteMatcher::new();
    matcher.add_routes(create_matcher_test_routes());

    let result = matcher.match_route("/api/users/123", "GET").unwrap();
    assert_eq!(result.service.as_ref(), "user-service");
    assert_eq!(result.grpc_method.as_ref(), "user.UserService/GetUser");
    assert_eq!(result.path_params.get("id"), Some(&"123".to_string()));
}

#[test]
fn test_matcher_route_not_found() {
    let mut matcher = RouteMatcher::new();
    matcher.add_routes(create_matcher_test_routes());

    let result = matcher.match_route("/api/nonexistent", "GET");
    assert!(result.is_err());
}

#[test]
fn test_matcher_clear() {
    let mut matcher = RouteMatcher::new();
    matcher.add_routes(create_matcher_test_routes());
    assert_eq!(matcher.route_count(), 2);

    matcher.clear();
    assert_eq!(matcher.route_count(), 0);
}

// ============================================================================
// RequestRouter Tests
// ============================================================================

fn create_test_routes() -> Vec<RouteConfig> {
    vec![
        RouteConfig {
            path: "/api/users".to_string(),
            method: "GET".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/ListUsers".to_string(),
            ..Default::default()
        },
        RouteConfig {
            path: "/api/users/:id".to_string(),
            method: "GET".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/GetUser".to_string(),
            ..Default::default()
        },
        RouteConfig {
            path: "/api/users/:id".to_string(),
            method: "DELETE".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/DeleteUser".to_string(),
            ..Default::default()
        },
        RouteConfig {
            path: "/api/posts/:post_id/comments/:comment_id".to_string(),
            method: "GET".to_string(),
            service: "post-service".to_string(),
            grpc_method: "post.PostService/GetComment".to_string(),
            ..Default::default()
        },
    ]
}

#[test]
fn test_static_route_matching() {
    let router = RequestRouter::new(create_test_routes());

    let result = router.route("/api/users", "GET").unwrap();
    assert_eq!(result.service.as_ref(), "user-service");
    assert_eq!(result.grpc_method.as_ref(), "user.UserService/ListUsers");
    assert!(result.path_params.is_empty());
}

#[test]
fn test_dynamic_route_single_param() {
    let router = RequestRouter::new(create_test_routes());

    let result = router.route("/api/users/123", "GET").unwrap();
    assert_eq!(result.service.as_ref(), "user-service");
    assert_eq!(result.grpc_method.as_ref(), "user.UserService/GetUser");
    assert_eq!(result.path_params.get("id"), Some(&"123".to_string()));
}

#[test]
fn test_dynamic_route_multiple_params() {
    let router = RequestRouter::new(create_test_routes());

    let result = router.route("/api/posts/456/comments/789", "GET").unwrap();
    assert_eq!(result.service.as_ref(), "post-service");
    assert_eq!(result.grpc_method.as_ref(), "post.PostService/GetComment");
    assert_eq!(result.path_params.get("post_id"), Some(&"456".to_string()));
    assert_eq!(
        result.path_params.get("comment_id"),
        Some(&"789".to_string())
    );
}

#[test]
fn test_method_matching() {
    let router = RequestRouter::new(create_test_routes());

    // GET should match GetUser
    let result = router.route("/api/users/123", "GET").unwrap();
    assert_eq!(result.grpc_method.as_ref(), "user.UserService/GetUser");

    // DELETE should match DeleteUser
    let result = router.route("/api/users/123", "DELETE").unwrap();
    assert_eq!(result.grpc_method.as_ref(), "user.UserService/DeleteUser");
}

#[test]
fn test_case_insensitive_method() {
    let router = RequestRouter::new(create_test_routes());

    // Lowercase method should work
    let result = router.route("/api/users", "get").unwrap();
    assert_eq!(result.service.as_ref(), "user-service");
}

#[test]
fn test_route_not_found() {
    let router = RequestRouter::new(create_test_routes());

    let result = router.route("/api/nonexistent", "GET");
    assert!(result.is_err());
    assert!(matches!(
        result.unwrap_err(),
        RouterError::RouteNotFound { .. }
    ));
}

#[test]
fn test_wrong_method() {
    let router = RequestRouter::new(create_test_routes());

    // POST not configured for /api/users
    let result = router.route("/api/users", "POST");
    assert!(result.is_err());
}

#[test]
fn test_trailing_slash_handling() {
    let router = RequestRouter::new(create_test_routes());

    // Without trailing slash should work
    let result = router.route("/api/users", "GET");
    assert!(result.is_ok());
}

#[test]
fn test_segment_count_mismatch() {
    let router = RequestRouter::new(create_test_routes());

    // Too many segments
    let result = router.route("/api/users/123/extra", "GET");
    assert!(result.is_err());

    // Too few segments for dynamic route
    let result = router.route("/api/posts/456/comments", "GET");
    assert!(result.is_err());
}

#[test]
fn test_update_routes() {
    let mut router = RequestRouter::new(create_test_routes());

    // Initial route count
    assert_eq!(router.route_count(), 4);

    // Update with new routes
    let new_routes = vec![RouteConfig {
        path: "/api/products".to_string(),
        method: "GET".to_string(),
        service: "product-service".to_string(),
        grpc_method: "product.ProductService/ListProducts".to_string(),
        ..Default::default()
    }];

    router.update_routes(new_routes);

    // Should have only 1 route now
    assert_eq!(router.route_count(), 1);

    // Old routes should not work
    let result = router.route("/api/users", "GET");
    assert!(result.is_err());

    // New route should work
    let result = router.route("/api/products", "GET").unwrap();
    assert_eq!(result.service.as_ref(), "product-service");
}

#[test]
fn test_get_routes_for_service() {
    let router = RequestRouter::new(create_test_routes());

    // Get routes for user-service
    let user_routes = router.get_routes_for_service("user-service");
    assert_eq!(user_routes.len(), 3);

    // Get routes for post-service
    let post_routes = router.get_routes_for_service("post-service");
    assert_eq!(post_routes.len(), 1);

    // Get routes for non-existent service
    let empty_routes = router.get_routes_for_service("nonexistent-service");
    assert_eq!(empty_routes.len(), 0);
}

#[test]
fn test_route_count() {
    let router = RequestRouter::new(create_test_routes());
    assert_eq!(router.route_count(), 4);

    let empty_router = RequestRouter::new(vec![]);
    assert_eq!(empty_router.route_count(), 0);
}

#[test]
fn test_get_all_routes() {
    let router = RequestRouter::new(create_test_routes());

    let all_routes = router.get_all_routes();
    assert_eq!(all_routes.len(), 4);

    // Verify all routes are present
    let services: Vec<String> = all_routes.iter().map(|r| r.service.clone()).collect();
    assert!(services.contains(&"user-service".to_string()));
    assert!(services.contains(&"post-service".to_string()));
}

#[test]
fn test_service_route_map_tracking() {
    let routes = vec![
        RouteConfig {
            path: "/api/users".to_string(),
            method: "GET".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/ListUsers".to_string(),
            ..Default::default()
        },
        RouteConfig {
            path: "/api/users/:id".to_string(),
            method: "GET".to_string(),
            service: "user-service".to_string(),
            grpc_method: "user.UserService/GetUser".to_string(),
            ..Default::default()
        },
        RouteConfig {
            path: "/api/products".to_string(),
            method: "GET".to_string(),
            service: "product-service".to_string(),
            grpc_method: "product.ProductService/ListProducts".to_string(),
            ..Default::default()
        },
    ];

    let router = RequestRouter::new(routes);

    // Verify service route map
    let user_routes = router.get_routes_for_service("user-service");
    assert_eq!(user_routes.len(), 2);

    let product_routes = router.get_routes_for_service("product-service");
    assert_eq!(product_routes.len(), 1);
}
