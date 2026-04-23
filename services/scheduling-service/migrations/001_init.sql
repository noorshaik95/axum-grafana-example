-- W6.2 — scheduling-service initial schema
-- Tenant-scoped: schema-per-tenant provisioning means this DB belongs to ONE tenant.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS oh_schedules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  instructor_id UUID NOT NULL,
  day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  slot_duration_minutes INT NOT NULL DEFAULT 15 CHECK (slot_duration_minutes > 0),
  format VARCHAR(20) NOT NULL DEFAULT 'online',
  location TEXT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT oh_schedules_time_order CHECK (start_time < end_time)
);

CREATE INDEX IF NOT EXISTS oh_schedules_instructor_idx
  ON oh_schedules (instructor_id, is_active, day_of_week);

CREATE TABLE IF NOT EXISTS oh_bookings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  schedule_id UUID REFERENCES oh_schedules(id) ON DELETE SET NULL,
  instructor_id UUID NOT NULL,
  student_id UUID NOT NULL,
  slot_date DATE NOT NULL,
  slot_start_time TIME NOT NULL,
  questions TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS oh_bookings_instructor_day_idx
  ON oh_bookings (instructor_id, slot_date, status);

CREATE INDEX IF NOT EXISTS oh_bookings_student_idx
  ON oh_bookings (student_id, slot_date);

-- Prevent double-booking the same (instructor, slot_date, slot_start_time)
-- while a booking is still 'confirmed'.
CREATE UNIQUE INDEX IF NOT EXISTS oh_bookings_unique_confirmed_slot
  ON oh_bookings (instructor_id, slot_date, slot_start_time)
  WHERE status = 'confirmed';
