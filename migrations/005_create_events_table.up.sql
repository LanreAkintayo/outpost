CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    event_type_id UUID NOT NULL REFERENCES event_types(id) ON DELETE RESTRICT,
    payload JSONB NOT NULL,
    idempotency_key VARCHAR(255),
    recipient_id VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_application_id ON events(application_id);
CREATE INDEX IF NOT EXISTS idx_events_event_type_id ON events(event_type_id);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_events_app_idempotency 
    ON events(application_id, idempotency_key) 
    WHERE idempotency_key IS NOT NULL AND idempotency_key != '';
