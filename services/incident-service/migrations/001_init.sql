-- W4.1 — incident-service initial schema
-- Ratified: plan/CONTRACTS.md#incident.IncidentService

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS incidents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    priority    VARCHAR(5)   NOT NULL CHECK (priority IN ('P0','P1','P2','P3','P4')),
    status      VARCHAR(20)  NOT NULL DEFAULT 'open' CHECK (status IN ('open','watching','resolved')),
    tenant_id   UUID,
    service     VARCHAR(128),
    created_by  UUID,
    resolved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incidents_status        ON incidents (status);
CREATE INDEX IF NOT EXISTS idx_incidents_tenant_status ON incidents (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_incidents_service_status ON incidents (service, status);
CREATE INDEX IF NOT EXISTS idx_incidents_priority      ON incidents (priority);
CREATE INDEX IF NOT EXISTS idx_incidents_created_at    ON incidents (created_at DESC);

CREATE TABLE IF NOT EXISTS incident_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    actor_id    UUID,
    event_type  VARCHAR(50) NOT NULL,
    content     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incident_events_incident_id ON incident_events (incident_id, created_at);
