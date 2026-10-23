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
// backend/internal/service/openai_invalid_encrypted_content.go
// 剥 message 项的 encrypted_content（store=false 回放场景下 assistant message
// 也带密文）；保留 id 与可见内容。
func trimOpenAIInvalidEncryptedContentItems(reqBody map[string]any) bool

// 从上游错误文案解析失效 item id（"for item amsg_xxx could not be verified"）。
func openAIInvalidEncryptedContentItemID(msg string) string

// 剥掉 input 中 id 等于失效 item id 的项的 id 字段（切断上游的服务端 lookup）。
func stripOpenAIInvalidEncryptedContentItemID(reqBody map[string]any, itemID string) bool

// 删掉 input 中的 assistant 侧历史（reasoning / compaction / role=assistant）。
func stripOpenAIAssistantHistoryForInvalidEncryptedContent(reqBody map[string]any) bool

// 终极兜底：把 input 裁剪成只保留最后一条 user 消息。
func trimOpenAIInputToLastUserMessage(reqBody map[string]any) bool

// 诊断：概括**出站**请求体（顶层 key + 续链字段 + input 结构摘要）。
func openAIOutboundBodySummary(body []byte) string
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
「密文来自旧账号」的问题原样复制到新账号）。恢复动作有三种，互相独立，任一成功即可重试：

1. **剥 reasoning / compaction 的密文**（`trimOpenAIEncryptedReasoningItems`）：
   - `reasoning` 项：删掉 `encrypted_content` 字段，**保留** `id` / `summary` / 其他字段；
   - `compaction` / `compaction_summary` 项：**整项删除**（这两类必须带密文存在，
     缺密文上游直接 400）；
   - `input` 中的普通消息、工具调用、函数输出：**原样保留**。
2. **剥 message 的密文**（`trimOpenAIInvalidEncryptedContentItems`）：
   - `message` 项删掉 `encrypted_content` 字段，**保留** `id` 与可见内容。
   - **这一步不能省**：OpenAI Responses 在 `store=false` 的回放场景下会给
     **assistant message** 也带上 `encrypted_content`，其失效时上游报的是
     `The encrypted content for item amsg_xxx could not be verified. Reason:
     encrypted content hydration failed: ...`。只剥 reasoning 会让这类错误
     「判定命中但无可剥之物」→ 放弃自愈 → 400 原样回客户端 → 对话卡死。
     （2026-10-23 生产事故的真实形态。）
3. **断开续链**（丢 `previous_response_id`）：仅在既无密文可剥、也无法从错误文案
   定位到失效 item 时使用，且要求请求体不含 `function_call_output`（含工具输出时
   锚点是工具续链依据，不能丢）。
4. **丢弃 assistant 侧历史**：前三级都做不了时，删除 input 中的
   `reasoning` / `compaction` / `compaction_summary` / `role=assistant` 项，保留 user
   消息与工具调用/工具输出。代价是模型看不到之前的 assistant 回答与加密推理；
   收益是请求能成功、对话能继续。**对话卡死（客户端反复重放同一份历史）比丢一段
   历史更糟**，所以这个取舍是划算的。只在确认是密文失效错误时触发。
5. **裁剪为最后一条 user 消息**（终极兜底）：连 assistant 侧历史都识别不出来时
   （客户端回放的 item 不带 `role`），把 input 裁剪成只保留最后一条 user 消息。
   上下文基本丢失，但请求一定发得出去，对话不会卡死。
   > 注意：codex OAuth 路径的 transform 会剥掉 `previous_response_id` 与 item 的
   > `id`，所以这一步在 codex 路径上通常不适用；apikey 路径保留锚点。
   > 剥掉 item 的 `id`（`stripOpenAIInvalidEncryptedContentItemID`）同理：两条路径的
   > transform 都会剥 id，该动作通常不生效，仅作兜底。

5. 记录 lineage（`markOpenAIWSInvalidEncryptedContentLineage`），使**后续 turn** 在进场时
   就被预剥离，不再重复触发上游拒绝。
6. **只重试一次**：`httpInvalidEncryptedContentRetryTried` 标志位保证上限。
   第二次仍失败 → 错误回客户端，**不得形成重试风暴**。

#### 3.4 剥离后的语义边界

被剥掉的是**上游已经拒绝的**密文——它不可能被保留。剥离后：

- 普通对话上下文完整（消息正文、工具调用、函数输出都在）；
- 上一轮的**加密推理/加密消息**内容丢失，模型读不到这部分上下文；
- `previous_response_id` 在「剥密文」路径下**保留**（HTTP 路径下它是 API-key 客户端的
  续链锚点，剥离失效密文不影响它继续有效）。既有测试
  `TestOpenAIGatewayService_Forward_HTTPIngressRetriesInvalidEncryptedContentOnce`
  明确断言 "encrypted-content retry must preserve API-key continuation"。
  > WS 路径丢弃锚点是 WS 特有语义（连接复用 + session 锚点），**不要**照搬到 HTTP 路径。
- 客户端协议形状不变，错误仍在「重试也失败」时照常返回。

### 3.5 诊断必须看「出站 body」，不是客户端原始 body

排查这类问题最容易踩的坑：**看错了对象**。

