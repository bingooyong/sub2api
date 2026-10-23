# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

<!--
Document your project's database conventions here.

Questions to answer:
- What ORM/query library do you use?
- How are migrations managed?
- What are the naming conventions for tables/columns?
- How do you handle transactions?
-->

(To be filled by the team)

---

## Query Patterns

<!-- How should queries be written? Batch operations? -->

(To be filled by the team)

---

## Migrations

### Scenario: Integrating migrations with duplicate numeric prefixes

#### 1. Scope / Trigger

Use this contract when an upstream integration introduces a migration whose
numeric prefix is lower than, or equal to, migrations already applied by a
local fork.

#### 2. Signatures

- Migration files: `backend/migrations/*.sql`
- Migration registry: `schema_migrations.filename`
- Runner: `backend/internal/repository/migrations_runner.go`

#### 3. Contracts

- A migration's identity is its complete filename, not its numeric prefix.
- The runner orders pending migrations by complete filename and records each
  complete filename in `schema_migrations`.
- Published migration files are immutable. Do not rename, renumber, or edit
  them to make prefixes unique.
- Multiple files such as `241_first.sql` and `241_second.sql` are valid and
  must both be retained.

#### 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| Filename is absent from `schema_migrations` | Run and record the migration |
| Filename is already recorded with the expected checksum | Skip it safely |
| Published migration content no longer matches its recorded checksum | Fail unless the runner has an explicit, migration-specific checksum compatibility rule; do not silently rewrite history |
| A lower-numbered upstream migration arrives after higher local migrations | Run it if its complete filename is absent |

#### 5. Good/Base/Bad Cases

- Good: an existing database has local `243_*` entries; newly imported
  `241_alpha.sql` and `241_beta.sql` both execute once.
- Base: an empty database executes every file in complete-filename order.
- Bad: renaming imported migrations to unused numbers, which changes their
  identity and makes future upstream integrations ambiguous.

#### 6. Tests Required

- Run migrations against an empty database.
- Build a representative pre-upgrade database by restoring the old schema
  constraints and marking only the migrations that existed in that release.
- Assert every newly imported filename is recorded and its schema change is
  present.
- Run the migration runner a second time and assert idempotency.
- For constraint migrations, assert the constraint definition before and
  after the upgrade; checking only the migration row is insufficient.

#### 7. Wrong vs Correct

Wrong:

```text
Rename 241_add_feature.sql to 250_add_feature.sql because 241 is already used.
```

Correct:

```text
Keep 241_add_feature.sql unchanged and verify schema_migrations by its full filename.
```

---

## Scenario: Additive migrations plus dropping a database-level platform whitelist

### 1. Scope / Trigger

Read this contract when an upstream integration adds tables, columns, and
indexes in several consecutive files while also removing a CHECK constraint
that duplicated an application-level catalog.

This is the shape of the `v0.2.17` import: five new migration files, four of
which are purely additive and one of which drops two platform CHECK
constraints.

### 2. Signatures

- Migration files: `backend/migrations/*.sql`
- Platform catalog: `backend/internal/domain/platforms.go`
  - `domain.IsConcretePlatform(id string) bool`
  - `domain.ConcretePlatformIDs() []string`
  - `domain.PlatformIDsWhere(match func(PlatformSpec) bool) []string`
- Quota platform allowlist: `backend/internal/service/domain_constants.go`
  - `service.AllowedQuotaPlatforms`
- Composite route target validation: `backend/internal/service/admin_group.go`
  - `compositeRouteFromInput(groupID int64, input CompositeRouteInput)`
- Guard test: `backend/migrations/platform_check_constraints_drop_migration_test.go`

### 3. Contracts

- The `groups.i18n` column stores per-locale display text only. Group
  uniqueness, search, and logs continue to use `name`. See
  `backend/internal/domain/group_i18n.go`: `GroupI18n.Localize` returns the
  original `name`/`description` for any locale or field that has no
  translation, and `GroupI18n.CanonicalLocale` normalizes BCP 47 casing
  without validating the locale.
