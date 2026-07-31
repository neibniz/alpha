WITH ranked_balance AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            ORDER BY CASE WHEN id = -1 THEN 0 ELSE 1 END, id
        ) AS row_number
    FROM payment
    WHERE LOWER(platform) = 'balance' AND enable = TRUE
)
UPDATE payment
SET enable = FALSE
FROM ranked_balance
WHERE payment.id = ranked_balance.id
  AND ranked_balance.row_number > 1;

CREATE UNIQUE INDEX IF NOT EXISTS ux_payment_single_enabled_balance
    ON payment ((LOWER(platform)))
    WHERE LOWER(platform) = 'balance' AND enable = TRUE;
