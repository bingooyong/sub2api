# OpenAI 加密推理密文失效的识别与自愈

> 契约来源：2026-10-23 生产事故。上游判定客户端回放的加密推理密文
> （`reasoning.encrypted_content`）失效时，网关必须识别并自愈，而不是把 400 原样回给
> 客户端导致整段对话卡死。

---

## Scenario: 识别上游的密文失效错误并重放一次

### 1. Scope / Trigger

读这份契约当你需要：

- 修改 OpenAI 转发路径的**错误分类**或**重试逻辑**；
- 新增/调整「上游 4xx 是否值得重试」的判断；
- 排查「上游报 400 但客户端反复重放同一份历史」的对话卡死问题；
- 在 HTTP 入站路径与 WSv2 路径之间复制或对齐错误处理。

触发即适用，因为这个错误有三个易错点：**错误体形态不固定**、**两条转发路径口径易漂移**、
**重试动作的副作用边界容易被忽视**。

### 2. Signatures

```go
// backend/internal/service/openai_invalid_encrypted_content.go
// 判定上游错误是否表示「加密推理密文已失效」。HTTP 与 WS 两条路径共用。
func isOpenAIInvalidEncryptedContentUpstreamError(code, msg string) bool
```

```go
// backend/internal/service/openai_gateway_request_body.go
// 剥离请求体中的失效密文；返回是否发生了修改。
func trimOpenAIEncryptedReasoningItems(reqBody map[string]any) bool
```

```go
// backend/internal/service/openai_ws_forwarder_support.go
// WS 错误事件分类，产出稳定的 reason 字符串。
func classifyOpenAIWSErrorEventFromRaw(codeRaw, errTypeRaw, msgRaw string) (string, bool)
```

```go
// backend/internal/service/gateway_upstream_response.go
// 提取上游 error.code；error.code 为 null 时返回空串。
func extractUpstreamErrorCode(body []byte) string
```

上游错误体的实际形态（生产取证，2026-10-23）：

```json
{"error":{"message":"The encrypted content gAAA...3WpB could not be verified. Reason: Encrypted content could not be decrypted or parsed. (request id: ...) [trace_id: ...]","type":"invalid_request_error","param":"","code":null}}
```

其他已知形态：

```json
{"error":{"code":"invalid_encrypted_content","type":"invalid_request_error","message":"bad encrypted content"}}
{"error":{"code":"thinking_signature_invalid","message":"The encrypted content abc could not be verified. Reason: ..."}}
```

### 3. Contracts

#### 3.1 判定不得依赖 `error.code`

`error.code` 在生产里**经常是 `null`**，语义只体现在 `error.message` 文案里。
`extractUpstreamErrorCode` 遇 `null` 返回空串，因此：

> 任何只认 `error.code` 的判定都会漏判最常见的形态。

判定必须同时接受「错误码」与「文案」两个来源：

- `code == "invalid_encrypted_content"` → 命中
- 文案含 `invalid_encrypted_content` → 命中
- 文案含 `encrypted content` **且**含 `could not be verified` 或 `could not be decrypted` → 命中

**必须同时出现「密文对象」与「不可验证/不可解密」两个语义要素**，否则会把无关的签名类
错误（`Invalid signature`）误判为密文失效。

#### 3.2 两条路径共用同一判定

HTTP 入站路径与 WSv2 路径的**判定**必须同源。历史上两条路径各写一套：WS 用文案兜底
（正确），HTTP 绑死在 `error.code` 上（漏判）——同一个上游错误在 WS 下能自愈、在 HTTP 下
不能，这种「同语义不同判定」的漂移就是本缺陷的直接成因。

> 判定同源 ≠ 恢复动作相同。两条路径的恢复动作**允许不同**（见 3.3）。

#### 3.3 HTTP 路径的恢复动作

命中后在**同一账号**上重试**一次**（不切账号：400 是确定性请求错误，换账号只会把
「密文来自旧账号」的问题原样复制到新账号）：

1. `trimOpenAIEncryptedReasoningItems` 剥离失效密文：
   - `reasoning` 项：删掉 `encrypted_content` 字段，**保留** `id` / `summary` / 其他字段；
   - `compaction` / `compaction_summary` 项：**整项删除**（这两类必须带密文存在，
     缺密文上游直接 400）；
   - `input` 中的普通消息、工具调用、函数输出：**原样保留**。
2. `previous_response_id`：**保持不动**。HTTP 路径下它是 API-key 客户端的续链锚点，
   剥离失效密文不影响它继续有效。既有测试
   `TestOpenAIGatewayService_Forward_HTTPIngressRetriesInvalidEncryptedContentOnce`
   明确断言 "encrypted-content retry must preserve API-key continuation"。
   > WS 路径丢弃锚点是 WS 特有语义（连接复用 + session 锚点），**不要**照搬到 HTTP 路径。
3. 记录 lineage（`markOpenAIWSInvalidEncryptedContentLineage`），使**后续 turn** 在进场时
   就被预剥离，不再重复触发上游拒绝。
4. **只重试一次**：`httpInvalidEncryptedContentRetryTried` 标志位保证上限。
   第二次仍失败 → 错误回客户端，**不得形成重试风暴**。

