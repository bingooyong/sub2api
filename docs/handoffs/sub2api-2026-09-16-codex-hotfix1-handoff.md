# Sub2API Codex hotfix1 交接 — 2026-09-16

## 当前状态与目标

- 仓库：`/Users/bingooyong/Code/01Code/github.com/bingooyong/sub2api`。
- 工作分支：`dev`；跟踪分支：`origin/dev`；未建立单独的功能分支。
- 本次代码 HEAD：`f2b15cdfc9e4bca8129334788374efd96b1a5e62`。之后可能只有交接文档提交。
- 本地 `origin/dev` 引用：`feeeae9166eeb891d3d9b0cf59e60a1493522180`；检查时 dev ahead 114。未 fetch，因此不代表远端实时状态。
- 目标版本：`v0.2.5-hotfix1`。本地修复已提交，Linux amd64 包已生成；发布一致性和 Linux 运行验证仍待完成。
- 未推送本轮代码、未移动标签、未创建远程 Release/GHCR 镜像、未部署生产。
- 交接前无已跟踪文件修改。存在用户本地未跟踪工具目录：`.agent/`、`.agents/`、`.cursor/`、`.zcode/` 和部分 `.trellis/` 文件；不要批量加入提交或清理。
- Trellis 无当前活动任务；相关任务已经归档到 `.trellis/tasks/archive/2026-09/`。不要重新归档或关闭无关的 `00-bootstrap-guidelines`。

## 用户约束

- 保留 Codex 指纹、账号/API Key 隔离和内部稳定哈希命名空间。
- 用户不需要历史标记兼容，接受全部开启新会话；这不授权删除历史数据库数据、重置账号 seed 或终止其他会话。
- 抓包及分析放项目内，不能放 `/tmp`；原始网络数据只留本地且 Git 忽略。
- 版本沿用 `v0.2.5-hotfix1`，不要自行递增 hotfix2。
- 已授权本地代码、测试及提交；发布到外部或部署须按具体授权范围判断。

## 本轮过程与提交

1. 上游 v0.2.5 已合并，原指纹修改保留。
2. `175875895`：新生成图片/Spark/todo 标记去品牌化，Python 工具别名改为 `python__codex`，Responses-shaped Chat 保留独立 tool-call ID，native WS 开始暂存指纹。
3. 用户以 tcpdump 捕获现有 Codex 经本地代理的连接。快照及脱敏分析已保存，之后审查 WS 分支发现首帧/后续帧缺失收敛应用。
4. `25fc7d27e`：修复 v2 passthrough 头/体指纹不一致；共享 WS helper 每帧复制 staged IDs，仅清空原始 body session 捕获状态，避免默认缓存键沿用首帧判断；新增入口回归测试。
5. `f2b15cdfc`：删除旧品牌 marker 的四处幂等识别条件；更新规范、审计和 changelog。仍使用新标记判断幂等。
6. 生成本地 Linux amd64 压缩包，包含嵌入前端的二进制、resources 和 entrypoint。安装工具版本不匹配导致非锁定依赖构建，见下文。
7. 本次交接只补齐操作入口和事实记录，不继续修改运行代码或部署。

## 关键实现

- `backend/internal/service/openai_codex_identity.go`：身份配置/透传。
- `backend/internal/service/openai_codex_fingerprint.go`：稳定派生、staging、`applyOpenAIWSFingerprintClientMetadata`。
- `backend/internal/service/openai_ws_forwarder_ingress.go`：先暂存、再选择转发分支；pooled 分支也用逐帧 helper。
- `backend/internal/service/openai_ws_v2_passthrough_adapter.go`：首帧及后续 `response.create` / `session.update` 在 account scoping 后应用指纹。
- `backend/internal/service/openai_ws_v2_fingerprint_test.go`：实际入口、握手和多次写入验证。
- `backend/internal/service/openai_codex_transform.go`、`openai_messages_todo_guard.go`：仅识别当前无品牌标记。
- `.trellis/spec/backend/codex-forwarding.md`、`docs/audits/codex-request-compatibility.md`、`CHANGELOG.md`：契约和审计。

删除旧 marker 检测不等于过滤调用方文本。若调用方显式发送旧标记，它仍可能原样出现在正文；此次未实现历史文本全局清洗或拒绝规则。Go module 路径、内部命名空间、其他供应商 UA 中仍可能出现 sub2api。

动态 OS/终端 UA 生成未实现；当前复用已有 UA/version 配置与身份透传机制。没有修改 TLS/HTTP2 栈，不能宣称完全模拟官方 CLI。

## 抓包证据（仅本地）

目录：`docs-local/codex-wire-audit-2026-09-16/`，目录权限 0700，快照 0600，Git 忽略。

- 快照：`codex-process.snapshot.pcapng`，56,880,808 字节。
- SHA-256：`7dff40a5ee316726fda3a7342c0dfaf99dd7dfb44075e6efe873c11a13e46c82`。
- 复核命令：`python3 docs-local/codex-wire-audit-2026-09-16/analyze_capture.py`。
- 结果：40,771 packets，127.545 秒，71 个可归属目标连接，71 个 CONNECT 200。
- CONNECT UA：`codex-tui/0.153.4 (Mac OS 26.6.2; arm64) iTerm.app/3.4.22 (codex-tui; 0.153.4)`。
- 71 个 ClientHello 同型；70 个可见 ServerHello 为 TLS1.2 / `0xc02c`；无 ALPN。
- 存在 40 个重组间隙、1 个带 RST 的连接；不能当作请求失败数。
- 仅观察本机到中转站，未解密内层 HTTPS UA、请求体、SSE/WS，未捕获网关到 OpenAI 的 TLS。
- WS bug 是源码与测试发现，不能说在加密抓包里看到了它。
- `tcpdump -r ... -nn | wc -l` 与解析器计数一致；Apple `--count` 曾显示 0，不应采用该结果。

