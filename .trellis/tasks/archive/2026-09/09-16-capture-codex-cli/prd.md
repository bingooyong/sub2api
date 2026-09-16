# Capture Codex CLI wire behavior

## Goal
通过脱敏本地 loopback 方式核对 Codex CLI 应用层请求，避免凭记忆推断 UA、请求体和事件行为。

## Scope
仅记录白名单头、字段名、事件类型和退出结果；不保存 token、Cookie、prompt、真实会话 ID 或安装 ID。TLS/HTTP2/OAuth 真实端点不在本次自动采样范围。

## Acceptance criteria
- 运行隔离 CLI 采样或记录明确的连接失败原因。
- 输出脱敏 JSON 及限制说明。
- 不将失败采样误报为成功，不改动产品代码。
