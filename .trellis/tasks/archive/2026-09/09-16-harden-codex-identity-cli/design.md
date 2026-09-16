# Technical design

先将“上游可见身份”与“内部命名空间”分离：内部 hash domain 保持不变，避免旋转稳定 ID；正文 marker/tool alias 改为兼容开关或无品牌等价值。CLI identity 由结构化配置（版本、UA 前缀/后缀策略）统一解析，所有 HTTP/WS 出站边界复用同一快照。WS 每个 turn 在构造 payload 与 handshake headers 前完成 fingerprint staging，保证 body/header 一致。Chat Completions 兼容分支显式启用 PreserveToolCallIDs。

回滚以逐文件 revert 为边界；先保留旧 marker 的兼容读取/响应映射，再切换默认发送行为。通过 mock upstream 捕获最终 headers/body，验证无明文泄漏和身份一致性。

## Review refinements

- No blanket replacement of user content: the no-brand guarantee covers gateway-generated protocol markers, not arbitrary prompts or user-declared tool names. Legacy exact gateway marker pairs require idempotent migration; tool alias migration must distinguish declared literal tools from historical aliases and retain collision checks.
- Reuse the existing admin UA/version and version-sync configuration. If adding a runtime platform option, make it explicit and backwards-compatible; never mistake the gateway host for the caller machine. Do not randomly rotate identity or bump versions without source evidence.
- Native WS must preserve per-account/per-turn isolation, avoid applying convergence twice on HTTP-to-WS paths, and handle connection reuse when a handshake identity changes. Validate the alleged gaps in code before fixing.
- Keep existing hashes and caller identity opt-out intact. No added dependencies. Publish findings and application-layer limitations in changelog/audit; no assertion of TLS equivalence.
