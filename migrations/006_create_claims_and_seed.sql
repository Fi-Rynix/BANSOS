CREATE TABLE IF NOT EXISTS claims (
    id         SERIAL      PRIMARY KEY,
    user_id    INT         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    batch_id   INT         NOT NULL REFERENCES batches (id) ON DELETE RESTRICT,
    status     VARCHAR(20) NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled', 'expired', 'waitlisted')),
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_claims_user_batch UNIQUE (user_id, batch_id)
);

CREATE INDEX IF NOT EXISTS idx_claims_user_history
    ON claims (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_claims_batch_status
    ON claims (batch_id, status, created_at, id);

CREATE TABLE IF NOT EXISTS claim_logs (
    id          SERIAL      PRIMARY KEY,
    claim_id    INT         NOT NULL REFERENCES claims (id) ON DELETE RESTRICT,
    actor_id    INT         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    from_status VARCHAR(20),
    to_status   VARCHAR(20) NOT NULL,
    note        TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_claim_logs_claim
    ON claim_logs (claim_id, created_at, id);

INSERT INTO modules (code, title, category, level, description) VALUES
    ('GO-101',  'Dasar Pemrograman Go',          'backend',  'beginner',     'Sintaks, tipe data, fungsi, dan error handling di Go'),
    ('GO-201',  'REST API dengan Go Fiber',      'backend',  'intermediate', 'Routing, middleware, validasi, dan kontrak respons'),
    ('SQL-101', 'Dasar PostgreSQL',              'database', 'beginner',     'Query, index, relasi, dan transaksi'),
    ('NOSQL-101', 'Pengenalan MongoDB',          'database', 'beginner',     'Dokumen, embedding, index, dan update atomik'),
    ('DEVOPS-101', 'Docker untuk Developer',     'devops',   'beginner',     'Image, container, compose, dan volume')
ON CONFLICT (code) DO NOTHING;

INSERT INTO batches (module_id, quota, allocated, opens_at, closes_at, status)
SELECT m.id, s.quota, 0, now() + s.opens_offset, now() + s.closes_offset, s.status
FROM modules m
JOIN (VALUES
    ('GO-101',     5,  INTERVAL '-1 day',  INTERVAL '6 days',  'open'),
    ('GO-201',     3,  INTERVAL '-1 hour', INTERVAL '2 days',  'open'),
    ('SQL-101',    20, INTERVAL '3 days',  INTERVAL '10 days', 'scheduled'),
    ('NOSQL-101',  10, INTERVAL '-10 days', INTERVAL '-3 days', 'closed'),
    ('DEVOPS-101', 15, INTERVAL '7 days',  INTERVAL '14 days', 'draft')
) AS s (code, quota, opens_offset, closes_offset, status) ON s.code = m.code
WHERE NOT EXISTS (SELECT 1 FROM batches b WHERE b.module_id = m.id);
