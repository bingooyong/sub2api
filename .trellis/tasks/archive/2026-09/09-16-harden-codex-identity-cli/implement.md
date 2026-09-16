# Implementation plan

1. 建立 marker/tool alias/identity 的现状测试与 loopback 捕获基线。
2. 实现无品牌兼容值或显式兼容开关，更新响应反向映射和配置文档。
3. 将 Codex fingerprint staging 接入 native WS ingress/v2，并补 body/header 一致性测试。
4. 修复 Chat Completions Responses-shaped 分支的 PreserveToolCallIDs。
5. 增强 CLI identity 配置解析（版本与平台后缀）及统一请求构造。
6. 更新 CHANGELOG.md，运行 Go 测试、vet/gofmt/diff-check 和可用的前端检查。
7. 审查最终 diff，确认不改 hash domain、账号隔离和现有指纹策略。
