# 更新 160 到 v0.2.21-hotfix1

## Goal

把本地定制分支 `v0.2.21-hotfix1`（HEAD `2f592f2f8`）构建成 `linux/amd64` 运行时镜像，
发布到 VPS 160（`144.202.121.160`），替换当前线上版本 `local/sub2api:0.2.18-hotfix1-23efd173b`，
并完成数据核对与回滚点留存。

## 背景

- 线上：`local/sub2api:0.2.18-hotfix1-23efd173b`（ID `sha256:26320bb2ee36…`），
  已稳定运行 12 天，`schema_migrations` = **301** 条。
- 本地：`v0.2.21-hotfix1`，HEAD `2f592f2f8`，`backend/cmd/server/VERSION` = `0.2.21-hotfix1`，
  工作区干净，**领先线上 276 个提交**。
- 本地分支已合并上游 `v0.2.21`（提交 `81231fa12 merge upstream v0.2.21 preserving fork
  customizations`），并在此之上叠加了定制修复（账号质量多账号检测、手动探针鉴权修复、
  分组成员恢复、lint 修复等）。
- 上游 `upstream/main` 另有 2 个未打 tag 的提交。**本次不合并**——未打 tag 的 main
  没有稳定性保证，等下一个 tag 再合。

## Requirements

- R1 **产物来源唯一**：从 `git archive 2f592f2f8` 的冻结快照构建，前端 dist 与后端二进制
  必须来自同一份快照。
- R2 **迁移审查先行**：逐个读完 5 条新迁移的 SQL，按「加表 / 加列 / 加索引 / 删改」分类，
  明确每条对线上数据的影响。
- R3 **定制语义保持**：`GATEWAY_DISABLE_CODEX_IDENTITY_ENFORCEMENT=true` 继续生效；
  上游新增的 TUI→CLI 归一化仍必须以 `codexIdentityEnforcement` 为闸门。
- R4 **零数据丢失**：迁移全部为加表 / 加列 / 加索引，无 DROP / RENAME / DELETE，
  不可逆点为零；回滚只需换回旧镜像。
- R5 **发布流程合规**：走 `.trellis/spec/ops/sub2api-160-deploy.md` 的完整流程
  （备份 → 隔离演练 → 切换 → 验证），停服窗口内完成离站备份校验。
- R6 **磁盘先行**：160 磁盘是当前最紧资源，发布前必须 `df -h /` 并按可回滚性分级清理。
- R7 **回滚点留存**：发布后保留当前版与上一版镜像及其配套 final 备份。

## 迁移审查结论（v0.2.18 → v0.2.21，301 → 306 条）

| 文件 | 类型 | 数据影响 | 可逆性 |
| --- | --- | --- | --- |
| `247_scoped_admin_api_keys.sql` | CREATE TABLE | 新表 `scoped_admin_api_keys`，存管理密钥的不可逆摘要与撤销记录；不含明文密钥 | 完全可逆（新表不影响旧代码） |
| `248_group_image_count_billing.sql` | ALTER TABLE ADD COLUMN | `groups.image_count_billing` 默认 false；仅 OpenAI / Composite 分组使用 | 完全可逆（加列带默认值，旧代码忽略） |
| `249_add_usage_logs_api_key_cost_index_notx.sql` | CREATE INDEX CONCURRENTLY | `usage_logs(api_key_id, created_at) INCLUDE (actual_cost)`，仅加索引 | 完全可逆 |
| `249_mcp_oauth.sql` | CREATE TABLE | 新表 `mcp_oauth_grants`（MCP OAuth 授权码 + S256 PKCE，不落明文令牌） | 完全可逆 |
| `250_openai_cache_write_ttl.sql` | ALTER TABLE ADD COLUMN × 多表 | `usage_logs` 加 2 个 cache_creation 计数字段；`channel_model_pricing` / `channel_pricing_intervals` / `channel_account_stats_model_pricing` 加 `cache_write_12h_price` | 完全可逆 |

**关键结论**

