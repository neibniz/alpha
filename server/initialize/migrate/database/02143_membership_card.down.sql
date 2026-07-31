DROP TABLE IF EXISTS "membership_plan";

ALTER TABLE "user"
    DROP COLUMN IF EXISTS "member_expired_at";
