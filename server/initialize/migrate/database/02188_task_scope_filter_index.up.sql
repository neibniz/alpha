CREATE INDEX IF NOT EXISTS "idx_task_type_scope_status_created_at"
    ON "task" (
        "type",
        (COALESCE(NULLIF("scope"::jsonb->>'Type', ''), NULLIF("scope"::jsonb->>'type', ''))),
        "status",
        "created_at" DESC
    );
