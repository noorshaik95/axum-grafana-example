-- Email Service Database Schema
-- Epic 9: Email notifications and welcome emails

-- Email templates table
CREATE TABLE IF NOT EXISTS email_templates (
    id VARCHAR(36) PRIMARY KEY,
    template_name VARCHAR(100) UNIQUE NOT NULL,
    template_type VARCHAR(50) NOT NULL, -- welcome_tenant_admin, password_reset, etc.
    subject VARCHAR(500) NOT NULL,
    html_body TEXT NOT NULL,
    text_body TEXT NOT NULL,
    variables JSONB, -- Template variables schema
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_templates_type ON email_templates(template_type);
CREATE INDEX idx_email_templates_name ON email_templates(template_name);

-- Email queue table (for async sending)
CREATE TABLE IF NOT EXISTS email_queue (
    id VARCHAR(36) PRIMARY KEY,
    to_email VARCHAR(255) NOT NULL,
    to_name VARCHAR(255),
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255),
    subject VARCHAR(500) NOT NULL,
    html_body TEXT NOT NULL,
    text_body TEXT,
    template_id VARCHAR(36) REFERENCES email_templates(id),
    template_data JSONB,

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, sending, sent, delivered, failed, bounced
    priority INTEGER NOT NULL DEFAULT 5, -- 1-10, lower is higher priority
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,

    -- Timestamps
    scheduled_at TIMESTAMP,
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,

    -- Error tracking
    error_message TEXT,
    error_code VARCHAR(50),

    -- Metadata
    tenant_id VARCHAR(36),
    user_id VARCHAR(36),
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_retry CHECK (retry_count <= max_retries)
);

CREATE INDEX idx_email_queue_status ON email_queue(status);
CREATE INDEX idx_email_queue_scheduled ON email_queue(scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_email_queue_priority ON email_queue(priority, created_at) WHERE status = 'pending';
CREATE INDEX idx_email_queue_to_email ON email_queue(to_email);
CREATE INDEX idx_email_queue_tenant ON email_queue(tenant_id);
CREATE INDEX idx_email_queue_created ON email_queue(created_at DESC);

-- Email events table (tracking delivery, opens, clicks)
CREATE TABLE IF NOT EXISTS email_events (
    id VARCHAR(36) PRIMARY KEY,
    email_id VARCHAR(36) NOT NULL REFERENCES email_queue(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- sent, delivered, opened, clicked, bounced, complained, failed
    event_data JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_events_email ON email_events(email_id);
CREATE INDEX idx_email_events_type ON email_events(event_type);
CREATE INDEX idx_email_events_timestamp ON email_events(timestamp DESC);

-- Email statistics table (aggregated metrics)
CREATE TABLE IF NOT EXISTS email_statistics (
    id VARCHAR(36) PRIMARY KEY,
    date DATE NOT NULL,
    template_type VARCHAR(50),
    tenant_id VARCHAR(36),

    total_sent INTEGER NOT NULL DEFAULT 0,
    total_delivered INTEGER NOT NULL DEFAULT 0,
    total_opened INTEGER NOT NULL DEFAULT 0,
    total_clicked INTEGER NOT NULL DEFAULT 0,
    total_bounced INTEGER NOT NULL DEFAULT 0,
    total_complained INTEGER NOT NULL DEFAULT 0,
    total_failed INTEGER NOT NULL DEFAULT 0,

    delivery_rate DECIMAL(5,2),
    open_rate DECIMAL(5,2),
    click_rate DECIMAL(5,2),
    bounce_rate DECIMAL(5,2),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(date, template_type, tenant_id)
);

CREATE INDEX idx_email_stats_date ON email_statistics(date DESC);
CREATE INDEX idx_email_stats_template ON email_statistics(template_type);
CREATE INDEX idx_email_stats_tenant ON email_statistics(tenant_id);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default email templates
INSERT INTO email_templates (id, template_name, template_type, subject, html_body, text_body, variables, language)
VALUES
    (
        'tmpl-welcome-001',
        'tenant_admin_welcome',
        'welcome_tenant_admin',
        'Welcome to {{tenant_name}} - Setup Your Account',
        '<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4F46E5; color: white; padding: 20px; text-align: center; }
        .content { background: #f9f9f9; padding: 30px; }
        .button { display: inline-block; padding: 12px 30px; background: #4F46E5; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to {{tenant_name}}!</h1>
        </div>
        <div class="content">
            <p>Hi {{admin_name}},</p>
            <p>Congratulations! Your <strong>{{tier}}</strong> tier account has been successfully created.</p>
            <p>You can now access your dashboard and start setting up your platform:</p>
            <p style="text-align: center;">
                <a href="{{setup_url}}" class="button">Setup Your Account</a>
            </p>
            <p><strong>Your Details:</strong></p>
            <ul>
                <li>Subdomain: <a href="https://{{subdomain}}">{{subdomain}}</a></li>
                <li>Email: {{admin_email}}</li>
                <li>Plan: {{tier}}</li>
            </ul>
            <p>If you have any questions, please don''t hesitate to contact our support team.</p>
            <p>Best regards,<br>The Platform Team</p>
        </div>
        <div class="footer">
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>',
        'Welcome to {{tenant_name}}!

Hi {{admin_name}},

Congratulations! Your {{tier}} tier account has been successfully created.

You can now access your dashboard and start setting up your platform:
{{setup_url}}

Your Details:
- Subdomain: {{subdomain}}
- Email: {{admin_email}}
- Plan: {{tier}}

If you have any questions, please don''t hesitate to contact our support team.

Best regards,
The Platform Team

---
This is an automated message, please do not reply to this email.',
        '{"tenant_name": "string", "admin_name": "string", "admin_email": "string", "setup_url": "string", "subdomain": "string", "tier": "string"}'::jsonb,
        'en'
    )
ON CONFLICT (id) DO NOTHING;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('001_init') ON CONFLICT DO NOTHING;
