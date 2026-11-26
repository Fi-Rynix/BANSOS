CREATE TABLE IF NOT EXISTS batches (
    id         SERIAL      PRIMARY KEY,
    module_id  INT         NOT NULL REFERENCES modules (id) ON DELETE RESTRICT,
    quota      INT         NOT NULL CHECK (quota > 0),
    allocated  INT         NOT NULL DEFAULT 0,
    opens_at   TIMESTAMPTZ NOT NULL,
    closes_at  TIMESTAMPTZ NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'draft'
               CHECK (status IN ('draft', 'scheduled', 'open', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_batches_allocated CHECK (allocated >= 0 AND allocated <= quota),
    CONSTRAINT chk_batches_window CHECK (closes_at > opens_at)
);

CREATE INDEX IF NOT EXISTS idx_batches_module
    ON batches (module_id);

CREATE INDEX IF NOT EXISTS idx_batches_status_opens
    ON batches (status, opens_at);
