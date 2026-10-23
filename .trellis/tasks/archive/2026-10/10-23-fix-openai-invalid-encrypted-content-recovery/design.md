# 设计：HTTP 路径加密推理密文失效自愈

## 1. 边界

改动只发生在 `backend/internal/service` 内，三处：

| 位置 | 现状 | 目标 |
| --- | --- | --- |
| `openai_gateway_forward.go:1113`（HTTP 非 WSv2 路径） | 判定绑死在 `error.code` 上 | 判定与上游错误体形态解耦 |
| `openai_ws_forwarder_support.go:761`（WS 路径） | 文案兜底判定，正确 | 与 HTTP 路径共用同一判定函数 |
| 新增 `openai_invalid_encrypted_content.go` | — | 单一判定真源 + HTTP 恢复动作补齐 |

不涉及：数据库、前端、对外协议形状、账号调度策略。

## 2. 判定函数

```go
// isOpenAIInvalidEncryptedContentUpstreamError(code, msg string) bool
```

判定输入是「上游 `error.code` + `error.message`」，输出是「是否属于加密推理密文失效」。

```
code == "invalid_encrypted_content"                       → true
msg 含 "invalid_encrypted_content"                         → true
msg 含 "encrypted content" 且含 "could not be verified"    → true
msg 含 "encrypted content" 且含 "could not be decrypted"   → true
其他                                                       → false
```

### 为什么不能只认 code

生产错误体（2026-10-23 取证自 160）：

```json
{"error":{"message":"The encrypted content gAAA...3WpB could not be verified. Reason: Encrypted content could not be decrypted or parsed. (request id: ...) [trace_id: ...]","type":"invalid_request_error","param":"","code":null}}
```

`error.code` 是 `null` → `extractUpstreamErrorCode`（`gateway_upstream_response.go:316`）返回空串 →
`code == "invalid_encrypted_content"` 与 `code == "thinking_signature_invalid"` 两个分支都不成立 →
自愈不触发 → 400 原样回客户端。

### 为什么误伤风险可控

- 判定要求文案同时出现「密文对象」与「不可验证/不可解密」两个语义要素；
  无关的签名类错误（如 `Invalid signature`）不含 `encrypted content`，不命中。
- 命中最坏后果是**多发一次重试**（剥掉失效密文后重放），不会放大成重试风暴：
  `httpInvalidEncryptedContentRetryTried` 保证只重试一次。

### 为什么共用一个函数

当前两条路径各写一套判定，HTTP 这套更严 → 同一个上游错误在 WS 下能自愈、在 HTTP 下不能，
这种「同语义不同判定」的漂移是本次缺陷的直接成因。共用函数后，新增错误形态只需改一处。

## 3. HTTP 恢复动作

命中后在**同一账号**上重试一次（不切账号：400 是确定性请求错误，换账号只会把
「密文来自旧账号」的问题原样复制到新账号）：

1. `trimOpenAIEncryptedReasoningItems(decoded)`
   —— 删掉 `reasoning` 项的 `encrypted_content` 字段、整项删掉 `compaction` /
   `compaction_summary`。普通消息、工具调用、函数输出不动。
2. `previous_response_id` **保持原样，不动**。
   HTTP 路径下它是 API-key 客户端的续链锚点，剥离失效密文不影响它继续有效；
   既有测试 `TestOpenAIGatewayService_Forward_HTTPIngressRetriesInvalidEncryptedContentOnce`
   明确断言 "encrypted-content retry must preserve API-key continuation"。
   WS 路径丢弃锚点（`openai_gateway_forward.go` 的 WS 恢复分支）是 WS 特有语义
   （连接复用 + session 锚点），不适用于 HTTP 入站。
   > 实施过程中的教训：最初按 WS 语义在 HTTP 路径也丢弃锚点，直接打挂上述既有测试。
   > 两条路径的**判定**必须同源，但**恢复动作**可以不同。
3. 记录 lineage（`markOpenAIWSInvalidEncryptedContentLineage`），使**后续 turn** 在
   进场时就被预剥离，不再重复触发上游拒绝。这是既有行为，保持不变。

## 4. 数据流

```
客户端 /v1/responses
  └─ Forward (HTTP, 非 WSv2)
       └─ 上游 400
            ├─ 判定 isOpenAIInvalidEncryptedContentUpstreamError(code, msg)
            │    ├─ false → handleErrorResponse → 400 回客户端（现状行为不变）
            │    └─ true  → trim 失效密文 (+ 丢 previous_response_id) → 重试一次
            │                 ├─ 成功 → 正常返回，客户端无感
            │                 └─ 再次 400 → 不再重试 → 回客户端（终止，避免风暴）
            └─ lineage 记录 → 下一 turn 进场预剥离
```

## 5. 兼容性与风险

- 对客户端协议无变化：错误仍在「上游确实不可用且重试也失败」时返回。
- 对正常路径无影响：判定只在 400 且文案命中时生效。
- 唯一行为变化：`code=null` 形态的密文失效错误现在会被自愈（这是修复目标）。
  剥离后推理上下文丢失，模型无法读到上一轮的加密推理——这是上游已经拒绝的密文，
  不可能保留；普通对话上下文完整。
- WS 路径的判定会被轻微放宽（新增 `could not be decrypted` 短语），语义同源；
  WS 分类仅影响「是否触发重连恢复」，不改变错误码映射。

## 6. 测试策略

| 层 | 用例 | 断言 |
| --- | --- | --- |
| 单元 | 判定函数表驱动 | `code=null` + 生产文案 → true；两个历史 code → true；`Invalid signature` → false；普通 400 → false |
| 集成 | HTTP 路径（`code=null` 生产形态） | 上游被调用 2 次；第二次请求体无 `encrypted_content`；普通消息/工具输出仍在 |
| 集成 | `previous_response_id` | 重试体**保留**该字段（HTTP 续链契约不回归） |
| 集成 | 无关 400 | 上游只被调用 1 次（不重试） |
| 集成 | WS 分类 | `classifyOpenAIWSErrorEventFromRaw` 对 `code=null` 生产文案返回 `invalid_encrypted_content` |
| 回归 | 既有测试 | `go test ./internal/service/...` 全绿 |

## 7. 不做的事

- 不做「同一会话固定同一账号」的调度改造：那是账号策略，收益大但影响面广，另立任务。
- 不改 `store=false` 下 `encrypted_content` 承载推理上下文的既有契约
  （`openai_codex_transform.go` 的 `filterCodexInputWithOptions`）。
- 不发版到 160：发版属独立任务，按 vutl 仓库 `.trellis/spec/ops/sub2api-160-deploy.md` 执行。
