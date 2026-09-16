# Harden Codex identity and CLI simulation

## Goal
降低 Codex 请求中可被上游直接识别为 Sub2API 的明文痕迹，并增强可配置的 Codex CLI 应用层模拟，同时保持现有协议兼容和指纹隔离语义。

## In scope
- 评估并替换/门控图片桥接、Spark 提示、Messages todo guard、`python__sub2api` 等上游可见标记；保留必要兼容能力。
- 为 HTTP、passthrough、WebSocket 路径统一 CLI 身份和指纹策略，修复已发现的 WS convergence 缺口。
- 修复 Responses-shaped Chat Completions 的 tool-call ID 保留选项。
- 增加回归测试、配置说明、本地 mock/loopback 验证和 changelog 条目。

## Out of scope
- TLS ClientHello、HTTP/2 栈或官方 OAuth/设备认证的完全复制。
- 生产部署、远端推送和破坏性数据迁移。

## Acceptance criteria
1. 默认 Codex 请求不包含 `sub2api` 明文；兼容标记仅在明确启用且有测试覆盖时出现，或由等价无品牌机制替代。
2. Python 工具名称在上游协议中保持可用且不暴露 `sub2api`，响应映射完整。
3. HTTP 与 WebSocket 的指纹收敛模式行为一致，session/thread/device/full 均有回归测试。
4. Chat Completions Responses-shaped 请求不再碰撞不同 tool-call ID。
5. CLI 模拟可通过配置选择身份版本/平台后缀，生成一致的 UA、originator、version 和请求元数据；loopback 验证通过。
6. 后端测试、格式检查和 diff 检查通过，并更新 changelog。
