# 技术设计：更新 160 到 v0.2.21-hotfix1

## 1. 现状与目标

| 维度 | 线上（当前） | 本地（目标） |
| --- | --- | --- |
| 版本 | `0.2.18-hotfix1-23efd173b` | `0.2.21-hotfix1-2f592f2f8` |
| 提交 | `23efd173b8aaddf342da98a8a24c9516a31405fe` | `2f592f2f8…`（领先 276 个提交） |
| 上游基线 | v0.2.18 | **v0.2.21**（已合并） |
| `schema_migrations` | 301 | 306 |

发布形态不变：本地交叉编译 → 160 封装运行时镜像 → 换 Compose 的 image 行。

## 2. 关键取舍

### 2.1 不合并上游 main 的 2 个未打 tag 提交

`upstream/main` 比本地分支领先 2 个提交，但**没有对应的 release tag**。未打 tag 的 main
提交不具备稳定性保证，且一旦引入就要再走一遍审查与演练。选择等下一个 tag 再合，
避免在发布窗口里叠加无版本锚点的变更。

### 2.2 本次的迁移是「纯向前」，回滚不需要逆向迁移

与历史版本（235/236 的 `RENAME COLUMN`）不同，本次 5 条迁移全部是
**CREATE TABLE / ADD COLUMN / CREATE INDEX CONCURRENTLY**：

- 无 `DROP COLUMN` / `DROP TABLE` / `RENAME` / `DELETE` / 数据回填
- 新列均带 `DEFAULT` 且旧代码不查询它们 → 旧镜像可安全读新 schema
- 因此**回滚 = 换回旧镜像 + 重启**，不需要设计逆向迁移、不需要动 `schema_migrations`
  的已执行记录

这条结论是本次回滚成本低的前提，必须在演练里验证「迁移后旧镜像能正常启动并读库」。

### 2.3 `CREATE INDEX CONCURRENTLY` 的事务风险已排除

`249_add_usage_logs_api_key_cost_index_notx.sql` 使用了 `CONCURRENTLY`，
该语句**不能在事务块内执行**。迁移器以
`nonTransactionalMigrationSuffix = "_notx.sql"` 识别此类文件并在事务外执行
（`backend/internal/repository/migrations_runner.go`），上游已有 6 个同类先例
（120 / 153 / 174 / 195 / 226 / 233）。**无需额外处理。**

### 2.4 前端 dist 的复用判据

spec 要求「dist 与后端二进制来自同一份源码快照」。本轮前端源码相对上一版**是否有改动**
决定是重建还是复用：

- 有改动 → 必须在快照内重建（`pnpm install` + `pnpm run build`），
  注意 pnpm 11 的设置已迁入 `frontend/pnpm-workspace.yaml`（与 package.json 逐字一致），
  且 pnpm 10 的 lockfile 无法通过 pnpm 11 的 `--frozen-lockfile` 校验 →
  用 `--no-frozen-lockfile` 并**显式断言钉版版本已落地**（构建成功 ≠ overrides 生效）
- 无改动 → 可复用上一轮 dist，但必须用
  `git diff --stat <旧发布 commit>..<新 commit> -- frontend/src` 证明为空，
  并在发布记录里写明这个等价性论证

### 2.5 停服窗口策略

沿用 2026-10-23 验证过的三件套，目标把停服压到 2–3 分钟：

1. **停服前**把已校验的在线备份复制成本机 `<RUN>/final/` 作为 rsync 增量底座
   （链路仅约 200KB/s，全量 89MB 要 8 分钟）
2. 本机 `nohup auto_offsite_verify.py` 先起（会话死了它也活着）
3. 远端 `deploy_remote.py --apply` 用 `setsid nohup` **后台执行**
   （前台执行会在停服瞬间把发起部署的 AI 会话一起 502 杀掉，发布流程跟着死）

实测参考：2026-10-23 停服 95–240 秒。

### 2.6 磁盘管理按「可回滚性」分级

160 根分区 23G，发布后约 90% 已用。清理顺序：

1. 中间产物（本次构建目录、历史 release tarball）——可无损重建
2. **已不可安全回滚**的旧镜像与配套旧备份（schema 已变 300+ migrations，且可由源码重建）
3. 被取代的中间版本镜像与备份

**保留**：当前版与上一版镜像 + 其配套 final 备份 + `sub2api-backups/*` 里的最新一次 final。
隔离演练恢复数据库要额外约 1GB，磁盘不足会直接以 `No space left on device` 失败。

## 3. 风险矩阵

| 风险 | 可能性 | 影响 | 缓解 |
| --- | --- | --- | --- |
| 磁盘不足导致演练失败 | 中 | 阻断发布 | 阶段 A2 先清理；演练前复查 `df -h /` |
| `_notx` 迁移在事务内执行失败 | 低 | 迁移中断、启动失败 | 迁移器已有专门机制；演练验证 301→306 |
| 前端 dist 与后端不同源 | 低 | UI 与后端版本不匹配 | A5 用 git diff 证明；必要时重建 |
| 定制开关在上游合并中被覆盖 | 低 | Codex 身份语义被破坏 | A4 复核门控代码；发布后核对请求身份头 |
| 停服窗口超时 | 低 | 会话被杀、发布中断 | rsync 底座 + 后台 apply + 验证器独立进程 |
| 上游 276 个提交引入回归 | 中 | 功能异常 | 隔离演练 + 发布后接口/UI/数据三重复核 |
| 回滚需要逆向迁移 | 极低 | — | 本次无 DROP/RENAME，实测旧镜像读新 schema |

## 4. 回滚策略

- **触发条件**：health 不恢复 / 版本接口异常 / 核心表数据不一致 / 日志出现 panic
- **动作**：`deploy_remote.py` 自带 health 超时自动回滚；人工回滚 =
  Compose 换回 `local/sub2api:0.2.18-hotfix1-23efd173b` + `docker compose up -d --no-deps`
- **不需要**：逆向迁移、改 `schema_migrations`、恢复旧 dump
  （旧 dump 会丢弃升级后写入，除非用户明确要求，否则不做）
- **前提**：旧镜像与本轮 final 备份必须在回滚点留存完成前不被清理

## 5. 交付物

- 线上镜像 `local/sub2api:0.2.21-hotfix1-2f592f2f8`
- `docs/Sub2API-GPT-5.6适配记录.md` 新增本次升级章节
- vutl `AGENTS.md` 更新当前版本 / 备份路径 / manifest SHA
- 两侧 journal 记录本次会话
- vutl `ops` 层 spec 增补本轮新经验（如有）
