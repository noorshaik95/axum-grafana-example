// Include generated proto code
#[allow(clippy::module_inception)]
pub mod auth {
    tonic::include_proto!("auth");
}

pub mod gateway {
    tonic::include_proto!("gateway");
}

// Declare submodules
pub mod cache;
mod constants;
pub mod flag_middleware;
mod service;
pub mod tenant_resolver;
pub mod types;

// Export middleware module
pub mod middleware;

// Re-export the tenant resolver so app/routes can wire it into middleware state.
pub use tenant_resolver::TenantResolver;

// Re-export the unavailable-provider default so routes.rs doesn't
// have to reach into the submodule explicitly.
pub use flag_middleware::UnavailableProvider;

// Tests module
#[cfg(test)]
mod tests;

// Re-export public types
pub use types::{AuthError, AuthResult};

// Re-export public service
pub use service::AuthService;
