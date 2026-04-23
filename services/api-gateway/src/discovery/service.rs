use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;
use tokio::task::JoinHandle;
use tracing::{debug, error, info, warn};

use crate::config::{DiscoveryConfig, RouteConfig, RouteOverride, ServiceConfig};
use crate::grpc::GrpcClientPool;
use crate::router::RequestRouter;

use super::{ConventionMapper, DiscoveryError, OverrideHandler, ReflectionClient, RouteValidator};

/// Service responsible for discovering routes from backend gRPC services
pub struct RouteDiscoveryService {
    grpc_pool: Arc<GrpcClientPool>,
    config: DiscoveryConfig,
    convention_mapper: ConventionMapper,
}

impl RouteDiscoveryService {
    /// Create a new route discovery service
    pub fn new(grpc_pool: Arc<GrpcClientPool>, config: DiscoveryConfig) -> Self {
        Self {
            grpc_pool,
            config,
            convention_mapper: ConventionMapper::new(),
        }
    }

    /// Discover routes from all configured services
    pub async fn discover_routes(
        &self,
        services: &HashMap<String, ServiceConfig>,
    ) -> Result<Vec<RouteConfig>, DiscoveryError> {
        info!(
            services = services.len(),
            "Starting route discovery from all services"
        );

        let mut all_routes = Vec::new();
        let mut success_count = 0;
        let mut failure_count = 0;

        // Discover routes from each service
        for (service_name, service_config) in services {
            // Filter services based on auto_discover flag
            if !service_config.auto_discover {
                debug!(
                    service = %service_name,
                    "Skipping service (auto_discover is false)"
                );
                continue;
            }

            match self
                .discover_service_routes(service_name, service_config)
                .await
            {
                Ok(routes) => {
                    info!(
                        service = %service_name,
                        routes = routes.len(),
                        "Successfully discovered routes from service"
                    );
                    all_routes.extend(routes);
                    success_count += 1;
                }
                Err(e) => {
                    // Log error but continue with other services
                    error!(
                        service = %service_name,
                        error = %e,
                        "Failed to discover routes from service, skipping"
                    );
                    failure_count += 1;
                }
            }
        }

        info!(
            total_routes = all_routes.len(),
            services_success = success_count,
            services_failed = failure_count,
            "Route discovery completed"
        );

        Ok(all_routes)
    }

    /// Discover routes from a single service
    async fn discover_service_routes(
        &self,
        service_name: &str,
        service_config: &ServiceConfig,
    ) -> Result<Vec<RouteConfig>, DiscoveryError> {
        debug!(
            service = %service_name,
            endpoint = %service_config.endpoint,
            "Discovering routes from service"
        );

        // Get channel and create reflection client
        let channel = self
            .grpc_pool
            .get_channel(service_name)
            .map_err(|e| DiscoveryError::ConnectionFailed(format!("{}: {}", service_name, e)))?;

        let mut reflection_client = ReflectionClient::new(channel);

        // List all gRPC services
        let services = self
            .list_grpc_services(&mut reflection_client, service_name)
            .await?;

        // Discover routes from all gRPC services
        let routes = self
            .discover_routes_from_grpc_services(&mut reflection_client, service_name, &services)
            .await?;

        // Validate no duplicates
        RouteValidator::check_duplicates(&routes)?;

        Ok(routes)
    }

    /// List all gRPC services from a backend service
    async fn list_grpc_services(
        &self,
        reflection_client: &mut ReflectionClient,
        service_name: &str,
    ) -> Result<Vec<String>, DiscoveryError> {
        let services = reflection_client.list_services().await.map_err(|e| {
            // Check if this is an UNIMPLEMENTED error (reflection not supported)
            if let super::ReflectionError::GrpcError(status) = &e {
                if status.code() == tonic::Code::Unimplemented {
                    return DiscoveryError::ReflectionNotSupported(service_name.to_string());
                }
            }
            DiscoveryError::QueryFailed(format!("{}: {}", service_name, e))
        })?;

        if services.is_empty() {
            warn!(
                service = %service_name,
                "No gRPC services found via reflection"
            );
            return Err(DiscoveryError::EmptyService(service_name.to_string()));
        }

        debug!(
            service = %service_name,
            grpc_services = services.len(),
            "Found gRPC services via reflection"
        );

        Ok(services)
    }

