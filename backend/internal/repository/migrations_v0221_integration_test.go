//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestMigrationsRunner_V0219UpgradeToV0221PreservesLocalQualityHistory(t *testing.T) {
	ctx := context.Background()
	upstreamMigrations := []string{
		"249_add_usage_logs_api_key_cost_index_notx.sql",
		"249_mcp_oauth.sql",
		"250_openai_cache_write_ttl.sql",
	}
	localMigrations := []string{
		"243_pelican_scheduled_tests.sql",
		"247_account_quality_ops.sql",
		"248_account_quality_judgment.sql",
		"249_account_quality_operation_history.sql",
	}
	readHistory := func(names []string) string {
		t.Helper()
		var history string
		var count int
		require.NoError(t, integrationDB.QueryRowContext(ctx, `
			SELECT COUNT(*), jsonb_object_agg(filename, to_jsonb(schema_migrations))::text
			FROM schema_migrations WHERE filename = ANY($1)
		`, pq.Array(names)).Scan(&count, &history))
		require.Equal(t, len(names), count)
		return history
	}
	localHistoryBefore := readHistory(localMigrations)
	var operationID int64
	t.Cleanup(func() {
		// Repair the shared disposable database even when a schema assertion fails.
		_, err := integrationDB.ExecContext(context.Background(),
			"DELETE FROM schema_migrations WHERE filename = ANY($1)", pq.Array(upstreamMigrations))
		require.NoError(t, err)
		require.NoError(t, ApplyMigrations(context.Background(), integrationDB))
		_, err = integrationDB.ExecContext(context.Background(),
			"DELETE FROM account_quality_operations WHERE id = $1", operationID)
		require.NoError(t, err)
	})

	// Restore the actual v0.2.19 schema, including the old usage/pricing shape.
	// Removing registry rows alone would only exercise IF NOT EXISTS branches.
	_, err := integrationDB.ExecContext(ctx, `
		DROP TABLE mcp_oauth_credentials, mcp_oauth_codes, mcp_oauth_grants;
		DROP INDEX idx_usage_logs_api_key_created_cost;
		ALTER TABLE usage_logs DROP COLUMN cache_creation_30m_tokens, DROP COLUMN cache_creation_12h_tokens;
		ALTER TABLE channel_model_pricing DROP COLUMN cache_write_12h_price;
		ALTER TABLE channel_pricing_intervals DROP COLUMN cache_write_12h_price;
		ALTER TABLE channel_account_stats_model_pricing DROP COLUMN cache_write_12h_price;
		ALTER TABLE channel_account_stats_pricing_intervals DROP COLUMN cache_write_12h_price;
	`)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx,
		"DELETE FROM schema_migrations WHERE filename = ANY($1)", pq.Array(upstreamMigrations))
	require.NoError(t, err)
	var missingObjects bool
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT to_regclass('public.mcp_oauth_grants') IS NULL
			AND to_regclass('public.idx_usage_logs_api_key_created_cost') IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'usage_logs'
					AND column_name IN ('cache_creation_30m_tokens', 'cache_creation_12h_tokens')
			)
	`).Scan(&missingObjects))
	require.True(t, missingObjects, "upgrade must begin from the old schema")
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO account_quality_operations (plan_id, account_id, quality_round_id, outcome, status, snapshot)
		VALUES (-1, -1, 'v0219-upgrade-' || gen_random_uuid()::text, 'isolated', 'success', '{"schedulable":true,"group_ids":[1,2]}')
		RETURNING id
	`).Scan(&operationID))
	var operationBefore string
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT to_jsonb(account_quality_operations)::text FROM account_quality_operations WHERE id = $1", operationID,
	).Scan(&operationBefore))

	require.NoError(t, ApplyMigrations(ctx, integrationDB))
	for _, name := range upstreamMigrations {
		content, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
		var checksum string
		require.NoError(t, integrationDB.QueryRowContext(ctx,
			"SELECT checksum FROM schema_migrations WHERE filename = $1", name).Scan(&checksum))
		require.Equal(t, hex.EncodeToString(sum[:]), checksum, "imported migration %s must keep its full identity", name)
	}
	require.Equal(t, localHistoryBefore, readHistory(localMigrations), "local migration checksums and timestamps must remain unchanged")
	func() {
		tx := testTx(t)
		defer tx.Rollback()
		for _, table := range []string{"mcp_oauth_grants", "mcp_oauth_codes", "mcp_oauth_credentials"} {
			requireColumn(t, tx, table, "expires_at", "timestamp with time zone", 0, false)
		}
		requireColumn(t, tx, "mcp_oauth_grants", "read_only", "boolean", 0, false)
		requireColumnDefaultContains(t, tx, "mcp_oauth_grants", "read_only", "true")
		requireForeignKeyOnDelete(t, tx, "mcp_oauth_grants", "user_id", "users", "CASCADE")
		requireForeignKeyOnDelete(t, tx, "mcp_oauth_codes", "grant_id", "mcp_oauth_grants", "CASCADE")
		requireForeignKeyOnDelete(t, tx, "mcp_oauth_credentials", "grant_id", "mcp_oauth_grants", "CASCADE")
		for _, column := range []string{"cache_creation_30m_tokens", "cache_creation_12h_tokens"} {
			requireColumn(t, tx, "usage_logs", column, "integer", 0, false)
			requireColumnDefaultContains(t, tx, "usage_logs", column, "0")
		}
		for _, table := range []string{"channel_model_pricing", "channel_pricing_intervals", "channel_account_stats_model_pricing", "channel_account_stats_pricing_intervals"} {
			requireColumn(t, tx, table, "cache_write_12h_price", "numeric", 0, true)
		}
		requireIndex(t, tx, "usage_logs", "idx_usage_logs_api_key_created_cost")
		var indexValid bool
		var indexDefinition string
		require.NoError(t, tx.QueryRowContext(ctx, `
			SELECT indisvalid, pg_get_indexdef(indexrelid)
			FROM pg_index WHERE indexrelid = 'idx_usage_logs_api_key_created_cost'::regclass
		`).Scan(&indexValid, &indexDefinition))
		require.True(t, indexValid)
		require.Contains(t, indexDefinition, "(api_key_id, created_at) INCLUDE (actual_cost)")
		requireColumn(t, tx, "account_quality_operations", "snapshot", "jsonb", 0, true)
		requireIndex(t, tx, "account_quality_operations", "idx_account_quality_operations_account")
	}()

	var historyBefore, historyAfter, operationAfter string
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT jsonb_object_agg(filename, to_jsonb(schema_migrations))::text FROM schema_migrations",
	).Scan(&historyBefore))
	require.NoError(t, ApplyMigrations(ctx, integrationDB))
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT jsonb_object_agg(filename, to_jsonb(schema_migrations))::text FROM schema_migrations",
	).Scan(&historyAfter))
	require.Equal(t, historyBefore, historyAfter, "repeat startup must not reapply migrations or alter history")
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT to_jsonb(account_quality_operations)::text FROM account_quality_operations WHERE id = $1", operationID,
	).Scan(&operationAfter))
	require.Equal(t, operationBefore, operationAfter, "upgrade and repeat startup must preserve the quality audit row")
}
