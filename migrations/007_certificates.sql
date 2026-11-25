CREATE TABLE IF NOT EXISTS certificates (
    id SERIAL PRIMARY KEY,
    serial VARCHAR(50) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id INTEGER NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    claim_id INTEGER NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'requested',
    issued_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT certificates_claim_unique UNIQUE (claim_id),
    CONSTRAINT certificates_user_module_unique UNIQUE (user_id, module_id)
);

CREATE INDEX IF NOT EXISTS certificates_serial_idx ON certificates (serial);
CREATE INDEX IF NOT EXISTS certificates_user_id_idx ON certificates (user_id);