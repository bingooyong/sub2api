# Changelog

## v0.2.5-hotfix1 - 2026-09-16

本版本合并上游 `v0.2.5`，并保留 Codex 客户端身份与指纹兼容修改。

### 后端稳定性与兼容性

- 增强 OpenAI Responses / Messages 转换，修复 Codex Lite namespace、tool-call ID 和 sequence number 兼容问题。
- 修复 Gemini、Antigravity、Ollama Cloud、Grok 等上游兼容问题。
- 优化 OpenAI WebSocket 连接池的连接重选、排队和容量计算。
- 修复 Accept-Encoding、超时、错误流和上下文窗口相关问题。

### 账号与管理

- 支持订阅批量操作和 API Key 批量编辑。
- 支持显式清除代理凭据。
- 修复账号刷新、限额、退款和注册密码确认问题。
- 增强 OpenAI token 统计和调度指标。

### 前端

- 新增订阅批量操作、Key 批量编辑和代理筛选/凭据管理界面。
- 改进注册、余额提醒、Usage 和 Channel 状态刷新。
- 补充中英文 locale 与回归测试。

### 配置与部署

- compact 默认模型升级为 GPT-5.5。
- WebSocket 连接池 OAuth/API Key 默认系数调整为 5.0。
- 更新网关和部署配置示例。

### Codex 身份与指纹

- 保留 Codex `User-Agent`、`Originator`、`Version` 透传及账号隔离策略。
- 保留 Codex 安装、会话、线程和窗口指纹处理；不修改既有哈希命名空间。
- Codex CLI 模拟支持 Responses、Responses Lite、SSE、tool-call correlation 和 `client_metadata`。
- 默认 Codex 身份仍为应用层兼容，不代表 TLS、HTTP/2 或官方 OAuth 设备认证完全复制。
- 图片桥接、Spark 限制提示和 todo guard 改用无品牌标记，Python 工具别名改为 `python__codex`；保留旧标记的幂等检测和原有哈希命名空间。
- 修复 WebSocket v2 直通首帧及后续帧未应用已暂存指纹的问题，使配置的收敛身份与握手一致。
- WebSocket 按帧独立识别默认 `prompt_cache_key`，避免后续帧复用首帧 session 判断；保留自定义缓存键的账号隔离语义。
- 补充真实 WebSocket 入口的多帧回归测试，覆盖 off/device/session/full 模式和普通 API Key 账号。
