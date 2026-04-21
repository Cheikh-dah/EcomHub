-- user_identities: map external auth (e.g. Clerk) to internal users
-- stores.status: public storefront only when 'active'

ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

CREATE TABLE IF NOT EXISTS user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    provider_email TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_identities_provider_subject UNIQUE (provider, provider_subject)
);

CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities (user_id);

ALTER TABLE stores ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

UPDATE stores SET status = 'active' WHERE status IS NULL OR btrim(status) = '';

ALTER TABLE stores ALTER COLUMN status SET DEFAULT 'active';
ALTER TABLE stores ALTER COLUMN status SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.stores'::regclass
          AND conname = 'stores_status_check'
    ) THEN
        ALTER TABLE stores
            ADD CONSTRAINT stores_status_check
            CHECK (status IN ('active', 'suspended', 'deleted'));
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_stores_status ON stores (status);
