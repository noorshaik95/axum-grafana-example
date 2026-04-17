-- In-platform messaging tables
-- Replaces the old SMTP email queue model with user-to-user messaging

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    thread_id UUID NOT NULL,
    parent_id UUID,
    from_user_id UUID NOT NULL,
    subject VARCHAR(1000),
    body TEXT NOT NULL,
    is_deleted_by_sender BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_thread ON messages(thread_id);
CREATE INDEX idx_messages_tenant ON messages(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS message_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id),
    recipient_user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    is_read BOOLEAN DEFAULT false,
    is_archived BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_recipients_user ON message_recipients(tenant_id, recipient_user_id, is_read);
CREATE INDEX idx_recipients_message ON message_recipients(message_id);

INSERT INTO schema_migrations (version) VALUES ('002_create_messages') ON CONFLICT DO NOTHING;
