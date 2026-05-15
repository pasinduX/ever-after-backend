-- Enable uuid extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Weddings
CREATE TABLE IF NOT EXISTS weddings (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    owner_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_names    TEXT NOT NULL,
    wedding_date    DATE NOT NULL,
    venue           TEXT NOT NULL DEFAULT '',
    welcome_message TEXT NOT NULL DEFAULT '',
    qr_slug         TEXT NOT NULL UNIQUE,
    qr_code_url     TEXT NOT NULL DEFAULT '',
    tier            TEXT NOT NULL DEFAULT 'elopement' CHECK (tier IN ('elopement','heritage','legacy')),
    privacy         TEXT NOT NULL DEFAULT 'public' CHECK (privacy IN ('public','private','password_protected')),
    password_hash   TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_weddings_owner_id ON weddings(owner_id);
CREATE INDEX IF NOT EXISTS idx_weddings_qr_slug  ON weddings(qr_slug);

-- Uploads
CREATE TABLE IF NOT EXISTS uploads (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    wedding_id  TEXT NOT NULL REFERENCES weddings(id) ON DELETE CASCADE,
    guest_name  TEXT,
    file_url    TEXT NOT NULL,
    file_key    TEXT NOT NULL,
    file_type   TEXT NOT NULL CHECK (file_type IN ('photo','video')),
    mime_type   TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    category    TEXT NOT NULL DEFAULT 'other' CHECK (category IN ('ceremony','candid','dancing','family','other')),
    is_approved BOOLEAN NOT NULL DEFAULT TRUE,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_uploads_wedding_id ON uploads(wedding_id);
CREATE INDEX IF NOT EXISTS idx_uploads_category   ON uploads(category);

-- Orders
CREATE TABLE IF NOT EXISTS orders (
    id                       TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    wedding_id               TEXT NOT NULL REFERENCES weddings(id) ON DELETE CASCADE,
    user_id                  TEXT NOT NULL REFERENCES users(id),
    tier                     TEXT NOT NULL CHECK (tier IN ('elopement','heritage','legacy')),
    amount_cents             BIGINT NOT NULL,
    currency                 TEXT NOT NULL DEFAULT 'usd',
    status                   TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','refunded')),
    stripe_session_id        TEXT NOT NULL UNIQUE,
    stripe_payment_intent_id TEXT,
    paid_at                  TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_wedding_id ON orders(wedding_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id    ON orders(user_id);
