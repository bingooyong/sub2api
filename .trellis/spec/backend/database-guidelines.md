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

## Naming Conventions

<!-- Table names, column names, index names -->

(To be filled by the team)

---

## Common Mistakes

- Treating the numeric prefix as the migration's unique identity.
- Testing a backfill after deleting only its migration row while leaving the
  database in the post-migration schema state. Restore the representative old
  schema first so the test proves the real transition.