## 已验证结果与范围

`25fc7d27e` 对应 WS 修复：

- 修复前 device/session/full 回归失败，修复后通过；off/API Key 对照通过。
- `go test -tags=unit -p 2 ./internal/service -count=1` 通过，206.682 秒。
- `go test -tags=integration -p 2 ./internal/service -run '^(TestPassthroughFingerprint_|TestWSFingerprint_)' -count=1` 通过，7.434 秒；不是完整数据库/container 集成测试。
- `go vet -tags=unit ./internal/service` 通过。
- CI 指定 `golangci-lint v2.13.0 run --timeout=30m ./...` 通过，0 issues。
- Trellis check 独立审查完成；唯一 errcheck 问题在测试 defer CloseNow 中修复。

`f2b15cdfc` 旧 marker 检测删除：

- 相关图片桥接、Spark、Todo 单测通过（1.489 秒）；gofmt/diff-check 通过。
- 本次删除复用已有测试，没有新增回归文件；未在此提交之后重跑完整 service 单测和 lint。不要把上一提交的全套验证表述为最终提交的完整验证。

日志在上述 docs-local 目录：`service-unit.log`、`service-vet.log`、`ws-integration.log`、`golangci-final.log`、`ws-fingerprint-before.log`、`ws-fingerprint-after.log`。

前端：i18n 三项检查、vue-tsc、Vite 构建通过，但依赖来源存在下面的可复现性问题。

## 打包现状与必须纠正的前文表述

本地产物：`release/sub2api-v0.2.5-hotfix1-linux-amd64.tar.gz`（Git 忽略）。

- 文件大小：37,716,742 字节。
- SHA-256：`952629d577de7536e55239af1345f3253da8db162f79c106e21d71fc41cb7aed`。
- ELF x86-64 静态二进制；版本字符串 `0.2.5-hotfix1`，源码修订 `f2b15cdfc`。
- 构建注入 `main.Version`、`main.Commit`、`main.BuildType=release`，未注入 `main.Date`，默认 unknown。
- macOS 上运行 Linux ELF 返回 exec format error；只验证了格式、内嵌字符串和摘要，没有 Linux `-version` 或启动健康检查。

**发布前待办：**

1. CI/Docker 固定 pnpm 9，本地误用 pnpm 11.16.0。它忽略 package.json 的 pnpm.overrides，frozen 安装报配置不匹配。后续非 frozen 安装改变了依赖解析并用于前端构建。虽然最终恢复了 Git lockfile，已经生成的前端/二进制仍来源于那次非锁定安装；包不能视作严格由提交锁文件复现，建议正式发布前用 pnpm 9 + frozen lockfile 重建。
2. `frontend/node_modules` **仍存在**，不是上一条最终回复声称的“已清理”。临时 pnpm-workspace.yaml 已删除，lockfile 已恢复；现有 node_modules 不能据此视为匹配锁文件。
3. `backend/cmd/server/VERSION` 仍为 `0.2.4`。此次包版本靠 ldflags 覆盖；普通构建可能回退到旧版本。
4. 本地 `v0.2.5-hotfix1` 标签仍指向 `175875895dcd3e9ef3bb1843eeb3216bce045986`，不包含后两个修复。不要按该标签部署，也不要未检查远端发布状态就强制移动。
5. 未创建发布镜像、未进行 Linux runtime smoke、未上传包、未部署到公网服务。

之前安装超时发生在 npm 镜像最后几个包，重试利用缓存后成功；不要为解决 frozen mismatch 再次默认使用 `--no-frozen-lockfile`。最初有从仓库根目录直接运行 Go 的失败命令，Go module 在 backend，运行时使用正确工作目录。

## 下一会话启动

先从仓库根目录执行只读检查：

```bash
cat HANDOFF.md
cat docs/handoffs/sub2api-2026-09-16-codex-hotfix1-handoff.md
git status -sb
git log -5 --oneline
git rev-parse HEAD 'v0.2.5-hotfix1^{}'
cat backend/cmd/server/VERSION
shasum -a 256 release/sub2api-v0.2.5-hotfix1-linux-amd64.tar.gz
```

如果继续正式打包，在独立构建目录使用 CI 同版本 pnpm 9，从提交中的原 lockfile frozen 安装后运行 `pnpm run build`。保留当前包以便对比，不覆盖用户文件。Go 模块目录为 backend；显式注入版本、完整修订和构建时间，使用 `-tags=embed`、`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`，随后在 Linux/容器上执行 `-version` 并验证打包内容。补跑最终源码的相关测试/lint，记录新的摘要。

如用户要求部署，可读取全局 skill `/Users/bingooyong/.codex/skills/sub2api-local-deploy/SKILL.md`，按其备份、迁移、切换和验证流程执行；单凭本交接不触发部署。

交接流程来源：`/Users/bingooyong/.codex/skills/handoff/SKILL.md`。本仓库没有 `docs/ai-knowledge/`，未新建重复知识库。
