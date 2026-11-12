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


## Session 4: Account quality operations

**Date**: 2026-09-30
**Task**: Account quality operations
**Branch**: `dev`

### Summary

Implemented opt-in scheduled account quality probes with independent bounded judge retries, non-mutating ambiguous outcomes, transactional isolation and conflict-aware restore. Added durable redacted operation audit history, admin API/UI, intelligence-test support, migrations, tests, and code-spec documentation; preserved local forwarding identity/fingerprint/WebSocket behavior.

### Git Commits

| Hash | Message |
|------|---------|
| `c11317269` | (see git log) |

### Status

[OK] **Completed**


## Session 5: Upgrade to v0.2.14-hotfix1

**Date**: 2026-10-08
**Task**: Upgrade to v0.2.14-hotfix1
**Branch**: `dev`

### Summary

Merged pinned upstream v0.2.14, preserved local Codex and account operations behavior, fixed Wire generation, added realistic migration backfill coverage, updated frontend expectations, and passed backend/frontend/migration/build verification.

### Git Commits

| Hash | Message |
|------|---------|
| `32f15a78e` | (see git log) |
| `8986de74e` | (see git log) |

### Status

[OK] **Completed**


## Session 6: Upgrade to v0.2.17-hotfix1

**Date**: 2026-10-23
**Task**: Upgrade to v0.2.17-hotfix1
**Branch**: `dev`

### Summary

Merged upstream v0.2.17, preserved local Codex identity and account quality operations, verified encrypted reasoning recovery for server 160 errors, passed full backend tests/build, and recorded frontend/live database environment gaps.

### Git Commits

| Hash | Message |
|------|---------|
| `8934e084b` | (see git log) |
| `cac4d290e` | (see git log) |

### Status

[OK] **Completed**


## Session 7: 修复 OpenAI 加密推理密文失效的 HTTP 路径自愈判定

**Date**: 2026-10-23
**Task**: 修复 OpenAI 加密推理密文失效的 HTTP 路径自愈判定
**Branch**: `v0.2.17-hotfix1`

### Summary

HTTP 路径的 invalid_encrypted_content 自愈只认 error.code，上游返回 code=null 时判定落空导致对话卡死；改为码+文案双通道判定并与 WS 路径共用同一函数

### Main Changes

- 新增 backend/internal/service/openai_invalid_encrypted_content.go：isOpenAIInvalidEncryptedContentUpstreamError(code, msg)
- openai_gateway_forward.go：HTTP 路径改用共享判定，日志补充 upstream_code
- openai_ws_forwarder_support.go：WS 路径改用同一判定函数，消除两条路径口径漂移
- 新增 openai_invalid_encrypted_content_test.go：判定表驱动 + HTTP 集成 + WS 一致性
- 新增 .trellis/spec/backend/openai-encrypted-reasoning-recovery.md（code-spec 7 节）

### Git Commits

(No commits - planning session)

### Testing

- [OK] go build ./... 通过；go vet ./internal/service/ 无告警；gofmt 无差异
- [OK] go test ./internal/service/ -count=1 全绿（150s，exit 0）
- [OK] 定向用例：code=null 生产形态自愈重试成功、无关 400 不重试、连续 400 只重试一次

### Status

[OK] **Completed**

### Next Steps

- 发版到 160（走 vutl 仓库 .trellis/spec/ops/sub2api-160-deploy.md 流程）——本任务按用户要求不发版
- 根因层面的账号调度策略（同一会话固定同一账号）另立任务


## Session 8: 修复加密推理密文失效自愈并发布到 160（87d78f265）

**Date**: 2026-10-23
**Task**: 修复加密推理密文失效自愈并发布到 160（87d78f265）
**Branch**: `dev`

### Summary

定位两种密文失效形态（code=null 与 message 项 hydration failed），修复判定与恢复动作并发布到 160，停服 101 秒

### Main Changes

- 新增 openai_invalid_encrypted_content.go：双通道判定 + 剥 message 密文 + 解析失效 item id
- openai_gateway_forward.go：恢复动作扩为三类（剥 reasoning/compaction、剥 message、丢 previous_response_id）
- openai_ws_forwarder_support.go：WS 路径共用同一判定，消除口径漂移
- 新增 openai_invalid_encrypted_content_test.go（6 组用例）
- spec：openai-encrypted-reasoning-recovery.md 补三种恢复动作与 message 密文形态

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test ./internal/service/ -count=1 全绿（151s，exit 0）
- [OK] 隔离演练：migrations 300 不变、accounts 6859/users 1/keys 7/groups 8 一致
- [OK] 切换后：版本接口 0.2.17-hotfix1、三个 health ok、UI 200、/v1/models 19 个模型、核心表 inventory 一致、日志 0 错误

