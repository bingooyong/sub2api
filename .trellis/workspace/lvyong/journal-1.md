# Journal - lvyong (Part 1)

> AI development session journal
> Started: 2026-09-11

---



## Session 1: Codex forwarding compatibility audit and fixes

**Date**: 2026-09-12
**Task**: Codex forwarding compatibility audit and fixes
**Branch**: `dev`

### Summary

Completed the Codex metadata and tool-call audit, local official CLI observation, compatibility fixes, independent review, and full backend verification. Changes remain uncommitted.

### Main Changes

- Preserve valid caller UA, originator, and version at Codex forwarding boundaries when enforcement is disabled; retain explicit overrides and synthetic identities.
- Keep WebSocket reuse compatible with complete handshake metadata; preserve native tool-call IDs independently of item-ID validation.
- Saved the audit and sanitized Codex CLI 0.153.4 exec observation in docs/audits and documented exact-version Ubuntu formatting evidence.

### Git Commits

(No commits - planning session)

### Testing

- [OK] Go 1.27.0: full backend unit suite passed, 56 package checks.
- [OK] Full backend integration suite passed, 50 package checks including PostgreSQL and Redis on local Colima.
- [OK] golangci-lint v2.13.0 reported 0 issues; service go vet, formatting, whitespace checks, and independent source review passed.

### Status

[OK] **Completed**


## Session 2: Chinese Codex fingerprint source reference

**Date**: 2026-09-12
**Task**: Chinese Codex fingerprint source reference
**Branch**: `dev`

### Summary

Completed docs/audits/sub2api-fingerprint-findings.zh-CN.md with 19 findings, code excerpts, activation conditions, and revision-specific status. Preserved the existing local fix commit 06bb4df4c; no runtime edits or new commits.

### Main Changes

- Added the Chinese source reference and linked it from the English audit; narrowly allowed the file in .gitignore.
- Addressed independent review findings on revision status, passthrough model gates, Messages template scope, and device_id precedence.

### Git Commits

(No commits - planning session)

### Testing

- [OK] Passed: 38 revision-pinned source blocks, 7 inline UA examples, 80 local links including 78 line anchors, and saved CLI sample comparison.
- [OK] Passed: git diff --check, document whitespace/fences, gitignore visibility, and clean staged/unstaged backend and deploy diffs.
- [OK] Documentation-only change: backend suites were not rerun; earlier results remain in the English audit. Existing forwarding spec required no changes.

### Status

[OK] **Completed**


## Session 3: Release v0.2.4-hotfix1: Codex 身份保持 + 中文指纹参考

**Date**: 2026-09-13
**Task**: Release v0.2.4-hotfix1: Codex 身份保持 + 中文指纹参考
**Branch**: `dev`

### Summary

完成 OpenAI Codex 网关身份保持实现与中文请求指纹参考文档,并发布 v0.2.4-hotfix1。

## 改动
- **Codex 身份保持** (06bb4df4c): 在 openai_codex_identity.go 新增 preserveCodexClientIdentityHeaders,
  网关/WS 池/forward/messages/passthrough 路径在身份强制关闭且无管理员 UA 时透传客户端
  User-Agent/Originator/Version,保留级联中继真实指纹;配套 3 个测试文件 + 2 个新增网关测试
  锁定 call-id 与服务身份行为。
- **文档** (feeeae916): 新增 docs/audits/sub2api-fingerprint-findings.zh-CN.md(609 行),
  涵盖身份/提示词/工具名/调用 ID/会话标识/传输层硬编码,逐项给出文件名、源码摘录、触发条件与
  当前状态;codex-request-compatibility.md 增加一行交叉链接;.gitignore 放行新文档。

## Release
- 打 annotated tag v0.2.4-hotfix1,push origin 触发 .github/workflows/release.yml (run 34680884187)
- 4 个 job 全部 success: update-version / build-frontend / release(GoReleaser) / sync-version-file
- GitHub Release:5 平台二进制 (linux/darwin × amd64/arm64, windows/amd64) + checksums.txt,
  prerelease=true(因 -hotfix1 后缀)
- GHCR 多架构镜像:ghcr.io/bingooyong/sub2api:0.2.4-hotfix1(amd64+arm64 合并 manifest)
  同时更新 :latest/:0.2/:0 浮动 tag
- VERSION 文件从 0.2.4 回写到 main
- DockerHub 未配置 secret(预期跳过)

## 验证
- docker pull ghcr.io/bingooyong/sub2api:0.2.4-hotfix1 = 330MB
- OCI labels: revision=feeeae916, version=0.2.4-hotfix1, source=bingooyong/sub2api 全部正确
- docker run -version: 嵌入 commit/built 元数据与 dev HEAD 完全一致
- 内嵌 Built=2026-09-12T07:32:22Z,介于 release job 07:31:37-07:42:54 时段内

### Git Commits

| Hash | Message |
|------|---------|
| `feeeae916` | (see git log) |
| `06bb4df4c` | (see git log) |

### Status

[OK] **Completed**