- **无 DROP / RENAME / DELETE / 数据回填** → schema 向前兼容，回退旧镜像不需要逆向迁移
  （旧代码不认识新列，但新列均带默认值且旧代码不查询它们）。
- **`249_..._notx.sql` 的事务风险已排除**：迁移器以 `nonTransactionalMigrationSuffix =
  "_notx.sql"` 识别该类文件，在**事务外**执行（`CREATE INDEX CONCURRENTLY` 不能在
  事务块内运行）。上游已有 5 个同类先例（120 / 153 / 174 / 195 / 226 / 233）。
- **命名空间重复合法**：`249_add_usage_logs_api_key_cost_index_notx.sql` 与
  `249_mcp_oauth.sql` 同为 249 前缀——迁移身份是**完整文件名**，两条都会执行。
- 线上 301 条基线的最后一条是 `246_enable_gemini_image_generation_groups.sql`，
  与上次发布一致。

## 定制语义复核（v0.2.21）

- `backend/internal/config/config.go:1104` `DisableCodexIdentityEnforcement` 仍在；
  `config.go:2050` 仍有 `cfg.Gateway.DisableCodexIdentityEnforcement = true`。
- `backend/internal/service/openai_codex_identity.go:164`：
  `if codexIdentityEnforcement.Load() { … NormalizeCodexClientIdentityToCLI … }`
  —— **我们在 v0.2.18 合并时加的闸门在 v0.2.21 中依然存在且未被上游覆盖** ✓
- `openai_codex_identity.go:238`（enforcement 关闭时走 `pairCodexIdentityHeaders`）
  与 `:272`（`preserveCodexClientIdentityHeaders`）均保持定制语义 ✓

## Constraints

- 160 只有 1C/1G，**绝不在 160 上编译**；本机交叉编译 + 160 只封装运行时镜像。
- 本机与 160 链路约 200KB/s，89MB 备份全量传输约 8 分钟 → 停服前必须先落好 rsync 底座。
- 远端 `deploy_remote.py --apply` 必须 `setsid nohup` 后台执行（160 是 AI 会话自身的
  推理上游，前台执行会在停服瞬间把会话一起杀掉）。
- 本次不合并上游 main 的 2 个未打 tag 提交。
- 发布产物不含 `.env`、数据目录、私钥、本地工具目录。

## Acceptance Criteria

- [ ] 线上镜像 tag = `local/sub2api:0.2.21-hotfix1-2f592f2f8`，OCI revision 与
      二进制 `-version` 的 commit 均为完整 40 位 `2f592f2f8…`
- [ ] `schema_migrations` 由 301 增至 **306**，新增的 5 条文件名逐条在库中存在
- [ ] 同名迁移条目的 checksum **逐条未变**（历史迁移未被篡改）
- [ ] 所有业务表的 count 与字段哈希与切换前 inventory **逐项一致**
- [ ] 管理员版本接口返回 `0.2.21-hotfix1`
- [ ] 两个 HTTPS 域名 `/health` 与本机 `/health` 均返回 `{"status":"ok"}`
- [ ] `/v1/models` 返回模型数与关键模型 ID（含 gpt-5.6 全家）
- [ ] UI 首页与静态 assets 均为 200，且资源文件名是本轮构建产物
- [ ] 启动后日志中 panic / fatal / ERROR 为 0
- [ ] 隔离演练通过：备份可恢复、迁移 301→306、核心表计数一致
- [ ] 演练容器 / 卷 / 网络已清理；旧镜像与本轮完整备份仍保留
- [ ] `docs/Sub2API-GPT-5.6适配记录.md` 与 `AGENTS.md` 的当前版本、备份路径、
      manifest SHA 已更新

## Notes

- 本地 276 个提交**尚未推送到 origin**，发布是否同时 push 由用户决定。
- 149 节点从本机不通（自举死锁相关，已单独排查）**与本次发布无关**——发布链路只依赖
  本机 → 160 的 SSH 与 160 → 上游。
