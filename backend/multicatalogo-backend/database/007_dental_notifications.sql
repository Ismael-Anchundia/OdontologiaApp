CREATE TABLE IF NOT EXISTS dental_notification_outbox (
    id BIGSERIAL PRIMARY KEY,
    appointment_id BIGINT NOT NULL REFERENCES dental_appointments(id) ON DELETE CASCADE,
    patient_id INTEGER NOT NULL REFERENCES dental_patients(id),
    channel VARCHAR(16) NOT NULL CHECK (channel IN ('email','whatsapp','sms')),
    recipient VARCHAR(255) NOT NULL,
    template_key VARCHAR(32) NOT NULL CHECK (template_key IN ('appointment_confirmation','appointment_reminder_24h','appointment_reminder_48h')),
    message TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sending','sent','retry','cancelled','needs_configuration')),
    attempts SMALLINT NOT NULL DEFAULT 0,
    locked_at TIMESTAMPTZ,
    provider_reference VARCHAR(255),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    UNIQUE (appointment_id, template_key, channel)
);
CREATE INDEX IF NOT EXISTS idx_dental_notification_due ON dental_notification_outbox(status, scheduled_at)
WHERE status IN ('pending','retry');