`ensureReqBody()` 返回的是**客户端原始 body + patch**，而真正发往上游的是 `body`
（经 codex transform 改写后）。两者差异很大：

- codex transform 会剥掉 item 的 `id`；
- codex transform 会剥掉 `reasoning` 项（只保留 `encrypted_content` 通道）；
- codex transform 可能删除 `previous_response_id`。

**用客户端原始 body 判断会得出错误结论**。2026-10-23 的排查中，因为打印的是
decoded 而非出站 body，一度误判为「出站 body 里还有 id 可以剥」，而实际上出站
body 里既没有 id 也没有密文——上游是凭**服务端状态**去 hydrate 密文的。

诊断日志必须打印**出站 body**：

```go
logger.LegacyPrintf("service.openai_gateway",
    "[OpenAI] upstream 400 in forward loop (account: %s, upstream_code: %q, invalid_encrypted: %t, outbound: %s, msg: %s)",
    account.Name, upstreamCode, invalidEncryptedContentError,
    openAIOutboundBodySummary(body), truncateString(upstreamMsg, 200))
```

`openAIOutboundBodySummary` 只输出：顶层 key 列表（不含 `instructions`）、
续链字段（`previous_response_id` / `conversation` / `include` / `store` /
`reasoning` / `model`）与 input 的结构摘要（每个 item 的 type/role/id/有无密文）。
**绝不输出用户对话内容与密文本身。**

### 4. Validation & Error Matrix

| 条件 | 必须的结果 |
| --- | --- |
| 上游 400，`code=null`，文案含 `encrypted content` + `could not be verified` | 命中 → 剥离密文 → 重试一次 |
| 上游 400，`code=invalid_encrypted_content` | 命中（既有行为，不回归） |
| 上游 400，`code=thinking_signature_invalid` + 密文失效文案 | 命中（既有行为，不回归） |
| 上游 400，文案为 `Invalid signature` | **不**命中，不重试 |
| 上游 400，普通错误（如 `bad request`） | **不**命中，不重试 |
| 命中且 `input` 里有 `reasoning` 项带 `encrypted_content` | 删该字段，保留 `id` / `summary` |
| 命中且 `input` 里有 `message` 项带 `encrypted_content` | 删该字段，保留 `id` / 正文（**最常见的真实形态**） |
| 命中且 `input` 里有 `compaction` 项 | 整项删除（带密文的 compaction 不能半剥） |
| 命中但 body 里**完全没有**密文（密文在上游服务端） | 剥失效 item 的 id → 丢锚点 → 删 assistant 侧历史 → 裁剪为最后一条 user 消息，依次尝试 |
| 命中且连 assistant 侧历史都识别不出来（item 不带 role） | 裁剪为最后一条 user 消息后重试 |
| 命中但五种动作都做不了 | 记 `Skip ... because encrypted reasoning items are missing`，**不**重试 |
| 重试后再次 400 | 终止，回客户端，不第三次重试 |
| 命中且走了「剥密文」路径 | **保留** `previous_response_id`（HTTP 续链契约） |
| 命中且走了「丢锚点」路径 | 要求请求体不含 `function_call_output` |
| 命中且含 `function_call_output` | 保留工具输出与锚点 |

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
  - **message 密文形态**（`amsg_` 前缀 + `hydration failed`）→ 上游被调用 **2 次**；
    第二次请求体中该 message 的 `encrypted_content` 被删，`id` 与正文保留
  - 无关 400 → 上游只被调用 **1 次**
  - 连续两次 400 → 上游只被调用 **2 次**（不风暴）
  - apikey 路径上无密文可剥时 → 丢 `previous_response_id` 后重试
  - 含 `function_call_output` → 锚点保留
  - body 里没有任何 `encrypted_content`（服务端密文形态）→ 仍能通过「剥 id / 丢锚点 /
    删 assistant 侧历史 / 裁剪为最后一条 user 消息」中的某一级恢复
  - item 不带 `role` → 终极兜底裁剪为最后一条 user 消息后重试成功
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

Wrong：只剥 reasoning 的密文

```go
// message 项的 encrypted_content 也会让上游 hydration 失败
if trimOpenAIEncryptedReasoningItems(decoded) { ... }   // 对 message 密文返回 false
// → 判定命中但无可剥之物 → 放弃自愈 → 400 回客户端 → 对话卡死
```

Correct：reasoning 与 message 两类密文都要剥

```go
trimmed := trimOpenAIEncryptedReasoningItems(decoded)
trimmed = trimOpenAIInvalidEncryptedContentItems(decoded) || trimmed
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
// HTTP 路径：剥密文时锚点保留
body, err = marshalOpenAIUpstreamJSON(decoded) // decoded 只被剥了失效密文
```

### 8. 不做的事

- 不做「同一会话固定同一账号」的调度改造：那是账号策略，收益大但影响面广，另立任务。
  密文由上游签发，网关的职责是**识别并自愈**，不是保证密文永远可解。
- 不改 `store=false` 下 `encrypted_content` 承载推理上下文的既有契约
  （`openai_codex_transform.go` 的 `filterCodexInputWithOptions`）。
- 不在命中时切账号：400 是确定性请求错误，切账号不能修复密文。
