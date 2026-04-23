use std::collections::{HashMap, HashSet};
use tracing::{info, warn};

use crate::config::{RouteConfig, RouteOverride};

/// Handler for applying manual route overrides
pub struct OverrideHandler;

impl OverrideHandler {
    /// Apply manual overrides to discovered routes and add new routes for overrides
    /// that don't match any discovered routes
    ///
    /// # Arguments
    /// * `discovered` - List of discovered routes
    /// * `overrides` - List of manual route overrides
    ///
    /// # Returns
    /// * List of routes with overrides applied and new routes added
    pub fn apply_overrides(
        discovered: Vec<RouteConfig>,
        overrides: &[RouteOverride],
    ) -> Vec<RouteConfig> {
        if overrides.is_empty() {
            return discovered;
        }

        info!(
            discovered_routes = discovered.len(),
            overrides = overrides.len(),
            "Applying route overrides"
        );

        // Build a map of overrides by grpc_method for quick lookup. HTTP-proxy
        // overrides (§1a) have no grpc_method, so they are always treated as
        // new routes and skipped here.
        let override_map: HashMap<String, &RouteOverride> = overrides
            .iter()
            .filter(|o| o.http_proxy_target.is_none() && !o.grpc_method.is_empty())
            .map(|o| (o.grpc_method.clone(), o))
            .collect();

        // Track which overrides have been applied
        let mut applied_overrides: HashSet<String> = HashSet::new();
        let mut result = Vec::new();
        let discovered_count = discovered.len();

        // First pass: apply overrides to discovered routes
        for route in discovered {
            if let Some(override_config) = override_map.get(&route.grpc_method) {
                // Apply override to existing route
                let overridden_route = RouteConfig {
                    path: override_config
                        .http_path
                        .clone()
                        .unwrap_or_else(|| route.path.clone()),
                    method: override_config
                        .http_method
                        .clone()
                        .unwrap_or_else(|| route.method.clone()),
                    service: route.service.clone(),
                    grpc_method: route.grpc_method.clone(),
                    http_proxy_url: None,
                    http_proxy_path_template: None,
                };

                info!(
                    grpc_method = %route.grpc_method,
                    original_path = %route.path,
                    original_method = %route.method,
                    override_path = %overridden_route.path,
                    override_method = %overridden_route.method,
                    "Applied route override to discovered route"
                );

                applied_overrides.insert(route.grpc_method.clone());
                result.push(overridden_route);
            } else {
                // No override, use discovered route as-is
                result.push(route);
            }
        }

        // Second pass: add new routes for overrides that weren't applied
        for override_config in overrides {
            // HTTP-proxy passthrough routes (§1a): no grpc_method, just
            // forward the HTTP request as-is to the configured target.
            if let Some(proxy_target) = &override_config.http_proxy_target {
                if let (Some(http_path), Some(http_method), Some(service)) = (
                    &override_config.http_path,
                    &override_config.http_method,
                    &override_config.service,
                ) {
                    let new_route = RouteConfig {
                        path: http_path.clone(),
                        method: http_method.clone(),
                        service: service.clone(),
                        grpc_method: String::new(),
                        http_proxy_url: Some(proxy_target.clone()),
                        http_proxy_path_template: override_config
                            .http_proxy_path_template
                            .clone(),
                    };

                    let path_template_display = override_config
                        .http_proxy_path_template
                        .as_deref()
                        .unwrap_or("(verbatim)");
                    info!(
                        http_path = %http_path,
                        http_method = %http_method,
                        service = %service,
                        proxy_target = %proxy_target,
                        path_template = %path_template_display,
                        "Added new HTTP-proxy route from override"
                    );

                    result.push(new_route);
                } else {
                    warn!(
                        proxy_target = %proxy_target,
                        "HTTP-proxy override skipped: missing http_path, http_method, or service"
                    );
                }
                continue;
            }

            if !applied_overrides.contains(&override_config.grpc_method) {
                // This override doesn't match any discovered route, so add it as a new route
                if let (Some(http_path), Some(http_method), Some(service)) = (
                    &override_config.http_path,
                    &override_config.http_method,
                    &override_config.service,
                ) {
                    let new_route = RouteConfig {
                        path: http_path.clone(),
                        method: http_method.clone(),
                        service: service.clone(),
                        grpc_method: override_config.grpc_method.clone(),
                        http_proxy_url: None,
                        http_proxy_path_template: None,
                    };

                    info!(
                        grpc_method = %override_config.grpc_method,
                        http_path = %http_path,
                        http_method = %http_method,
                        service = %service,
                        "Added new route from override (not discovered)"
                    );

                    result.push(new_route);
                } else {
                    warn!(
                        grpc_method = %override_config.grpc_method,
                        "Override not applied: missing required fields (http_path, http_method, or service)"
                    );
                }
            }
        }

        info!(
            total_routes = result.len(),
            discovered = discovered_count,
            added_from_overrides = result.len() - discovered_count,
            "Route override processing complete"
        );

        result
    }
}
