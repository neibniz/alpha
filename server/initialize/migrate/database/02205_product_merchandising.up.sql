ALTER TABLE subscribe
    ADD COLUMN IF NOT EXISTS display_template varchar(32) NOT NULL DEFAULT 'standard',
    ADD COLUMN IF NOT EXISTS badge varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS min_quantity integer NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS max_quantity integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sale_starts_at timestamptz NULL,
    ADD COLUMN IF NOT EXISTS sale_ends_at timestamptz NULL;

ALTER TABLE subscribe
    ADD CONSTRAINT subscribe_min_quantity_check CHECK (min_quantity >= 1),
    ADD CONSTRAINT subscribe_max_quantity_check CHECK (max_quantity >= 0),
    ADD CONSTRAINT subscribe_quantity_range_check CHECK (max_quantity = 0 OR max_quantity >= min_quantity),
    ADD CONSTRAINT subscribe_sale_window_check CHECK (
        sale_starts_at IS NULL OR sale_ends_at IS NULL OR sale_ends_at > sale_starts_at
    );
