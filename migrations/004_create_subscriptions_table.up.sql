CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    event_type_id UUID NOT NULL REFERENCES event_types(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_subscriptions_endpoint_event UNIQUE (endpoint_id, event_type_id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_endpoint_id ON subscriptions(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_event_type_id ON subscriptions(event_type_id);
