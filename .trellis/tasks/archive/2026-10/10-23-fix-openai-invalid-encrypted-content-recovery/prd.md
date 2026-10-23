# 修复 OpenAI 加密推理密文失效（invalid encrypted content）的 HTTP 路径自愈判定

## Goal

上游（OpenAI 官方或中转）判定客户端回放的加密推理密文失效时，HTTP（非 WSv2）转发路径
必须能够识别该错误、剥掉失效密文并重试一次，而不是把 400 原样回给客户端导致整段对话卡死。

## 背景（生产取证）

- 线上现象：160 的 sub2api 转发 `/v1/responses`（模型 `gpt-6.1-sol`、账号 6642
  `agentrouter-bingoo-github`）时，上游返回 400，响应体形如：

  ```json
  {"error":{"message":"The encrypted content gAAA...3WpB could not be verified. Reason: Encrypted content could not be decrypted or parsed. (request id: ...) [trace_id: ...]","type":"invalid_request_error","param":"","code":null}}
  ```

  客户端（Codex）重放同一份历史 → 无限失败，错误原样出现在用户界面。
- 发生频次：`ops_system_logs` 显示 2026-09-15 起零星出现，2026-10-23 06:00–07:00 UTC
  集中爆发（8 + 4 条）。**与 2026-10-23 的 0.2.17-hotfix1 升级无关**，是长期存在的间歇性缺陷。
- 网关已有自愈逻辑但判定落空：
  - `backend/internal/service/openai_gateway_forward.go:1113` 判定条件为
    `upstreamCode == "invalid_encrypted_content"`，或
    `upstreamCode == "thinking_signature_invalid"` + 固定文案组合。
  - `extractUpstreamErrorCode`（`gateway_upstream_response.go:316`）在 `error.code` 为
    `null` 时返回空串 → 两个分支都不成立 → 不重试。
  - 日志中 30 分钟内没有任何一条
    `[OpenAI] Retrying non-WSv2 request once after invalid_encrypted_content`，佐证未触发。
- 同仓库 WS 路径判定正确：`openai_ws_forwarder_support.go:761` 用文案兜底
  （`encrypted content` + `could not be verified`），HTTP 路径与 WS 路径口径不一致。

## Requirements

- R1 HTTP 路径的失效密文判定不得依赖 `error.code`；上游以 `code=null`、
  `type=invalid_request_error` 形态返回时同样必须命中。
- R2 判定口径必须与 WS 路径一致，两条路径共用同一个判定函数，避免再次漂移。
- R3 命中后复用既有的 `trimOpenAIEncryptedReasoningItems` 剥除失效密文，**只重试一次**，
  不得变成重试风暴。
- R4 重试时**必须保留** `previous_response_id`。HTTP 路径下它是 API-key 客户端的续链
  锚点，剥离失效密文不影响它继续有效（既有契约，见
  `openai_ws_protocol_forward_test.go` 的 "encrypted-content retry must preserve API-key
  continuation" 断言）。WS 路径丢弃锚点是 WS 特有语义（连接复用 + session 锚点），
  **不适用于 HTTP 入站**，本任务不得改动。
- R5 不得误伤无关错误：与密文无关的 400（例如 `Invalid signature`）不得触发重试。
- R6 不得降低既有安全约束：剥离动作只影响 `reasoning.encrypted_content` 与
  `compaction/compaction_summary` 整项，`input` 里的普通消息、工具调用、函数输出保持不变。
- R7 回归测试必须覆盖本次生产错误体的真实形态（`code` 为 `null`）。

## Constraints

- 只改 sub2api 仓库源码，**本任务不发版到 160**（发版由后续独立任务执行，
  流程见 vutl 仓库 `.trellis/spec/ops/sub2api-160-deploy.md`）。
- 改动不涉及数据库迁移，不涉及前端，不改变请求/响应的对外协议形状。
- 保持 `store=false` 场景下 `encrypted_content` 承载推理上下文的既有行为不变
  （见 `openai_codex_transform.go` 中 `filterCodexInputWithOptions` 的注释契约）。

## Acceptance Criteria

- [ ] 上游返回 `{"error":{"message":"The encrypted content ... could not be verified. Reason: Encrypted content could not be decrypted or parsed.","type":"invalid_request_error","param":"","code":null}}`
      时，HTTP 路径剥离失效密文并重试一次，客户端拿到成功响应。
- [ ] 重试请求体中不再含 `encrypted_content`，且 `input` 中的普通消息与工具调用原样保留。
- [ ] 重试请求体保留 `previous_response_id`（不回归既有 HTTP 续链契约）。
- [ ] `error.code` 为 `invalid_encrypted_content` 的既有形态仍然命中（不回归）。
- [ ] `error.code` 为 `thinking_signature_invalid` 的既有形态仍然命中（不回归）。
- [ ] 文案为 `Invalid signature` 等无关错误时**不**触发重试。
- [ ] `go test ./internal/service/...` 全绿；新增测试显式断言"只重试一次"。
- [ ] 新增/修改的判定函数被 HTTP 路径与 WS 路径共用，并有单测锁定两者行为一致。
- [ ] 该契约写入 `.trellis/spec/backend/`（code-spec 深度，7 节）。

## Notes

- 触发源在客户端侧不可控：密文由上游签发，上游/中转若在会话中途换用另一把密钥或另一个
  账号，旧密文必然解不开。网关的职责是**识别并自愈**，而不是保证密文永远可解。
- 若上游账号/中转长期不稳定，根治方向是"同一会话固定同一账号"，属于账号调度策略，
  不在本任务范围内。
