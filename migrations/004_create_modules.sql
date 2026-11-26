CREATE TABLE IF NOT EXISTS modules (
    id          SERIAL       PRIMARY KEY,
    code        VARCHAR(30)  NOT NULL UNIQUE,
    title       VARCHAR(150) NOT NULL,
    category    VARCHAR(50)  NOT NULL,
    level       VARCHAR(20)  NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_modules_catalog
    ON modules (created_at DESC, id DESC)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS idx_modules_category_level
    ON modules (category, level)
    WHERE is_active;
