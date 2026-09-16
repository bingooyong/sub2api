# Technical design

采用“上游基线 + 本地提交重放/合并”的方式：先建立 v0.2.5 与 dev 的提交和文件差异矩阵，再按文件组处理。指纹保护范围包括 `backend/internal/service/openai_codex_*`、相关 gateway forwarding/identity 文件，以及 `docs/audits/*fingerprint*` 和兼容性审计文档。冲突优先保留本地指纹语义，再吸收上游独立修复；若同一函数同时修改，则合并逻辑并补测。

验证分层：先检查冲突和关键 diff，再运行后端 Go 测试与 lint、前端测试/typecheck（依据仓库脚本），最后 `git diff --check`、状态检查和提交审查。回滚点为合并前分支 tip。
