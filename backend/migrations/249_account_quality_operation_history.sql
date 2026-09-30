-- Quality operation audit survives result retention and plan deletion.
CREATE TABLE IF NOT EXISTS account_quality_operations (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    account_name TEXT NOT NULL DEFAULT '',
    quality_round_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    quality_action TEXT NOT NULL DEFAULT '',
    quality_judgment JSONB,
    pelican_config JSONB,
    snapshot JSONB,
    passed_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    result_ids BIGINT[] NOT NULL DEFAULT '{}',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (plan_id, quality_round_id)
);

CREATE INDEX IF NOT EXISTS idx_account_quality_operations_created
    ON account_quality_operations(id DESC);
CREATE INDEX IF NOT EXISTS idx_account_quality_operations_account
    ON account_quality_operations(account_id, id DESC);
