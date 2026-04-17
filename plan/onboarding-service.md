# Onboarding Service Plan — Restate Durable Workflow

## Owner Agent: `onboarding-expert`

## Stack: Rust + restate-sdk-rust, Kafka (rdkafka), PostgreSQL (sqlx)

---

## Objective

Replace the current naive Go REST CRUD onboarding service with a production-grade durable workflow orchestration layer using Restate.dev. The workflow must handle multi-step university onboarding with saga compensation, human-in-the-loop approval, and durable timers.

---

## Current State

- `services/onboarding-service/` — Go service with REST CRUD + Kafka worker
- Dockerfiles exist, service starts, but workflow logic is not durable

## Target State

- Full Restate Virtual Object in Rust
- Saga with compensation actions across 5 downstream services
- 72-hour durable approval timer
- Restate server as sidecar container

---

## Deliverables

### 1. Rewrite `services/onboarding-service/` in Rust

**Cargo.toml dependencies:**

```toml
restate-sdk = "0.4"
rdkafka = { version = "0.37", features = ["tokio"] }
sqlx = { version = "0.8", features = ["postgres", "runtime-tokio-native-tls", "uuid", "chrono"] }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
tokio = { version = "1", features = ["full"] }
reqwest = { version = "0.12", features = ["json"] }
anyhow = "1.0"
uuid = { version = "1", features = ["v4"] }
chrono = { version = "0.4", features = ["serde"] }
```

**Struct: `OnboardingState`**

```rust
#[derive(Serialize, Deserialize, Default)]
struct OnboardingState {
    id: String,
    status: OnboardingStatus,  // Draft | InReview | PendingApproval | Provisioning | Active | Rejected | Failed
    steps: HashMap<u8, serde_json::Value>,  // step 1-4 data
    institution_name: String,
    admin_email: String,
    submitted_at: Option<DateTime<Utc>>,
    approved_by: Option<String>,
    tenant_id: Option<String>,
    error: Option<String>,
}
```

**Virtual Object: `OnboardingWorkflow`**

Handlers (all idempotent):

```rust
// POST /onboarding/{id}/start
async fn start(&self, ctx: ObjectContext, payload: StartPayload) -> Result<()>

// PUT /onboarding/{id}/step/{n}
async fn save_step(&self, ctx: ObjectContext, req: SaveStepRequest) -> Result<()>

// POST /onboarding/{id}/submit
async fn submit_for_review(&self, ctx: ObjectContext) -> Result<()>

// POST /onboarding/{id}/approve
async fn approve(&self, ctx: ObjectContext, admin_id: String) -> Result<ApprovalResult>

// POST /onboarding/{id}/reject
async fn reject(&self, ctx: ObjectContext, reason: String) -> Result<()>

// PATCH /onboarding/{id}/reopen
async fn reopen(&self, ctx: ObjectContext) -> Result<()>

// GET /onboarding/{id}/status
async fn get_status(&self, ctx: ObjectContext) -> Result<OnboardingState>
```

**Provisioning Saga (inside `approve`):**

```rust
async fn run_provisioning_saga(ctx: &ObjectContext, state: &OnboardingState) -> Result<()> {
    // Each step wrapped in ctx.run() for exactly-once semantics

    // Step 1: Provision tenant
    let tenant_id = ctx.run("provision-tenant", || provision_tenant(state)).await?;

    // Step 2: Create Stripe customer
    let stripe_id = ctx.run("create-stripe-customer", || create_stripe_customer(state)).await
        .map_err(|e| { compensate_tenant(&tenant_id); e })?;

    // Step 3: Init storage bucket
    ctx.run("init-storage", || init_storage_bucket(&tenant_id)).await
        .map_err(|e| { compensate_stripe(&stripe_id); compensate_tenant(&tenant_id); e })?;

    // Step 4: Send welcome email
    ctx.run("send-welcome", || send_welcome_email(state)).await.ok(); // non-fatal

    // Step 5: Init metrics baseline
    ctx.run("init-metrics", || init_metrics_baseline(&tenant_id)).await.ok();

    // Step 6: Emit Kafka event
    ctx.run("emit-approved", || emit_kafka_event("onboarding.approved", &tenant_id)).await?;

    Ok(())
}
```

**Durable Timer (72h escalation):**

```rust
async fn submit_for_review(&self, ctx: ObjectContext) -> Result<()> {
    // ... update state to PendingReview ...

    // Spawn escalation timer (survives restarts)
    ctx.sleep(Duration::from_secs(72 * 3600)).await;
    let state = ctx.get::<OnboardingState>("state").await?.unwrap_or_default();
    if state.status == OnboardingStatus::PendingReview {
        emit_kafka_event("onboarding.escalated", &state.id).await.ok();
    }
    Ok(())
}
```

### 2. Restate Server in Docker Compose

```yaml
restate-server:
  image: docker.restate.dev/restatedev/restate:latest
  ports:
    - '9080:9080' # HTTP ingress
    - '9090:9090' # Admin API
    - '5005:5005' # Service discovery
  environment:
    RESTATE_LOG_LEVEL: info
  networks:
    - slate-network

onboarding-service:
  build: ...
  environment:
    RESTATE_ENDPOINT: 'http://restate-server:9090'
    KAFKA_BROKERS: 'kafka:9092'
    DATABASE_URL: 'postgresql://...'
  depends_on:
    - restate-server
  networks:
    - slate-network
```

### 3. Database Schema

Keep the existing migrations for audit log. Restate handles durable state internally.
Add table: `onboarding_audit_log(id, onboarding_id, event, data, created_at)`

### 4. API Gateway Config Update

Add routes to `config/gateway-config.docker.yaml`:

- All `/onboarding/*` routes → `http://restate-server:9080`

### 5. Kafka Events Produced

- `onboarding.started`
- `onboarding.step.completed` (with step number + data summary)
- `onboarding.pending_review`
- `onboarding.approved` (with tenantId)
- `onboarding.failed` (with error + compensations applied)
- `onboarding.rejected`
- `onboarding.escalated` (72h timeout)

### 6. Tests

- Unit tests for each workflow handler (mock Restate context)
- Integration test: full happy path (start → steps → submit → approve → provisioned)
- Compensation test: inject failure at step 3 → verify steps 1+2 compensated
- Timer test: mock 72h → verify escalation event emitted

---

## Migration from Go onboarding-service

1. Stop Go onboarding-service
2. Run Rust service registration against Restate server
3. Existing `onboarding_jobs` table retained for historical data
4. New onboarding flows use Restate state

---

## Files to Create/Modify

- [REWRITE] `services/onboarding-service/src/` — full Rust rewrite
- [NEW] `services/onboarding-service/Cargo.toml`
- [MODIFY] `services/onboarding-service/Dockerfile` — Rust multi-stage build
- [MODIFY] `docker-compose.yml` — add restate-server, update onboarding-service
- [MODIFY] `config/gateway-config.docker.yaml` — onboarding routes