#### 3.4 剥离后的语义边界

被剥掉的是**上游已经拒绝的**密文——它不可能被保留。剥离后：

- 普通对话上下文完整（消息、工具调用、函数输出都在）；
- 上一轮的**加密推理**内容丢失，模型读不到这部分上下文；
- 客户端协议形状不变，错误仍在「重试也失败」时照常返回。

### 4. Validation & Error Matrix

| 条件 | 必须的结果 |
| --- | --- |
| 上游 400，`code=null`，文案含 `encrypted content` + `could not be verified` | 命中 → 剥离密文 → 重试一次 |
| 上游 400，`code=invalid_encrypted_content` | 命中（既有行为，不回归） |
| 上游 400，`code=thinking_signature_invalid` + 密文失效文案 | 命中（既有行为，不回归） |
| 上游 400，文案为 `Invalid signature` | **不**命中，不重试 |
| 上游 400，普通错误（如 `bad request`） | **不**命中，不重试 |
| 命中但请求体里没有任何加密推理项 | 记 `Skip ... because encrypted reasoning items are missing`，**不**重试 |
| 重试后再次 400 | 终止，回客户端，不第三次重试 |
| 命中且 `previous_response_id` 存在 | **保留**该字段（HTTP 续链契约） |
| 命中且含 `function_call_output` | 保留工具输出与锚点 |
| 命中且含 `compaction` 项 | 整项删除（带密文的 compaction 不能半剥） |

### 5. Good/Base/Bad Cases

- **Good**：客户端回放的 `encrypted_content` 来自上游已更换的密钥/账号。网关剥掉它、
  保留对话上下文、重试一次成功——客户端完全无感，用户只看到一次轻微延迟。
- **Base**：上游首次就返回 `invalid_encrypted_content` 且请求体里确实没有加密推理项
  （例如客户端手工构造）。此时不重试，直接把错误回客户端——没有可剥的东西。
- **Bad**：只认 `error.code`。生产形态 `code=null` 直接漏判，400 原样回客户端，
  客户端重放同一份历史 → 无限失败 → 对话卡死。2026-09-15 起零星出现，
  2026-10-23 集中爆发。

### 6. Tests Required

- **判定函数表驱动**（`backend/internal/service/openai_invalid_encrypted_content_test.go`）：
  - `code=null` + 生产文案 → 命中
  - `code=invalid_encrypted_content` → 命中
  - `code=thinking_signature_invalid` + 密文失效文案 → 命中
  - `Invalid signature` → 不命中
  - 普通 `bad request` → 不命中
  - 空输入 → 不命中
- **HTTP 集成**：
  - 生产形态 `code=null` → 上游被调用 **2 次**；第二次请求体无 `encrypted_content`；
    摘要、普通消息、工具输出原样保留；`previous_response_id` 保留
  - 无关 400 → 上游只被调用 **1 次**
  - 连续两次 400 → 上游只被调用 **2 次**（不风暴）
- **路径一致性**：`classifyOpenAIWSErrorEventFromRaw` 对生产形态（`code` 为空/ `null`）
  返回 `invalid_encrypted_content`，与 HTTP 判定结论一致。
- **回归**：`go test ./internal/service/ -count=1` 全绿，尤其
  `TestOpenAIGatewayService_Forward_HTTPIngressRetriesInvalidEncryptedContentOnce`
  与 `..._HTTPIngressRetriesWrappedInvalidEncryptedContentOnce`。

### 7. Wrong vs Correct

Wrong：判定绑死在错误码上

```go
invalid := upstreamCode == "invalid_encrypted_content" ||
    (upstreamCode == "thinking_signature_invalid" &&
        strings.Contains(upstreamMsg, "could not be verified"))
// 上游返回 code=null 时 extractUpstreamErrorCode 返回空串 → 漏判 → 400 回客户端
```

Correct：判定与错误码解耦，码与文案双通道，且两条路径共用

```go
invalid := isOpenAIInvalidEncryptedContentUpstreamError(upstreamCode, upstreamMsg)
```

Wrong：把 WS 的恢复动作照搬到 HTTP 路径

```go
// WS 路径的语义：丢锚点重放
if previousResponseID != "" && !hasFunctionCallOutput {
    delete(wsReqBody, "previous_response_id")
}
// 照搬到 HTTP 会打挂 "must preserve API-key continuation" 既有契约
```

Correct：判定同源，恢复动作各按各的路径语义

```go
// HTTP 路径：锚点保留
body, err = marshalOpenAIUpstreamJSON(decoded) // decoded 只被剥了失效密文
```

### 8. 不做的事

- 不做「同一会话固定同一账号」的调度改造：那是账号策略，收益大但影响面广，另立任务。
  密文由上游签发，网关的职责是**识别并自愈**，不是保证密文永远可解。
- 不改 `store=false` 下 `encrypted_content` 承载推理上下文的既有契约
  （`openai_codex_transform.go` 的 `filterCodexInputWithOptions`）。
- 不在命中时切账号：400 是确定性请求错误，切账号不能修复密文。
