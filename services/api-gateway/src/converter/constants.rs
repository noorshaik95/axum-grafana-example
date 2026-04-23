/// List of headers to propagate to backend services
pub const PROPAGATE_HEADERS: &[&str] = &[
    "x-trace-id",
    "x-span-id",
    "x-parent-span-id",
    "x-request-id",
    "x-correlation-id",
    "traceparent",
    "tracestate",
    "user-agent",
    "x-forwarded-for",
    "x-real-ip",
];

/// Metadata keys that should be propagated back to the client
pub const CLIENT_PROPAGATE_HEADERS: &[&str] = &[
    "x-trace-id",
    "x-span-id",
    "x-request-id",
    "x-correlation-id",
    "traceparent",
    "tracestate",
];