    /// Discover routes from gRPC services using reflection
    async fn discover_routes_from_grpc_services(
        &self,
        reflection_client: &mut ReflectionClient,
        service_name: &str,
        grpc_services: &[String],
    ) -> Result<Vec<RouteConfig>, DiscoveryError> {
        let mut routes = Vec::new();
        let mut skipped_methods = 0;

        for grpc_service in grpc_services {
            // Get methods for this gRPC service
            let methods = reflection_client
                .list_methods(grpc_service)
                .await
                .map_err(|e| DiscoveryError::QueryFailed(format!("{}: {}", grpc_service, e)))?;

            debug!(
                service = %service_name,
                grpc_service = %grpc_service,
                methods = methods.len(),
                "Found methods in gRPC service"
            );

            // Map each method to an HTTP route using conventions
            for method in methods {
                tracing::info!(
                    service = %service_name,
                    grpc_service = %grpc_service,
                    method_name = %method.name,
                    full_name = %method.full_name,
                    "🔍 DISCOVERY: Processing method for route mapping"
                );

                match self.convention_mapper.map_method(
                    grpc_service,
                    &method.name,
                    &method.full_name,
                ) {
                    Some(mapping) => {
                        let route = RouteConfig {
                            path: mapping.http_path.clone(),
                            method: mapping.http_method.clone(),
                            service: service_name.to_string(),
                            grpc_method: mapping.grpc_method.clone(),
                            http_proxy_url: None,
                            http_proxy_path_template: None,
                        };

                        tracing::info!(
                            service = %service_name,
                            grpc_method = %method.name,
                            http_method = %route.method,
                            http_path = %route.path,
                            "✅ DISCOVERY: Successfully mapped gRPC method to HTTP route"
                        );

                        routes.push(route);
                    }
                    None => {
                        // Method doesn't match conventions, skip it
                        tracing::warn!(
                            service = %service_name,
                            grpc_method = %method.name,
                            full_name = %method.full_name,
                            "❌ DISCOVERY: Method does not match naming conventions, skipping"
                        );
                        skipped_methods += 1;
                    }
                }
            }
        }

        if skipped_methods > 0 {
            debug!(
                service = %service_name,
                skipped = skipped_methods,
                "Skipped methods that don't match conventions"
            );
        }

        if routes.is_empty() {
            warn!(
                service = %service_name,
                "No routes discovered (no methods matched conventions)"
            );
            return Err(DiscoveryError::EmptyService(service_name.to_string()));
        }

        Ok(routes)
    }

    /// Apply manual overrides to discovered routes
    ///
    /// This is a convenience wrapper around OverrideHandler::apply_overrides
    pub fn apply_overrides(
        &self,
        discovered: Vec<RouteConfig>,
        overrides: &[RouteOverride],
    ) -> Vec<RouteConfig> {
        OverrideHandler::apply_overrides(discovered, overrides)
    }

    /// Handle route discovery for a single service with error handling.
    async fn handle_service_discovery(
        &self,
        service_name: &str,
        service_config: &ServiceConfig,
        router: &Arc<RwLock<RequestRouter>>,
        stats: &mut RefreshStats,
    ) -> Vec<RouteConfig> {
        match self
            .discover_service_routes(service_name, service_config)
            .await
        {
            Ok(routes) => {
                info!(
                    service = %service_name,
                    routes = routes.len(),
                    "Successfully refreshed routes from service"
                );
                stats.success_count += 1;
                routes
            }
            Err(DiscoveryError::ConnectionFailed(ref e)) => {
                warn!(
                    service = %service_name,
                    error = %e,
                    "Service unreachable during refresh, retaining existing routes"
                );
                stats.failure_count += 1;
                self.retain_existing_routes(service_name, router, stats)
                    .await
            }
            Err(DiscoveryError::EmptyService(_)) => {
                info!(
                    service = %service_name,
                    "Service has no discoverable methods, removing routes"
                );
                stats.success_count += 1;
                Vec::new()
            }
            Err(e) => {
                error!(
                    service = %service_name,
                    error = %e,
                    "Failed to refresh routes from service, skipping"
                );
                stats.failure_count += 1;
                self.retain_existing_routes(service_name, router, stats)
                    .await
            }
        }
    }