### Status

[OK] **Completed**

### Next Steps

- 请用户在带失效密文的会话中复现，确认自愈生效
- 演练脚本的 health 检查失败未导致非零退出，建议下次修正


## Session 9: 加密推理密文自愈：补兜底与诊断日志，发版 5d30c3087

**Date**: 2026-10-23
**Task**: 加密推理密文自愈：补兜底与诊断日志，发版 5d30c3087
**Branch**: `dev`

### Summary

上一版上线后仍复现且无自愈日志，改为加足诊断信息并补最后一道兜底（丢弃 assistant 侧历史），停服 109 秒发布

### Main Changes

- 新增 stripOpenAIAssistantHistoryForInvalidEncryptedContent：最后一道兜底，丢弃 assistant 侧历史保留 user 消息与工具输出
- 新增 openAIInputShapeSummary/FromBytes：诊断用 input 结构摘要（只含类型/有无 id/密文字段）
- forward.go 400 分支与 handleErrorResponse 入口各加一条无条件诊断日志
- spec 补第四级恢复动作（丢弃 assistant 侧历史）

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test ./internal/service/ -count=1 全绿（150s）
- [OK] 切换后：停服 109 秒、版本接口 ok、三个 health ok、UI 200、日志 0 错误

### Status

[OK] **Completed**

### Next Steps

- 请用户在带失效密文的会话中重试，用诊断日志确认自愈是否生效


## Session 10: 合并上游 v0.2.18 并发布 0.2.18-hotfix1

**Date**: 2026-10-23
**Task**: 合并上游 v0.2.18 并发布 0.2.18-hotfix1
**Branch**: `dev`

### Summary

合并上游 45 个提交（含 dompurify 安全升级），适配 Codex 身份定制冲突，迁移 246 影响 0 行，停服 95 秒发布

### Main Changes

- 合并上游 v0.2.17..v0.2.18 的 45 个提交到 v0.2.18-hotfix1 分支
- 定制适配：上游把默认 Codex 身份由 codex-tui 改为 codex_cli_rs（issue #5198），归一化以 codexIdentityEnforcement 为闸门，只在强制统一路径生效
- VERSION 定为 0.2.18-hotfix1；迁移 246 审查确认对线上 8 个分组影响 0 行
- 解决 pnpm 11 构建坑：pnpm.* 设置迁到 pnpm-workspace.yaml，overrides 逐字迁移 + onlyBuiltDependencies 放行 esbuild/vue-demi
- docs 与 AGENTS.md 更新当前版本、备份路径、manifest、迁移审查结论

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test ./internal/service/ -count=1 全绿（150s）
- [OK] 隔离演练：迁移 300→301、246 已应用、accounts 6859/users 1/keys 7/groups 8 一致
- [OK] 切换后：停服 95 秒、版本接口 0.2.18-hotfix1、三个 health ok、UI index-BcCu4tlM.js 200、/v1/models 19 个模型
- [OK] schema_migrations 仅新增 246、同名 checksum 未变、所有业务表 inventory 逐项一致、日志 0 错误

### Status

[OK] **Completed**

### Next Steps

- 请用户在旧会话（带失效密文）重试，验证密文自愈是否生效；必要时用 160 上的诊断日志定位
- 评估是否关闭 GATEWAY_DISABLE_CODEX_IDENTITY_ENFORCEMENT：上游认定 codex-tui 会被风控分层（server_is_overloaded）


## Session 11: 沉淀 v0.2.18 合并与密文自愈的 code-spec

**Date**: 2026-10-23
**Task**: 沉淀 v0.2.18 合并与密文自愈的 code-spec
**Branch**: `dev`

### Summary

新增 Codex 身份定制与前端构建工具链两份 code-spec，密文 spec 补终极兜底与诊断对象要点

### Main Changes

- 新增 .trellis/spec/backend/codex-identity-customization.md（7 节）
- 新增 .trellis/spec/frontend/build-toolchain.md（7 节，pnpm 11 设置迁移）
- openai-encrypted-reasoning-recovery.md 补第 5 级兜底、§3.5 诊断对象、签名与断言点
- backend/index.md 与 frontend/index.md 登记新 spec 并补 Quality Check 指针

### Git Commits

(No commits - planning session)

### Testing

- [OK] spec 引用的全部函数与常量已逐一校验存在于代码（无 MISSING）

### Status

[OK] **Completed**

### Next Steps

