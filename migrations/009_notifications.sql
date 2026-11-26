CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    outbox_id BIGINT NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL,
    subject VARCHAR(200) NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id)
);

CREATE INDEX IF NOT EXISTS notifications_user_id_idx ON notifications (user_id, outbox_id DESC);