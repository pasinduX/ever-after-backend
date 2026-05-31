CREATE TABLE IF NOT EXISTS guests (
    id             TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wedding_id     TEXT NOT NULL REFERENCES weddings(id) ON DELETE CASCADE,
    captain_name   TEXT NOT NULL,
    phone          TEXT,
    side           TEXT NOT NULL CHECK (side IN ('bride','groom','both')),
    members_invited INTEGER NOT NULL DEFAULT 1,
    members_coming  INTEGER NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','declined')),
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_guests_wedding_id ON guests(wedding_id);
