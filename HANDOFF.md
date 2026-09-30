# Handoff Index

## Current Handoff

- 最新交接文档：[账号质量运维](docs/handoffs/sub2api-2026-09-30-account-quality-operations-handoff.md)。
- 当前分支：`v0.2.10-hotfix1`；HEAD：`4996ae1ed`。
- 账号质量运维已实现、验证并完成 Trellis 归档；未推送、打标签、发布或部署。
- `.omc/` 仍为未跟踪本地运行状态，不属于交接提交。

## History

| Date | Scope | Topic | Status | Document | Commit/Ref | Next Action |
| --- | --- | --- | --- | --- | --- | --- |
| 2026-09-30 | sub2api | 账号质量运维与智能测试 | 本地实现与验证完成；集成容器测试受 Docker/Colima 阻塞 | [账号质量运维](docs/handoffs/sub2api-2026-09-30-account-quality-operations-handoff.md) | `4996ae1ed` / `v0.2.10-hotfix1` | Docker/Colima 可用后重跑 integration tests；明确授权后再推送或发布 |
| 2026-09-22 | sub2api | 合并上游 v0.2.7 并构建 hotfix1 | 本地构建与验证完成 | [前期背景](docs/handoffs/sub2api-2026-09-16-codex-hotfix1-handoff.md) | `dev` / `d329d452b` | 按明确授权再发布或部署 |
| 2026-09-16 | sub2api | Codex 指纹修复及 hotfix1 打包 | 历史快照；已由 2026-09-22 收尾取代 | [交接文档](docs/handoffs/sub2api-2026-09-16-codex-hotfix1-handoff.md) | `dev` / `f2b15cdfc` | 参见上方当前状态 |
