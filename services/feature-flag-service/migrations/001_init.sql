CREATE TABLE IF NOT EXISTS flags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key VARCHAR(100) UNIQUE NOT NULL,
  description TEXT,
  enabled BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS flag_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  flag_id UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
  rule_type VARCHAR(50) NOT NULL,
  rule_value JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_flag_rules_flag_id ON flag_rules(flag_id);

-- Seed the five platform flags per plan W3.2.
INSERT INTO flags (key, description, enabled)
VALUES
  ('new_grading_queue', 'Route grading work through the new priority queue', false),
  ('ai_draft_feedback', 'Enable AI-generated draft feedback in grading UI', false),
  ('study_plan_v2', 'Expose v2 personalized study plan to students', false),
  ('live_class_pulse', 'Enable live-class engagement pulse checks', false),
  ('mobile_push', 'Send mobile push notifications for course events', false)
ON CONFLICT (key) DO NOTHING;
