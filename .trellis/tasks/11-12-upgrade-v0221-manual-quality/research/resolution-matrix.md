# v0.2.21-hotfix1 integration resolution matrix

## Pinned revisions

- Local baseline: `8bb5def3d82f2162b546899c3f6b0ce7a7f928b7`.
- Upstream release: `10ba8f9b4dc2e6023d4a9879ab77b21841babd10` (`v0.2.21`).
- Shared paths: 52, enumerated in `shared-paths.txt`.
- Text conflicts previewed: `.gitignore`, `backend/cmd/server/VERSION`, `backend/internal/repository/custom_admin_key_scope.go`.

## Conflict decisions

| Path | Required resolution |
| --- | --- |
| `.gitignore` | Preserve local Trellis/audit/closed-plugin rules and upstream new documentation exceptions. |
| `backend/cmd/server/VERSION` | Set fork version `0.2.21-hotfix1`. |
| `backend/internal/repository/custom_admin_key_scope.go` | Retain fork account-list query parameters while accepting upstream authorization/scope filtering. Inspect final call signature and scoped-key tests. |

## Shared-area review contracts

| Area | Required behavior |
| --- | --- |
| Codex forwarding/body patches | Retain opt-out identity restoration, original native IDs and item references, account identity scoping and encrypted-content recovery through new upstream memory/JSON batching. |
| WS ingress/pool/adapter | Retain final handshake metadata compatibility key and staged first/later-frame fingerprint parity; include hosted-search behavior from upstream. |
| Account repo/admin/routes | Retain plan-tier filters and quality endpoints with new admin-key/MCP authorization behavior. |
| Frontend router/sidebar/locales | Retain Quality Operations route and account filters alongside MCP and other upstream settings. |
| Build/config/generated files | Accept upstream wiring and config, preserve pnpm 11 override parity and deployment identity switch defaults. |

## Migration identities

Keep imported full filenames `249_add_usage_logs_api_key_cost_index_notx.sql`, `249_mcp_oauth.sql`, `250_openai_cache_write_ttl.sql` unchanged. They coexist with local `249_account_quality_operation_history.sql`; migration identity is the full filename, not its numeric prefix. Confirm local quality history and new MCP/cache-write schema survive repeat startup.

## Verification

Merge completed in `81231fa12` with pinned upstream as parent 2. Core Codex identity/fingerprint/account-identity/native-transform files are byte-identical to the fork baseline. HTTP/passthrough/Messages restoration, WS identity partitioning, first/later-frame staging and full-filename migrations were reviewed. Validation results and environmental limits are recorded in `validation.md`.