    /// Retain existing routes for a service.
    async fn retain_existing_routes(
        &self,
        service_name: &str,
        router: &Arc<RwLock<RequestRouter>>,
        stats: &mut RefreshStats,
    ) -> Vec<RouteConfig> {
        let router_guard = router.read().await;
        let existing = router_guard.get_routes_for_service(service_name);
        let existing_count = existing.len();
        drop(router_guard);

        if existing_count > 0 {
            info!(
                service = %service_name,
                routes = existing_count,
                "Retained existing routes"
            );
            stats.retained_count += existing_count;
        }
        existing
    }

    /// Perform a single refresh cycle.
    async fn perform_refresh_cycle(
        &self,
        router: &Arc<RwLock<RequestRouter>>,
        services: &HashMap<String, ServiceConfig>,
        route_overrides: &[RouteOverride],
    ) -> RefreshStats {
        let cycle_start = std::time::Instant::now();
        let mut stats = RefreshStats::default();
        let mut all_routes = Vec::new();

        // Discover routes from each service
        for (service_name, service_config) in services {
            if !service_config.auto_discover {
                debug!(
                    service = %service_name,
                    "Skipping service (auto_discover is false)"
                );
                continue;
            }

            let routes = self
                .handle_service_discovery(service_name, service_config, router, &mut stats)
                .await;
            all_routes.extend(routes);
        }

        // Apply overrides and deduplicate
        let all_routes = self.apply_overrides(all_routes, route_overrides);
        info!(
            overrides_configured = route_overrides.len(),
            "Applied route overrides during periodic refresh"
        );

        let all_routes = RouteValidator::deduplicate_routes(all_routes);

        // Update router
        stats.total_routes = all_routes.len();
        let mut router_guard = router.write().await;
        router_guard.update_routes(all_routes);
        drop(router_guard);

        stats.duration = cycle_start.elapsed();
        stats
    }

    /// Start periodic refresh background task
    ///
    /// This spawns a background tokio task that periodically re-discovers routes
    /// from all configured services. The task handles partial failures gracefully:
    /// - If a service is unreachable, existing routes are retained
    /// - If a service responds, routes are updated (including deletions)
    /// - Logs detailed success/failure counts for each refresh cycle
    ///
    /// # Arguments
    /// * `self` - Arc-wrapped self for sharing across async tasks
    /// * `router` - Arc<RwLock<>> wrapped router for thread-safe updates
    /// * `services` - Map of service configurations to refresh from
    /// * `route_overrides` - Manual route overrides to apply after each discovery cycle
    ///
    /// # Returns
    /// * `JoinHandle<()>` - Handle to the background task
    pub fn start_refresh_task(
        self: Arc<Self>,
        router: Arc<RwLock<RequestRouter>>,
        services: HashMap<String, ServiceConfig>,
        route_overrides: Vec<RouteOverride>,
    ) -> JoinHandle<()> {
        let interval_secs = self.config.refresh_interval_seconds;

        info!(
            interval_seconds = interval_secs,
            "Starting periodic route refresh background task"
        );

        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(interval_secs));
            interval.tick().await; // Skip first tick

            loop {
                interval.tick().await;
                info!("Starting periodic route refresh cycle");

                let stats = self
                    .perform_refresh_cycle(&router, &services, &route_overrides)
                    .await;

                info!(
                    total_routes = stats.total_routes,
                    services_success = stats.success_count,
                    services_failed = stats.failure_count,
                    routes_retained = stats.retained_count,
                    duration_ms = stats.duration.as_millis(),
                    "Route refresh cycle completed"
                );
            }
        })
    }
}

/// Statistics for a refresh cycle.
#[derive(Default)]
struct RefreshStats {
    total_routes: usize,
    success_count: usize,
    failure_count: usize,
    retained_count: usize,
    duration: Duration,
}
