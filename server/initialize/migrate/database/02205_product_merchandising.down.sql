ALTER TABLE subscribe
    DROP CONSTRAINT IF EXISTS subscribe_sale_window_check,
    DROP CONSTRAINT IF EXISTS subscribe_quantity_range_check,
    DROP CONSTRAINT IF EXISTS subscribe_max_quantity_check,
    DROP CONSTRAINT IF EXISTS subscribe_min_quantity_check;

ALTER TABLE subscribe
    DROP COLUMN IF EXISTS sale_ends_at,
    DROP COLUMN IF EXISTS sale_starts_at,
    DROP COLUMN IF EXISTS max_quantity,
    DROP COLUMN IF EXISTS min_quantity,
    DROP COLUMN IF EXISTS badge,
    DROP COLUMN IF EXISTS display_template;
