-- Storefront UI customization foundation (theme-ready configuration).

ALTER TABLE stores
    ADD COLUMN IF NOT EXISTS theme_config JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE stores
SET theme_config = '{}'::jsonb
WHERE theme_config IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.stores'::regclass
          AND conname = 'stores_theme_config_object_check'
    ) THEN
        ALTER TABLE stores
            ADD CONSTRAINT stores_theme_config_object_check
            CHECK (jsonb_typeof(theme_config) = 'object');
    END IF;
END
$$;