- 把 pnpm-workspace.yaml 迁移提交进仓库（当前只是构建时在 staging 临时创建）


## Session 12: 沉淀粘性会话与上游错误处理的根因契约

**Date**: 2026-10-23
**Task**: 沉淀粘性会话与上游错误处理的根因契约
**Branch**: `dev`

### Summary

查明密文失效的根因链（粘性绑定被清除→换账号→密文必死），并补齐上游错误分类与验证状态契约

### Main Changes

- 新增 backend/openai-sticky-session-encrypted-content.md：粘性绑定生命周期与 9 类清除条件
- 新增 backend/openai-upstream-error-handling.md：多信号错误分类、重试预算、ops 事件语义
- openai-encrypted-reasoning-recovery.md 新增「当前验证状态」章节，区分已验证与待确认
- error-handling.md 从空白骨架补出上游错误处理要点与常见错误（To fill → Partial）
- backend/index.md 与 vutl ops/index.md 登记新 spec 并补指针

### Git Commits

(No commits - planning session)

### Testing

- [OK] 新 spec 引用的全部函数与常量已逐一确认存在

### Status

[OK] **Completed**

### Next Steps

- 验证粘性会话假设：把 scheduler 的 session_hash 绑定与 ops_error_logs 的 account_id 按 client_request_id 关联


## Session 13: 线上验证密文自愈生效并修正根因结论

**Date**: 2026-10-23
**Task**: 线上验证密文自愈生效并修正根因结论
**Branch**: `dev`

### Summary

生产实测确认自愈已生效（19 次连续成功 0 错误），同时推翻「粘性会话从未写入」的错误结论

### Main Changes

- 密文 spec：验证状态改为已验证生效，补生产时间线与 Recovered 前缀的判读方法
- 粘性会话 spec：改为实测证明粘性会话正常工作，保留两种候选根因与不可回溯的原因
- vutl ops spec：新增「Redis 审计要用 KEYS 不要信 --scan」一节
- 记忆文件补第 7 条（Redis 假阴性观测陷阱）

### Git Commits

(No commits - planning session)

### Testing

- [OK] 线上验证：最后一次密文错误 13:28 UTC，此后 13:29–14:29 期间 19 次请求全部成功、0 密文错误
- [OK] Redis 实测：10 个 sticky_session 绑定指向真实 account_id；探针请求当场创建新绑定

### Status

[OK] **Completed**

### Next Steps

- 若要区分两种候选根因，需要在 ops_error_logs 里记录 session_hash（当前缺失）


## Session 14: 密文失效自愈上线 + session_hash 诊断增强（23efd173b）

**Date**: 2026-10-23
**Task**: 密文失效自愈上线 + session_hash 诊断增强（23efd173b）
**Branch**: `dev`

### Summary

把可观测性缺口补上：诊断日志带 session_hash，可与 Redis 粘性绑定对照；同时把 pnpm 11 设置迁入 pnpm-workspace.yaml 修 CI 隐患

### Main Changes

- service: 新增 openAISessionHashContextKey 与 Set/GetOpenAISessionHashToContext
- handler: 三处生成 session hash 的位置写入 gin context（responses / messages / WS ingress）
- forward: 400 诊断日志新增 session_hash 字段
- 新增 frontend/pnpm-workspace.yaml：overrides 与 package.json 逐字一致 + onlyBuiltDependencies
- spec: 密文 spec 诊断节、粘性会话 spec §7.1 可观测性契约、build-toolchain spec 仓库现状

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test ./internal/service/ 全绿（149s）；go test ./internal/handler/ 全绿（41s）
- [OK] 发布后验证：停服 104 秒、无迁移、前端资源未变、探针请求 200、日志 0 错误

### Status

[OK] **Completed**

### Next Steps

- 若将来再复现密文失效，用诊断日志的 session_hash 对照 redis-cli KEYS 'sticky_session:*' 即可判定是否换过账号


## Session 15: Upgrade v0.2.21 and manual quality detection

**Date**: 2026-11-12
**Task**: Upgrade v0.2.21 and manual quality detection
**Branch**: `v0.2.21-hotfix1`

### Summary

Merged pinned v0.2.21 into v0.2.21-hotfix1 preserving fingerprint and identity customizations. Added non-mutating manual multi-account bicycle/candy/custom probes, bounded cancellation-safe comparison cards, strict SSE handling, sandbox previews, transport and branding regressions. Fixed pre-existing scheduled group restore SQL/schema mismatch. Frontend full 3143 tests, lint/typecheck/build, backend full unit/vet, fresh PostgreSQL/Redis upgrade/idempotency/restore integration, final 0-issue lint and embedded binary/version checks passed. Mock desktop/mobile browser smoke passed; no live GPT calls, deployment or push.

