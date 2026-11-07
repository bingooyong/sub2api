# Frontend Build Toolchain

## 1. Scope / Trigger

Read this contract when changing frontend dependencies, dependency overrides,
pnpm settings, CI Node versions, or release frontend assets. It also applies to
`ERR_PNPM_IGNORED_BUILDS` and `ERR_PNPM_LOCKFILE_CONFIG_MISMATCH` diagnostics.

## 2. Signatures

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run lint:check
pnpm run typecheck
pnpm run test:run
pnpm run build
```

`frontend/vite.config.ts` writes to `backend/internal/web/dist`. The server
embeds this directory with `//go:embed all:dist`.

## 3. Contracts

- pnpm 11 ignores `package.json`'s `pnpm` field. Keep dependency overrides in
  both `frontend/package.json` (`pnpm.overrides`, for older pnpm versions) and
  `frontend/pnpm-workspace.yaml` (`overrides`, for pnpm 11). Their keys and values
  must match the lockfile's recorded override set.
- Use a frozen install for validation and release builds. A mismatch means the
  active settings and lockfile disagree; it does not mean frozen installs are
  unsupported by pnpm 11. Inspect the actual override sets before changing the
  lockfile. Updating dependencies deliberately may regenerate the lockfile, but
  review that diff for unrelated upgrades and then verify a frozen install.
- pnpm 11 uses `allowBuilds` with boolean values. Permit only the existing
  script consumers, `esbuild` and `vue-demi`. Keep `onlyBuiltDependencies` for
  older pnpm versions. Do not approve arbitrary packages or commit pnpm's
  placeholder strings (`set this to true or false`).
- Quote YAML override keys containing selectors and their values. Keep the
  dependency policy mirrored when updating a security floor.
- vue-i18n 11 requires Node >=22. The frontend CI/release workflows and Docker
  build use Node 24. Keep `vue-i18n` and `@intlify/message-compiler` on compatible
  patched versions.
- Preserve the Vite alias to `vue-i18n/dist/vue-i18n.runtime.esm-bundler.js` for
  CSP compatibility. JIT is always enabled in vue-i18n 10 and later; the obsolete
  `__INTLIFY_JIT_COMPILATION__` define is unnecessary.
- Produce frontend assets from the same source snapshot as the server binary.
  A stale `dist` embeds the wrong UI even when the backend build succeeds.

The workspace script policy is deliberately narrow:

```yaml
onlyBuiltDependencies:
  - esbuild
  - vue-demi

allowBuilds:
  esbuild: true
  vue-demi: true
```

## 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| pnpm 11 warns that `package.json`'s `pnpm` field is ignored | Verify workspace overrides match; retain the older-version mirror |
| `ERR_PNPM_LOCKFILE_CONFIG_MISMATCH` | Compare active configuration with the lockfile; reconcile the intended settings and lockfile, then rerun frozen install |
| `ERR_PNPM_IGNORED_BUILDS` for esbuild/vue-demi | Verify `allowBuilds` contains boolean `true` for those packages |
| A new package requests a build script | Inspect why it is required before extending the allowlist |
| Dependency update causes many unrelated lockfile changes | Narrow the update and review the resolved versions before proceeding |
| `dist` is missing or comes from an older source snapshot | Build the frontend before building the embedded server |

## 5. Good / Base / Bad Cases

- Good: mirrored overrides and a reviewed lockfile pass a frozen install, then
  lint, typecheck, tests, and the production build.
- Base: only esbuild and vue-demi run dependency build scripts.
- Bad: disabling frozen checks for every pnpm 11 install or treating a successful
  build as proof that security overrides applied.
- Bad: raising a dependency's major version without updating Node prerequisites,
  checking runtime compatibility, or migrating assertions that depended on old
  translation behavior.

## 6. Tests Required

- A fresh `pnpm install --frozen-lockfile` must exit zero. Do not pipe a critical
  command through `tail` and report the pipe's final command as the install result.
- Verify override keys/values match across package.json, workspace YAML, and
  lockfile. Inspect installed versions for changed security floors.
- Run frontend lint, typecheck, and the relevant/full tests after a dependency
  upgrade. For vue-i18n changes include locale compilation/completeness and
  translated component assertions.
- Run `pnpm run build` and verify `backend/internal/web/dist/index.html` and its
  referenced assets exist before building the server.
- Measured on 2026-11-07: Node 26.8.1 and pnpm 11.16.0 successfully installed the
  reviewed v0.2.19-hotfix1 lockfile with `--frozen-lockfile`, including patched
  vue-i18n/compiler 11.4.16, decode-uri-component 0.5.1, and follow-redirects 1.16.2.
  Explicit `allowBuilds` booleans allowed the two existing install scripts.

## 7. Wrong vs Correct

Wrong: leave overrides only in `package.json`, or use non-boolean build approvals.

```yaml
allowBuilds:
  esbuild: set this to true or false
```

Correct: mirror overrides in the workspace file and explicitly allow known scripts.

```yaml
overrides:
  "follow-redirects@<1.16.2": ">=1.16.2"
  "decode-uri-component@<0.5.1": ">=0.5.1"

allowBuilds:
  esbuild: true
  vue-demi: true
```

Correct: keep the install's exit status and review its log independently.

```bash
pnpm install --frozen-lockfile > /tmp/sub2api-frontend-install.log 2>&1
```
