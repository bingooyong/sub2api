# 实施计划：更新 160 到 v0.2.21-hotfix1

## 阶段 A：准备与审查（不动生产）

- [ ] A1 确认基线：本地 HEAD `2f592f2f8`、VERSION `0.2.21-hotfix1`、工作区干净；
      线上 `0.2.18-hotfix1-23efd173b` healthy、`schema_migrations` = 301
- [ ] A2 磁盘与备份盘点：`df -h /`、`du -sh /root/sub2api-backups/*/`、`docker images`
      → 按「可回滚性分级」清理（中间产物 → 已不可回滚的旧镜像/旧备份），
      **保留当前版与上一版镜像 + 配套 final 备份**
- [ ] A3 迁移审查：读完 5 条新迁移 SQL（prd 已有结论），确认无 DROP/RENAME/DELETE
- [ ] A4 定制复核：确认 `codexIdentityEnforcement` 门控、`DisableCodexIdentityEnforcement`
      默认值、`preserveCodexClientIdentityHeaders` 在 v0.2.21 中未被上游覆盖
- [ ] A5 构建输入检查：前端是否有源码改动？
      - 有 → 必须重建 dist（同一快照内 `pnpm install` + `pnpm run build`）
      - 无 → 可复用上一轮 dist，但需用 `git diff --stat <旧>..<新> -- frontend/src` 证明为空

## 阶段 B：构建产物

- [ ] B1 冻结快照：`git archive 2f592f2f8 | tar -x -C <RUN>/staging`
- [ ] B2 前端 dist：按 A5 结论产出（重建或复用），落在 `staging/backend/internal/web/dist`
- [ ] B3 后端交叉编译：
      `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed -trimpath
       -ldflags="-s -w -X main.Version=0.2.21-hotfix1 -X main.Commit=<完整40位SHA>
       -X main.Date=<服务器UTC> -X main.BuildType=release" -o sub2api ./cmd/server`
      - **Date 必须取自服务器 `date -u`**（本机时钟可能不准）
- [ ] B4 打包（布局必须匹配 `Dockerfile.goreleaser` 的 COPY）：
      `sub2api` 在 context 根、`backend/resources`、`deploy/docker-entrypoint.sh`、
      `Dockerfile.goreleaser`
- [ ] B5 上传 160 → `docker build -f Dockerfile.goreleaser` + 三个 OCI label
      （revision = **真实完整 SHA**，source = `https://github.com/bingooyong/sub2api`）
- [ ] B6 产物自检：镜像 ID / OCI revision / `-version` 三者一致；架构为 linux/amd64

## 阶段 C：备份与演练（停服前）

- [ ] C1 在线备份：`backup_remote.py --backup-dir /root/sub2api-backups/<RUN>-online`
- [ ] C2 校验并落底座：`verify_backup.py` 通过 → rsync 到本机 `<RUN>/final/`
      （停服窗口内只传增量）
- [ ] C3 隔离演练：复用 `rehearse.sh`（改 IMAGE 与 BACKUP），断言
      - 备份可恢复（`pg_restore --exit-on-error`）
      - 迁移 301 → 306，5 条新迁移逐条在库中
      - 核心表计数与线上一致
      - 镜像能启动并 health ok
- [ ] C4 清理演练残留：容器 / 卷 / 网络

## 阶段 D：切换

- [ ] D1 预检：`deploy_remote.py` 只读模式，`--expected-current-id` / `--expected-target-id`
      / `--expected-revision` 全部核对
- [ ] D2 起本机离站验证器：`nohup auto_offsite_verify.py … &`（不依赖会话存活）
- [ ] D3 远端切换：`setsid nohup deploy_remote.py --apply … &`（会话被杀不影响发布）
- [ ] D4 轮询到 `healthy_needs_verification`，记录 `downtime_seconds`

## 阶段 E：验证与收尾

- [ ] E1 版本接口 / 三个 health / UI 首页与 assets / `/v1/models`
- [ ] E2 数据核对：`backup_remote.py --inventory-only` 双向比对
      - `schema_migrations`：只允许新增 5 条，同名 checksum 逐条未变
      - 业务表：count 与字段哈希逐项一致
      - `containers` 段差异属预期（镜像 ID 与运行状态变了）
- [ ] E3 日志：启动后 error/fatal/panic = 0
- [ ] E4 更新 `docs/Sub2API-GPT-5.6适配记录.md` 与 vutl `AGENTS.md`
      （当前版本、镜像 ID、产物修订、备份路径、manifest SHA、迁移审查结论）
- [ ] E5 记录会话到两侧 journal；必要时更新 vutl `ops` 层 spec
- [ ] E6 磁盘复核：保留当前版 + 上一版镜像与备份

## 验证命令

```bash
# 产物一致性
ssh vutl-vps-160 'docker inspect local/sub2api:0.2.21-hotfix1-2f592f2f8 --format "{{.Id}} {{.Os}}/{{.Architecture}}"'
ssh vutl-vps-160 'docker run --rm --entrypoint /app/sub2api local/sub2api:0.2.21-hotfix1-2f592f2f8 -version'
#   断言：版本串含 0.2.21-hotfix1 与完整 40 位 commit

# 迁移
ssh vutl-vps-160 'docker exec sub2api-postgres psql -U sub2api -d sub2api -Atc \
  "select count(*) from schema_migrations"'
#   断言：306
ssh vutl-vps-160 'docker exec sub2api-postgres psql -U sub2api -d sub2api -Atc \
  "select filename from schema_migrations where filename like '"'"'247%'"'"' or filename like '"'"'248%'"'"' or filename like '"'"'249%'"'"' or filename like '"'"'250%'"'"' order by filename"'
#   断言：247_scoped_admin_api_keys / 248_group_image_count_billing /
#         249_add_usage_logs_api_key_cost_index_notx / 249_mcp_oauth /
#         250_openai_cache_write_ttl 五条齐全

# 接口
curl -fsS -H "x-api-key: $SUB2API_ADMIN_API_KEY" https://sub.fireai.cc.cd/api/v1/admin/system/version
curl -fsS https://sub.fireai.cc.cd/health
curl -fsS https://sub2api.fireai.cc.cd/health
```

## 回滚点

- 旧镜像 `local/sub2api:0.2.18-hotfix1-23efd173b`（ID `sha256:26320bb2ee36…`）保留，
  本次迁移**无不可逆变更** → 回滚只需换回旧镜像并重启，不需要逆向迁移。
- 本轮 final 备份 + manifest SHA 写入 `docs/Sub2API-GPT-5.6适配记录.md` 后，
  才算完成回滚点留存。
