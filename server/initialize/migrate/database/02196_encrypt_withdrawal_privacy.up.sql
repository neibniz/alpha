DO $$
BEGIN
    IF to_regclass('public.user_withdrawal') IS NULL
       AND to_regclass('public.withdrawals') IS NOT NULL THEN
        ALTER TABLE "withdrawals" RENAME TO "user_withdrawal";
    END IF;
END $$;

ALTER TABLE "user_withdrawal"
    ALTER COLUMN "content" TYPE text,
    ALTER COLUMN "reason" TYPE text;
