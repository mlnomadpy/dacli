CREATE TABLE controlplane_event_cursor (
    stream TEXT PRIMARY KEY,
    sequence BIGINT NOT NULL CHECK (sequence >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