- `gemini_cached_contents` is created by `243`, extended by `244` with
  `channel_usage JSONB`, and extended by `245` with the mutation lease
  columns `mutation_token` and `mutation_lease_until`. The lease is a write
  intent: `expire_time` keeps its pre-modification value until the commit,
  and a lease holder is identified by `mutation_token`. See
  `backend/internal/repository/gemini_cached_content_repo.go`.
- After `242_drop_platform_check_constraints.sql`, the platform whitelist
  lives only in the application catalog. `user_platform_quotas.platform` is
  validated by `service.AllowedQuotaPlatforms` before repository writes, and
  `composite_model_routes.target_platform` is validated by
  `compositeRouteFromInput` and by the ent schema. New platforms require no
  database migration.
- The CHECK constraints on `channel_monitors.provider` and
  `channel_monitor_request_templates.provider` express the probe capabilities
  that channel monitoring has actually implemented. They are a different
  concept from the platform catalog and are deliberately not dropped.
- No later migration may re-add `user_platform_quotas_platform_check` or
  `composite_model_routes_target_platform_check`. The guard test scans every
  migration file ordered after `242_drop_platform_check_constraints.sql` and
  fails if one contains such an `ADD CONSTRAINT`.

### 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| A new platform is registered in `domain.platformList` | It becomes usable for quota and composite routes with no migration |
| A write targets an unregistered platform | Rejected by application validation before it reaches the repository |
| A migration re-adds a dropped platform CHECK constraint | The guard test fails; the migration must be rewritten |
| A migration drops a channel-monitor provider CHECK constraint | Rejected: that constraint encodes implemented probe capability, not the platform catalog |
| A locale is missing from `groups.i18n`, or a field within a locale is empty | The original `groups.name` / `groups.description` is used |
| Two mutation requests target the same cache concurrently | The lease admits one holder; the other is rejected until `mutation_lease_until` expires |
| A lease expires without being cleared | The recovery path rolls the upstream expiry back to `expire_time` |

### 5. Good/Base/Bad Cases

- Good: `243` creates the table, `244` adds `channel_usage`, `245` adds the
  lease columns. Each file is independently re-runnable (`IF NOT EXISTS`), and
  the final state is the union.
- Base: a database that has applied `243` but not `245` still starts; the
  runner applies the remaining files by complete filename.
- Bad: adding the platform CHECK constraint back "to be safe". It creates a
  second source of truth that drifts from the catalog and blocks every
  platform added after the constraint was written.

### 6. Tests Required

- `go test ./migrations` must pass, including
  `TestDropPlatformCheckConstraintsMigration` and
  `TestLaterMigrationsDoNotRestorePlatformCheckConstraints`.
- The migration runner must be idempotent: run it twice against the same
  database and assert the `schema_migrations` count does not change.
- For a constraint migration, assert the constraint definition before and
  after the upgrade. Checking only the migration row is insufficient.
- For `groups.i18n`, assert the column is present, defaults to `'{}'::jsonb`,
  and that a missing locale falls back to the original name.
- For the mutation lease, assert the holder commits the new `expire_time` and
  clears both lease columns, and that a competing writer is rejected while
  the lease is live.

### 7. Wrong vs Correct

Wrong:

```sql
-- Re-adding the whitelist after it was moved to the application layer
ALTER TABLE user_platform_quotas
    ADD CONSTRAINT user_platform_quotas_platform_check CHECK (platform IN (...));
```

Correct:

```sql
-- The catalog is the single source of truth; the database stays neutral
ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;
```

---

## Naming Conventions

<!-- Table names, column names, index names -->

(To be filled by the team)

---

## Common Mistakes

- Treating the numeric prefix as the migration's unique identity.
- Testing a backfill after deleting only its migration row while leaving the
  database in the post-migration schema state. Restore the representative old
  schema first so the test proves the real transition.
