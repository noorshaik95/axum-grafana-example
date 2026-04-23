//! Route matching logic.
//!
//! Handles matching incoming requests against configured routes.

use std::collections::HashMap;
use tracing::{debug, info, warn};

use crate::config::RouteConfig;

use super::decision::{RouterError, RoutingDecision};
use super::pattern::RoutePattern;

/// Key for looking up routes in the routing table.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub(crate) struct RouteKey {
    pub path_pattern: String,
    pub method: String,
}

/// Performs route matching for incoming requests.
pub struct RouteMatcher {
    /// Static routes for O(1) lookup
    pub static_routes: HashMap<RouteKey, RouteConfig>,
    /// Dynamic routes requiring pattern matching
    pub dynamic_routes: Vec<(RoutePattern, RouteConfig)>,
}

impl RouteMatcher {
    /// Create a new route matcher.
    pub fn new() -> Self {
        Self {
            static_routes: HashMap::new(),
            dynamic_routes: Vec::new(),
        }
    }

    /// Add routes to the matcher.
    pub fn add_routes(&mut self, routes: Vec<RouteConfig>) {
        for route in routes {
            let pattern = RoutePattern::parse(&route.path, &route.method);

            if pattern.is_static() {
                // Static routes go into the hash map for O(1) lookup
                let key = RouteKey {
                    path_pattern: route.path.clone(),
                    method: route.method.clone(),
                };
                self.static_routes.insert(key, route);
            } else {
                // Dynamic routes need pattern matching
                self.dynamic_routes.push((pattern, route));
            }
        }
    }

    /// Clear all routes.
    pub fn clear(&mut self) {
        self.static_routes.clear();
        self.dynamic_routes.clear();
    }

    /// Get the total number of routes.
    pub fn route_count(&self) -> usize {
        self.static_routes.len() + self.dynamic_routes.len()
    }

    /// Match an incoming request to a route.
    pub fn match_route(&self, path: &str, method: &str) -> Result<RoutingDecision, RouterError> {
        info!(
            path = %path,
            method = %method,
            static_routes = self.static_routes.len(),
            dynamic_routes = self.dynamic_routes.len(),
            "🔍 ROUTER: Attempting to route request"
        );

        // First try exact match for static routes (fast path)
        if let Some(decision) = self.try_static_match(path, method) {
            return Ok(decision);
        }

        debug!(
            path = %path,
            method = %method,
            "🔍 ROUTER: No static route match, trying dynamic routes"
        );

        // Try dynamic routes with pattern matching
        if let Some(decision) = self.try_dynamic_match(path, method) {
            return Ok(decision);
        }

        warn!(
            path = %path,
            method = %method,
            "❌ ROUTER: No route found"
        );

        Err(RouterError::RouteNotFound {
            path: path.to_string(),
            method: method.to_string(),
        })
    }

    /// Try to match against static routes.
    fn try_static_match(&self, path: &str, method: &str) -> Option<RoutingDecision> {
        let key = RouteKey {
            path_pattern: path.to_string(),
            method: method.to_uppercase(),
        };

        self.static_routes.get(&key).map(|route| {
            info!(
                path = %path,
                method = %method,
                service = %route.service,
                grpc_method = %route.grpc_method,
                http_proxy_url = ?route.http_proxy_url,
                http_proxy_path_template = ?route.http_proxy_path_template,
                "✅ ROUTER: Matched static route"
            );
            let mut decision = RoutingDecision::new(&route.service, &route.grpc_method);
            if let Some(proxy) = &route.http_proxy_url {
                decision = decision.with_http_proxy(proxy);
            }
            if let Some(template) = &route.http_proxy_path_template {
                decision = decision.with_path_template(template);
            }
            decision
        })
    }

    /// Try to match against dynamic routes.
    fn try_dynamic_match(&self, path: &str, method: &str) -> Option<RoutingDecision> {
        for (pattern, route) in &self.dynamic_routes {
            if pattern.method.to_uppercase() != method.to_uppercase() {
                continue;
            }

            if let Some(params) = pattern.matches(path) {
                info!(
                    path = %path,
                    method = %method,
                    service = %route.service,
                    grpc_method = %route.grpc_method,
                    http_proxy_url = ?route.http_proxy_url,
                    http_proxy_path_template = ?route.http_proxy_path_template,
                    params = ?params,
                    "✅ ROUTER: Matched dynamic route"
                );
                let mut decision = RoutingDecision::with_params(
                    &route.service,
                    &route.grpc_method,
                    params,
                );
                if let Some(proxy) = &route.http_proxy_url {
                    decision = decision.with_http_proxy(proxy);
                }
                if let Some(template) = &route.http_proxy_path_template {
                    decision = decision.with_path_template(template);
                }
                return Some(decision);
            }
        }

        None
    }

    /// Get all routes as a vector.
    pub fn get_all_routes(&self) -> Vec<RouteConfig> {
        let mut all_routes = Vec::new();

        // Add static routes
        all_routes.extend(self.static_routes.values().cloned());

        // Add dynamic routes
        all_routes.extend(self.dynamic_routes.iter().map(|(_, route)| route.clone()));

        all_routes
    }
}
