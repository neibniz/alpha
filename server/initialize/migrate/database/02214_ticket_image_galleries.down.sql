ALTER TABLE ticket_follow
    DROP COLUMN IF EXISTS images;

ALTER TABLE ticket
    DROP COLUMN IF EXISTS images;