### Git Commits

| Hash | Message |
|------|---------|
| `81231fa12` | (see git log) |
| `61dbfc2b4` | (see git log) |
| `6d96a1ee8` | (see git log) |
| `f5109cf14` | (see git log) |

### Status

[OK] **Completed**


## Session 16: Review and repair v0.2.21 manual quality detection

**Date**: 2026-11-12
**Task**: Review and repair v0.2.21 manual quality detection
**Branch**: `v0.2.21-hotfix1`

### Summary

Repaired six review findings and two independent followups; preserved fingerprint/branding contracts, verified final UI and binary, archived only this task.

### Main Changes

- Passive previews with CSS/resource/navigation filtering; shared 401 refresh, 423 compliance dialog, disclosure focus/ARIA.
- Observational OAuth cache/renewal with adopted-version expiry guard; known non-text mapped-model rejection and custom text compatibility.

### Git Commits

| Hash | Message |
|------|---------|
| `c58eab184` | (see git log) |

### Testing

- [OK] Final backend targeted service/admin regressions, full vet and complete-report golangci-lint: 0 issues.
- [OK] Frontend full393/3158 before narrow CSS followup; final affected51/helper11, lint/types/locale/build pass.
- [OK] Real Chromium eleven-fixture production/no-parentCSP preview and final production UI smoke pass; embedded binary reports0.2.21-hotfix1.
- [OK] Core fingerprint/identity files byte-identical to pre-upgrade; no migration/dependency changes; no live GPT/push/deploy.

### Status

[OK] **Completed**


## Session 17: 更新 160 到 0.2.21-hotfix1（合并上游 v0.2.19/20/21）

**Date**: 2026-11-12
**Task**: 更新 160 到 0.2.21-hotfix1（合并上游 v0.2.19/20/21）
**Branch**: `v0.2.21-hotfix1`

### Summary

本地 v0.2.21-hotfix1 构建发布到 160，迁移 301→306，停服 122 秒，业务表 8 张逐项一致

### Main Changes

- 上游 v0.2.19/20/21 合并 + 定制修复，产物修订 c9677aee4，镜像 ID eeb37ef6edaa
- 5 条新迁移全部为 CREATE TABLE / ADD COLUMN / CREATE INDEX CONCURRENTLY，无 DROP/RENAME/DELETE → 回滚无需逆向迁移
- 隔离演练改用 schema-only 恢复 + 迁移账本恢复（生产库 1447MB，160 磁盘放不下全量恢复）
- 演练脚本补：health 失败即非零退出、PG 未就绪时打印容器日志与磁盘
- 分支已推送 origin/v0.2.21-hotfix1

### Git Commits

(No commits - planning session)

### Testing

- [OK] 演练：migrations 301→306、5 条新迁移逐条在库、新表/列/索引全部创建、health ok
- [OK] 切换后：停服 122 秒、版本接口 0.2.21-hotfix1、三个 health ok、UI index-CuuCm_D7.js 200、/v1/models 23 个模型
- [OK] schema_migrations 仅新增 5 条且同名 checksum 未变；8 张业务表 inventory 逐项一致；日志 error/fatal/panic 为 0

### Status

[OK] **Completed**

### Next Steps

- 重建被误删的上一版镜像 0.2.18-hotfix1-23efd173b 以补齐回滚点（后台进行中）


## Session 18: 补齐回滚点与 pnpm 11 allowBuilds 修正

**Date**: 2026-11-12
**Task**: 补齐回滚点与 pnpm 11 allowBuilds 修正
**Branch**: `v0.2.21-hotfix1`

### Summary

重建被误删的上一版回滚镜像；修正 pnpm-workspace.yaml 使 pnpm 11 的 allowBuilds 与旧版 onlyBuiltDependencies 并存

### Main Changes

- 重建 local/sub2api:0.2.18-hotfix1-23efd173b（ID e066f9ed1234）补齐回滚点
- frontend/pnpm-workspace.yaml 同时保留 onlyBuiltDependencies 与 allowBuilds，兼容 pnpm 10/11
- spec 补「重建旧 commit」一节：要用该 commit 自己的 overrides 生成 yaml

### Git Commits

(No commits - planning session)

### Testing

- [OK] 仓库版配置实测：install 与 build 全通过，dompurify=3.4.17，dist 正常产出
- [OK] 重建镜像 -version 断言通过（0.2.18-hotfix1, commit 23efd173b…）

### Status

[OK] **Completed**
