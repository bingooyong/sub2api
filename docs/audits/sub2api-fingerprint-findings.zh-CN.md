# Sub2API 请求指纹与硬编码核查清单

本文列出请求身份、提示词、工具名、调用 ID、会话标识和传输层中值得关注的位置，附文件、源码摘录、触发条件及当前状态。

核查日期：2026-09-12。修复前基线：`4726bdd08`；已修复源码：`06bb4df4c`；当前分支：`dev`。文中“已修复”指本地源码提交，不表示已合入上游或部署上线。源码摘录注明所属提交；文件链接及行号对应 `06bb4df4c`，后续可按函数名定位。

“存在硬编码”“与某个 CLI 样本不同”“已确认错误”是不同结论。本文按通用网关的元数据保留和协议关联要求判断问题，不根据某个字符串推断封禁原因。代码注释中关于上游降载、404 或客户端识别的说法，也不等于本次已重新验证的服务端规则。

## 1. 先看结论与定位索引

| 编号 | 核查项 | 主要文件 | 判断与当前状态 |
| --- | --- | --- | --- |
| A1 | 固定 OS、架构、终端和客户端名称 | `openai_gateway_service.go`、`request.go` | 默认身份会偏离实际调用方；固定兜底仍保留 |
| A2 | 强制改写 UA、originator、version | `openai_codex_identity.go` | 默认开启的身份策略，独立于可选指纹收敛 |
| A3 | 版本下限与版本重建 | `openai_codex_identity.go`、`setting_gateway_runtime.go` | 可能改变客户端版本声明；统一身份模式仍保留此策略 |
| A4 | 关闭强制改写后仍补写或修改身份头 | 各 HTTP、Messages、搜索和 WS 构造器 | **已确认兼容性缺陷，已修复** |
| A5 | WS 复用未区分客户端握手身份 | `openai_ws_pool.go` | **已确认兼容性缺陷，已修复** |
| A6 | 原生 tool call ID 规范化造成碰撞 | `openai_gateway_forward.go`、`openai_codex_transform.go` | **已确认协议关联缺陷，已修复** |
| A7 | 跨轮引用忽略 ID 保留选项 | `openai_codex_transform.go` | **已确认兼容性缺陷，已修复** |
| B1 | 缺少 instructions 时填充默认提示 | `openai_gateway_forward.go`、`openai_codex_transform.go` | 正文被修改；内置提示文件没有 `sub2api` 文本 |
| B2 | 强制 instructions 模板 | `openai_codex_instructions_template.go` | Messages→Codex 桥接中显式配置后可覆盖原提示，仍保留 |
| B3 | 图片桥接标记 | `openai_codex_transform.go` | **上游可见明文**，有条件注入，仍保留 |
| B4 | Spark 图片限制标记 | `openai_codex_transform.go` | **上游可见明文**，独立于图片桥接开关，仍保留 |
| B5 | Claude Code todo guard | `openai_messages_todo_guard.go` | **上游可见明文**，Messages 兼容路径注入，仍保留 |
| B6 | `python__codex` 工具别名 | `openai_codex_tool_names.go` | **上游可见明文**，具有保留工具名兼容作用，仍保留 |
| C1 | 超长调用 ID 哈希前缀 | `openai_codex_transform.go` | 内部哈希输入，不以该字符串明文出站 |
| C2 | device/session/full 指纹收敛 | `openai_codex_fingerprint.go` | 显式配置会改变身份区分度；默认关闭，仍保留 |
| C3 | API Key / 账号会话隔离 | `openai_codex_account_identity.go` | 正常隔离机制，与可选收敛不同，仍保留 |
| C4 | compact 探测 ID | `openai_compact_probe.go` | 网关自建探测的内部派生，不是明文品牌字段 |
| D1 | 探测、认证、Live 的生成身份 | `openai_codex_identity.go` 等 | 网关自建请求，保留独立策略 |
| D2 | TLS / HTTP 传输特征 | `account.go`、`openai_plugin_transport.go` | **未验证与官方 CLI 等价，不能标成已确认的错误指纹** |

其他供应商和辅助请求中的固定 UA，见第 7 节。上述类别覆盖本次与请求指纹有关的核查项，不把全仓所有常量都判为错误。

## 2. 官方 CLI 的实际对照与 Ubuntu 版本纠正

本地采集使用官方 **Codex CLI 0.153.4**，macOS 26.6.2 / arm64，`exec` 模式，临时目录、忽略用户配置、本地 HTTP 自定义 provider 和虚拟密钥。完整脱敏记录见 [CLI 请求样本](codex-cli-0.153.4-exec-observation.json)。

```http
User-Agent: codex_exec/0.153.4 (Mac OS 26.6.2; arm64) iTerm.app/3.4.22 (codex_exec; 0.153.4)
originator: codex_exec
accept: text/event-stream
content-type: application/json
```

该样本没有 `version` 请求头。请求体使用 Responses Lite，`input` 含 developer/user 消息和 `additional_tools`，没有顶层 `instructions`、`tools`。

这说明固定 `codex-tui + Ubuntu + x86_64 + xterm-256color` 不能代表所有 CLI 调用。该样本仅证明这个平台、模式和 provider 的行为，不代表交互式 TUI、生产 OAuth 或 TLS 握手要求。

**`Ubuntu 22.4.0` 不一定是笔误。** 官方 `rust-v0.153.4` 的 [UA 构造代码](https://github.com/openai/codex/blob/rust-v0.153.4/codex-rs/login/src/auth/default_client.rs#L164-L175) 使用 `os_info.version()`；[Cargo.lock](https://github.com/openai/codex/blob/rust-v0.153.4/codex-rs/Cargo.lock#L10406-L10420) 固定 `os_info 3.14.0`。其[解析及显示逻辑](https://github.com/stanislav-tkach/os_info/blob/v3.14.0/os_info/src/version.rs#L55-L83) 会执行以下转换：

```text
Ubuntu 发行版字符串：22.04
解析后的数字版本：  (22, 4, 0)
UA 中的显示字符串：22.4.0
```

因此，需要关注的是“固定描述是否符合实际调用方”，不能仅凭发行版名称将 UA 中的 `22.4.0` 改判为错误。本次没有进行 Ubuntu 运行时抓取。

## 3. 请求身份及协议关联问题

### A1. 默认 UA 固定了客户端环境

**文件与位置：** [backend/internal/service/openai_gateway_service.go:40](../../backend/internal/service/openai_gateway_service.go#L40)，常量 `codexCLIUserAgentSuffix`。

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_service.go:40-40 -->
```go
codexCLIUserAgentSuffix = " (Ubuntu 22.4.0; x86_64) xterm-256color"
```

同文件 [第 64 行](../../backend/internal/service/openai_gateway_service.go#L64) 的编译期版本兜底：

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_service.go:64-64 -->
```go
codexCLIVersion = "0.146.0"
```

客户端名称来自 [backend/internal/pkg/openai/request.go:266](../../backend/internal/pkg/openai/request.go#L266)：

<!-- source: 06bb4df4c backend/internal/pkg/openai/request.go:266-266 -->
```go
const CodexDefaultOriginator = "codex-tui"
```

**影响：** 当使用默认规范身份时，实际运行于 macOS、Windows、ARM，或者使用 `codex_exec` 的调用方，可能统一被描述成另一套环境。版本号还可能由后台设置或自动同步覆盖，因此不能说所有出站请求始终都是 `0.146.0`。

**当前状态：** 常量仍保留。提交 `06bb4df4c` 使符合条件的身份透传请求可以保留调用方元数据，见 A4。

### A2. 默认强制重写三个身份字段

**文件与位置：** [backend/internal/service/openai_codex_identity.go:224](../../backend/internal/service/openai_codex_identity.go#L224)，`enforceCodexIdentityHeadersWithUA`。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:232-235 -->
```go
identity := resolveCodexOutboundIdentity(overrideUA)
h.Set("user-agent", identity.userAgent)
h.Set("originator", identity.originator)
h.Set("version", identity.version)
```

开关的进程初值在同文件 [第 51 行](../../backend/internal/service/openai_codex_identity.go#L51)：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:51-55 -->
```go
var codexIdentityEnforcement = func() *atomic.Bool {
	v := &atomic.Bool{}
	v.Store(true)
	return v
}()
```

**触发与影响：** 在调用身份强制改写函数、`originator` 非空且强制改写开启时，客户端提交的三元组会被规范身份替换。该开关与 C2 的指纹收敛是两套逻辑，关闭收敛不等于关闭 UA 改写。

**独立覆盖：** [openai_gateway_forward.go:1483](../../backend/internal/service/openai_gateway_forward.go#L1483) 还应用账号级 UA；[第 1490 行](../../backend/internal/service/openai_gateway_forward.go#L1490) 应用 `ForceCodexCLI`。这两项在透传修复后仍有优先级。

**当前状态：** 默认强制改写行为保留；关闭后的转发行为已修复。

### A3. 版本声明被抬升或重新组合

**文件与位置：** [backend/internal/service/openai_codex_identity.go:18](../../backend/internal/service/openai_codex_identity.go#L18)，`codexUpstreamMinVersion`；[第 249 行](../../backend/internal/service/openai_codex_identity.go#L249)，`pairCodexIdentityHeaders`。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:18-18 -->
```go
const codexUpstreamMinVersion = "0.144.0"
```

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:249-251 -->
```go
if v := strings.TrimSpace(h.Get("version")); v != "" && CompareVersions(v, codexUpstreamMinVersion) < 0 {
	h.Set("version", resolveCodexOutboundIdentity("").version)
}
```

规范身份还会按生效版本重建候选 UA，见 [同文件第 159 行](../../backend/internal/service/openai_codex_identity.go#L159)：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:159-163 -->
```go
version := codexClientVersionFromUA(canonical)
if rebuilt := openai.SetCodexUserAgentVersion(pairedUA, version); rebuilt != "" {
	pairedUA = rebuilt
}
return codexOutboundIdentity{userAgent: pairedUA, originator: originator, version: version}
```

**版本来源：** [setting_gateway_runtime.go:351](../../backend/internal/service/setting_gateway_runtime.go#L351)，`GetOpenAICodexClientVersion`，优先级为后台版本覆盖、自动同步版本、编译期兜底。

**影响：** 出站声明的版本未必是调用方实际运行的版本；只填写一条自定义 UA，也不一定逐字保留其中的版本。仓库中的 `0.144.0` 是现有策略，不是本次重新验证的服务端最低版本。

**当前状态：** 统一身份及合成请求的既有策略保留。A4 修复后，普通身份透传不会因版本较旧而自动抬升。

### A4. 关闭强制改写后仍会修改身份：已修复

**原问题文件：** [backend/internal/service/openai_codex_identity.go:228](../../backend/internal/service/openai_codex_identity.go#L228)。关闭强制改写时仍调用配对逻辑：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:228-231 -->
```go
if !codexIdentityEnforcement.Load() {
	pairCodexIdentityHeaders(h)
	return
}
```

未知客户端可能被替换，`originator` 可能重新配对，旧 `version` 可能被抬升。HTTP 白名单、compact 补值、WS 构造器和 Messages 外层逻辑还会影响字段是否存在。例如 [openai_gateway_forward.go:1461](../../backend/internal/service/openai_gateway_forward.go#L1461) 会补充 compact 版本：

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_forward.go:1461-1463 -->
```go
if req.Header.Get("version") == "" {
	req.Header.Set("version", CodexCanonicalClientVersion())
}
```

**修复方式：** 保留这些共享内部逻辑，在转发请求的最终边界恢复调用方的合法身份头。新增函数在 [openai_codex_identity.go:258](../../backend/internal/service/openai_codex_identity.go#L258)，其中的字段恢复代码为：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:268-277 -->
```go
for _, name := range [...]string{"User-Agent", "Originator", "Version"} {
	headers.Del(name)
	for _, value := range c.Request.Header.Values(name) {
		// Validate the original value before any trimming. Never turn malformed
		// input into a plausible client identity by stripping control bytes.
		if httpguts.ValidHeaderFieldValue(value) && !strings.ContainsAny(value, "\r\n") {
			headers.Add(name, value)
		}
	}
}
```

该函数还检查账号协议、开关、账号自定义 UA、`ForceCodexCLI` 及是否存在真实入站请求。以上摘录不是完整函数。

**各路径的修复入口：**

| 路径 | 当前文件位置 | 调用位置说明 |
| --- | --- | --- |
| Responses / compact | [openai_gateway_forward.go:1506](../../backend/internal/service/openai_gateway_forward.go#L1506) | 身份强制改写之后恢复 |
| passthrough / passthrough compact | [openai_gateway_passthrough.go:717](../../backend/internal/service/openai_gateway_passthrough.go#L717) | 最终请求头恢复 |
| WebSocket | [openai_ws_forwarder_payload.go:182](../../backend/internal/service/openai_ws_forwarder_payload.go#L182) | 握手头恢复 |
| Messages 兼容入口 | [openai_gateway_messages.go:374](../../backend/internal/service/openai_gateway_messages.go#L374) | 外层 ensure/enforce 之后再次恢复 |
| 搜索转 Responses | [openai_alpha_search.go:286](../../backend/internal/service/openai_alpha_search.go#L286) | 专用构造器恢复 |
| 独立 alpha/search | [openai_alpha_search.go:427](../../backend/internal/service/openai_alpha_search.go#L427) | 专用构造器恢复 |

内层 Messages bridge 构造器仍按原约定省略 `originator`，真正的 Messages 外层转发在最后恢复调用方元数据。没有入站请求的合成调用、普通 API-key 转发和显式 UA 覆盖保留各自的策略。

### A5. WebSocket 复用可能继承另一客户端的握手：已修复

**文件与位置：** [backend/internal/service/openai_ws_pool.go:2308](../../backend/internal/service/openai_ws_pool.go#L2308)，`normalizeOpenAIWSHandshakeCompatibility`。

旧基线在未开启收敛时，未将身份三字段纳入兼容键：

<!-- source: 4726bdd08 backend/internal/service/openai_ws_pool.go:2305-2312 -->
```go
func normalizeOpenAIWSHandshakeCompatibility(account *Account, headers http.Header) openAIWSHandshakeCompatibilityKey {
	key := openAIWSHandshakeCompatibilityKey{
		betaFeatures: normalizeOpenAIWSBetaFeatures(headers),
	}
	mode := activeCodexFingerprintMode(account)
	if mode == codexFingerprintOff {
		return key
	}
```

**影响：** 两个请求的 UA、originator 或 version 不同，却可能复用同一连接。WebSocket 握手已经完成，修改下一次请求的头对象不会重发握手。

当前新增的兼容键内容：

<!-- source: 06bb4df4c backend/internal/service/openai_ws_pool.go:2314-2318 -->
```go
if !codexIdentityEnforcement.Load() && account != nil && account.UsesOpenAICodexProtocol() {
	key.userAgent = fmt.Sprintf("%q", headers.Values("User-Agent"))
	key.originator = fmt.Sprintf("%q", headers.Values("Originator"))
	key.version = fmt.Sprintf("%q", headers.Values("Version"))
}
```

**修复效果：** 在关闭强制改写的 Codex 转发模式中，所有头值都参与比较，包括第二个值及“字段缺失 / 显式空值”的差别。相关回归见 [openai_ws_pool_test.go:638](../../backend/internal/service/openai_ws_pool_test.go#L638)。

### A6. 不同原生 tool call ID 被归并：已修复

**文件与位置：** [backend/internal/service/openai_gateway_forward.go:521](../../backend/internal/service/openai_gateway_forward.go#L521)，`Forward`；[openai_codex_transform.go:101](../../backend/internal/service/openai_codex_transform.go#L101)，调用 ID 规范化分支。

旧基线的原生 Codex 转换没有开启已有的保留选项：

<!-- source: 4726bdd08 backend/internal/service/openai_gateway_forward.go:521-525 -->
```go
codexResult = applyCodexOAuthTransformWithOptions(decoded, codexOAuthTransformOptions{
	IsCodexCLI:                          isCodexCLI,
	IsCompact:                           isCompactRequest,
	OmitPromotedSystemMessagesFromInput: omitPromotedSystemMessages,
})
```

可能出现的映射示意：

```text
原始 call_abc → fc_abc
原始 fc_abc   → fc_abc
```

这是调用关联错误：两个独立调用可能合并，不能只把它描述为“外观不像官方 ID”。

提交 `06bb4df4c` 在原生入口增加：

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_forward.go:524-524 -->
```go
PreserveToolCallIDs:                 true,
```

现在合法原生 ID 保留原值，调用和输出保持关联；超长值继续按既有长度规则处理，见 C1。回归见 [openai_gateway_call_id_preservation_test.go:14](../../backend/internal/service/openai_gateway_call_id_preservation_test.go#L14)。

### A7. 跨轮引用仍按旧规则改写：已修复

**文件与位置：** [backend/internal/service/openai_codex_transform.go:1660](../../backend/internal/service/openai_codex_transform.go#L1660)，`filterCodexInputWithOptions`。

旧基线：

<!-- source: 4726bdd08 backend/internal/service/openai_codex_transform.go:1660-1660 -->
```go
newItem["id"] = normalizeCodexCallID(trimmedID)
```

已修复源码（`06bb4df4c`）：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_transform.go:1660-1660 -->
```go
newItem["id"] = normalizeCodexFilterCallID("function_call", trimmedID, opts.PreserveCallIDs)
```

**触发：** `item_reference.id` 以 `call_` 开头，本轮没有对应调用或有效 item ID，进入跨轮回退路径。现在该路径也遵守 ID 保留选项。真实 item ID 的优先级、验证及非法 item ID 清理并未取消。

## 4. 上游可见的提示内容和工具名

### B1. 默认 instructions 注入：正文变化，但没有已指认的品牌文本

**文件与位置：** [backend/internal/service/openai_gateway_forward.go:387](../../backend/internal/service/openai_gateway_forward.go#L387)，默认补值入口；[openai_codex_transform.go:1404](../../backend/internal/service/openai_codex_transform.go#L1404)，`defaultCodexSynthInstructions`。

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_forward.go:387-391 -->
```go
instructions := gjson.GetBytes(body, "instructions")
instructionsEmpty := !instructions.Exists() || instructions.Type != gjson.String || strings.TrimSpace(instructions.String()) == ""
if instructionsEmpty && account.UsesOpenAICodexProtocol() && !compatMessagesBridge && !nativeCNResponses {
	markPatchSet("instructions", defaultCodexSynthInstructions(upstreamModel))
}
```

**触发：** 该入口把缺失、非字符串、空白 instructions 视为空；还受 Codex 协议及兼容路径条件控制。模型专用提示内容来自 [backend/internal/pkg/openai/constants.go:58](../../backend/internal/pkg/openai/constants.go#L58) 嵌入的 `instructions*.txt`。

**核验结论：** 本次检查的这些内置文件没有 `sub2api` 字样。默认补值逻辑保留普通非空客户端 instructions。它与本地 CLI 样本的正文结构不同，但不能据此推断 OAuth 接口禁止补值。

**另一路径：** passthrough 中，这条规则只作用于采用 Codex 协议的账号，且模型满足 `isOpenAICodexModel(reqModel)`（名称包含 `codex`）。[openai_gateway_passthrough.go:166](../../backend/internal/service/openai_gateway_passthrough.go#L166) 会为缺失字段补默认值；显式空白或非字符串可能先由 [openai_gateway_request_body.go:1398](../../backend/internal/service/openai_gateway_request_body.go#L1398) 校验并返回 403。普通 GPT-5 OAuth passthrough 不满足这条模型条件，因此不能把所有入口概括成同一种补值行为。

### B2. 强制模板可替换客户端 instructions

**文件与位置：** [backend/internal/service/openai_codex_instructions_template.go:36](../../backend/internal/service/openai_codex_instructions_template.go#L36)，模板渲染后的赋值。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_instructions_template.go:36-36 -->
```go
reqBody["instructions"] = rendered
```

**触发：** 当前生产调用入口是 [openai_gateway_messages.go:219](../../backend/internal/service/openai_gateway_messages.go#L219) 中 `ForwardAsAnthropic` 的 Codex 协议分支，即 Messages→Codex 桥接；配置模板并成功渲染出非空内容时才替换。与 B1 的默认补值不同，这里会覆盖原字段。需要保留已有提示内容时，模板必须引用 `{{ .ExistingInstructions }}`。此模板并非对每个原生 Responses 请求应用。

**当前状态：** 可选配置，仍保留；不是所有请求默认触发。配置说明见 [config.go:995](../../backend/internal/config/config.go#L995)。

### B3. 图片桥接包含明确的 Sub2API 标记

**文件与位置：** [backend/internal/service/openai_codex_transform.go:138](../../backend/internal/service/openai_codex_transform.go#L138)，桥接标记定义：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_transform.go:138-138 -->
```go
codexImageGenerationBridgeMarker = "<codex-image-generation-bridge>"
```

同文件 [第 1118 行](../../backend/internal/service/openai_codex_transform.go#L1118)，`applyCodexImageGenerationBridgeInstructions` 会将桥接文本追加到 instructions：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_transform.go:1118-1118 -->
```go
reqBody["instructions"] = existing + "\n\n" + codexImageGenerationBridgeText
```

**触发条件：** 需要桥接开关生效、请求被识别为 Codex 客户端、非 Responses Lite、分组允许图片，且账号没有剥离图片工具；compact 路径跳过。已有客户端 `image_gen` 函数工具、没有原生图片工具、Spark 模型或已经包含标记时，追加函数也会跳过。

**配置位置：** [openai_gateway_service.go:615](../../backend/internal/service/openai_gateway_service.go#L615)，账号覆盖优先于渠道，再到全局；[config.go:2377](../../backend/internal/config/config.go#L2377) 的全局默认值为 `false`。

**判断与状态：** 属于上游可见的功能性明文标记，也用于避免重复注入；仍保留。这里可以确认存在文本差异，不能确认其会导致封禁。

### B4. Spark 图片限制有独立标记

**文件与位置：** [backend/internal/service/openai_codex_transform.go:140](../../backend/internal/service/openai_codex_transform.go#L140)。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_transform.go:140-140 -->
```go
codexSparkImageUnsupportedMarker = "<codex-spark-image-unsupported>"
```

激活入口位于 [同文件第 301 行](../../backend/internal/service/openai_codex_transform.go#L301)：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_transform.go:301-303 -->
```go
if isCodexSparkModel(normalizedModel) && applyCodexSparkImageUnsupportedInstructions(reqBody) {
	result.Modified = true
}
```

**触发：** 模型规范化后为 `gpt-5.3-codex-spark`，追加函数将模型图片能力限制写入 instructions。该逻辑独立于 B3 开关，纯文本 Spark 请求也可能携带这段提示。

**判断与状态：** 上游可见的提示策略，仍保留；不能写成“关闭图片桥接即可取消所有图片相关标记”。

### B5. Messages 兼容桥添加 todo guard

**文件与位置：** [backend/internal/service/openai_messages_todo_guard.go:11](../../backend/internal/service/openai_messages_todo_guard.go#L11)。

<!-- source: 06bb4df4c backend/internal/service/openai_messages_todo_guard.go:11-11 -->
```go
openAICompatClaudeCodeTodoGuardMarker = "<claude-code-todo-guard>"
```

Codex 协议的 Messages 入口位于 [openai_gateway_messages.go:241](../../backend/internal/service/openai_gateway_messages.go#L241)：

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_messages.go:241-243 -->
```go
if shouldAutoInjectPromptCacheKeyForCompat(upstreamModel) {
	appendOpenAICompatClaudeCodeTodoGuardToRequestBody(reqBody)
}
```

**触发与位置：** 满足模型门控并存在非空输入时，提示作为 `input` 中的 developer 消息追加；已有标记时不重复。模型门控见 [openai_compat_prompt_cache_key.go:14](../../backend/internal/service/openai_compat_prompt_cache_key.go#L14)。非 Codex 协议的 Messages 分支也有相应入口。

**判断与状态：** 明文会随正文出站，属于 Messages 兼容逻辑，仍保留；不要误写成所有原生 Responses 请求都会注入。

### B6. 保留工具名 python 被映射为 python__codex

**文件与位置：** [backend/internal/service/openai_codex_tool_names.go:15](../../backend/internal/service/openai_codex_tool_names.go#L15)，常量；[第 64 行](../../backend/internal/service/openai_codex_tool_names.go#L64)，名称改写。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_tool_names.go:15-15 -->
```go
codexPythonToolAlias        = "python__codex"
```

<!-- source: 06bb4df4c backend/internal/service/openai_codex_tool_names.go:64-66 -->
```go
if strings.EqualFold(trimmed, codexReservedPythonToolName) {
	return codexPythonToolAlias
}
```

**触发：** Codex 转换中的函数工具声明、工具选择或调用节点使用保留名 `python` 时。代码会检查别名冲突，并记录反向映射供响应还原。

**判断与状态：** 名称确实能被上游看到，但具有保留工具名兼容目的；仍保留。仅看到 `sub2api` 不能证明这是协议错误，单方面改名也可能破坏响应还原。

## 5. 哈希、收敛与会话隔离

### C1. 超长 tool call ID 的哈希字符串不是明文请求标记

**文件与位置：** [backend/internal/service/openai_codex_transform.go:119](../../backend/internal/service/openai_codex_transform.go#L119)，`compactCodexCallIDForItemType`。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_transform.go:119-124 -->
```go
func compactCodexCallIDForItemType(itemType, id string) string {
	prefix := openAIResponsesToolCallIDPrefix(itemType) + "_"
	digest := sha256.Sum256([]byte("sub2api:codex-call-id:v1:" + id))
	encoded := hex.EncodeToString(digest[:])
	return prefix + encoded[:codexCallIDMaxLength-len(prefix)]
}
```

**触发：** 原始超长 ID 进入规范化后，结果仍超过 64 字节才执行这段压缩。哈希中的 `id` 可能已做过前缀规范化，因此不能把每个原始 65 字节 ID 都描述成直接对原字符串做 SHA-256。

**实际出站：** `fc_`、`ctc_` 或 `tsc_` 等类型前缀与十六进制摘要组合。`sub2api:codex-call-id:v1:` 是参与计算的域分隔字符串，不作为这个字段的明文值发送。

**判断与状态：** 压缩逻辑保留；正常长度原生 ID 的错误改写已由 A6 修复。改动哈希命名空间会改变稳定映射，不能把“更换盐值”当成已验证的兼容性修复。

### C2. 可选收敛会改变多个客户端的身份区分度

**文件与位置：** [backend/internal/service/openai_codex_fingerprint.go:227](../../backend/internal/service/openai_codex_fingerprint.go#L227)，`resolveConvergedInstallationID`。该函数优先返回账号配置的非空 `device_id`；未配置时，才在 [第 239 行](../../backend/internal/service/openai_codex_fingerprint.go#L239) 按账号种子派生安装标识：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_fingerprint.go:239-239 -->
```go
return deriveStableUUIDv4("sub2api:codex-install-id:v2:" + seed)
```

同文件 [第 247 行](../../backend/internal/service/openai_codex_fingerprint.go#L247)，会话标识派生：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_fingerprint.go:247-247 -->
```go
return deriveStableUUIDv4("sub2api:codex-session-id:v2:" + seed)
```

同文件 [第 257 行](../../backend/internal/service/openai_codex_fingerprint.go#L257)，线程标识派生：

<!-- source: 06bb4df4c backend/internal/service/openai_codex_fingerprint.go:257-257 -->
```go
return deriveStableUUIDv4("sub2api:codex-thread-id:v2:" + seed + ":" + clientSessionID)
```

`deriveStableUUIDv4` 位于 [第 214 行](../../backend/internal/service/openai_codex_fingerprint.go#L214)，根据哈希构造 UUID；上游看到 UUID，不会直接看到上述 `sub2api:` 字符串。

| 模式 | 主要效果 | 需要关注的区别 |
| --- | --- | --- |
| `off` | 不应用可选收敛 | 不等于关闭 A2，也不等于关闭 C3 |
| `device` | 优先使用账号配置的 `device_id`，否则按账号种子派生安装 ID | 多个调用方可能呈现同一安装身份 |
| `session` | 账号级 session；thread 按原客户端 session 派生 | 相同原 session 得到相同 thread；缺少原 session 时 thread 回退为账号 session |
| `full` | session 与 thread 使用同一个收敛标识 | 原本不同的会话、线程区分度进一步降低 |

**触发：** 显式选择模式并有有效账号种子；未配置、空值及非法模式默认 `off`。模式解析见 [第 113 行](../../backend/internal/service/openai_codex_fingerprint.go#L113)，收敛计算见 [第 307 行](../../backend/internal/service/openai_codex_fingerprint.go#L307)。

**写入位置：** 头部写入见 [第 363 行](../../backend/internal/service/openai_codex_fingerprint.go#L363)，body 写入见 [第 459 行](../../backend/internal/service/openai_codex_fingerprint.go#L459)。原 `prompt_cache_key` 与 body 原 session 对应时，还可能随之改写，见 [第 503 行](../../backend/internal/service/openai_codex_fingerprint.go#L503)。

**判断与状态：** 属于显式的关联策略，仍保留。需要关注标识区分度和缓存关联，但本次没有证明跨用户数据泄漏或服务端处罚因果。

### C3. API Key / 账号隔离不是可选指纹收敛

**文件与位置：** [backend/internal/service/openai_codex_account_identity.go:93](../../backend/internal/service/openai_codex_account_identity.go#L93)，账号作用域 ID 派生。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_account_identity.go:99-102 -->
```go
return deriveStableUUIDv4(fmt.Sprintf(
	"sub2api:codex-account-identity:%s:user:%d:account:%s:kind:%s:value:%s",
	codexAccountIdentityNamespaceVersion,
	apiKeyID,
```

**作用：** 派生输入包含 API Key ID、账号命名空间、标识类型和原始值，以免不同网关用户或不同上游凭据共享同一标识。不同隔离函数可输出 UUID 或短十六进制值，并非直接发送整个格式字符串。

**调用次序：** [openai_gateway_forward.go:1496](../../backend/internal/service/openai_gateway_forward.go#L1496) 先应用账号隔离，然后应用可选收敛：

<!-- source: 06bb4df4c backend/internal/service/openai_gateway_forward.go:1494-1499 -->
```go
// 账号 namespace 不改变客户端身份基数，但确保 scheduler failover 后不会把
// 同一组 Codex IDs 发送给另一份 OAuth 凭据。可选指纹收敛随后仍可覆盖这些值。
applyCodexAccountIdentityHeaders(req.Header, codexAccountIdentitySource(c, account), getAPIKeyIDFromContext(c))

// 指纹收敛：使用 Forward() 中预计算的收敛 ID 改写出站头，与请求体使用同一份 IDs。
applyStagedCodexFingerprintHeaders(c, account, req.Header)
```

**判断与状态：** 隔离机制保留。关闭可选收敛时，隔离仍可能执行；会话标识发生变化不能直接等同于“错误指纹”。

### C4. compact 探测使用网关自己的稳定 ID

**文件与位置：** [backend/internal/service/openai_compact_probe.go:163](../../backend/internal/service/openai_compact_probe.go#L163)。

<!-- source: 06bb4df4c backend/internal/service/openai_compact_probe.go:162-165 -->
```go
if accountID <= 0 {
	return deriveStableUUIDv4("sub2api:codex-compact-probe:v1:anonymous")
}
return deriveStableUUIDv4("sub2api:codex-compact-probe:v1:" + strconv.FormatInt(accountID, 10))
```

**判断与状态：** 这是合成探测请求的派生输入，上游接收派生值。与用户真实调用的工具 ID、客户端安装 ID 分别处理；没有证据支持把此 namespace 改名当成兼容性修复。

## 6. 合成请求与传输层

### D1. 探测、认证与 Live 仍有生成身份

**文件与位置：** [backend/internal/service/openai_codex_identity.go:196](../../backend/internal/service/openai_codex_identity.go#L196)，`applyOpenAICodexProbeHeaders`。

<!-- source: 06bb4df4c backend/internal/service/openai_codex_identity.go:196-202 -->
```go
func applyOpenAICodexProbeHeaders(h http.Header) {
	if h == nil {
		return
	}
	ensureCodexIdentityHeaders(h)
	h.Set("X-Codex-Window-ID", uuid.NewString())
}
```

相关入口：

| 请求类型 | 文件与位置 | 行为 |
| --- | --- | --- |
| 账号用量探测 | [account_usage_service.go:872](../../backend/internal/service/account_usage_service.go#L872) | 设置规范身份；可能读取缓存 UA，再应用既有配对逻辑 |
| 账号测试 | [account_test_service.go:867](../../backend/internal/service/account_test_service.go#L867) | 调用合成探测头构造函数 |
| 模型发现 | [upstream_models.go:1095](../../backend/internal/service/upstream_models.go#L1095) | 应用规范身份或账号覆盖 |
| 认证身份辅助函数 | [openai_codex_identity.go:99](../../backend/internal/service/openai_codex_identity.go#L99) | 写入 UA 与 originator，和推理请求的 version 策略分开 |
| Live / sideband | [openai_live.go:402](../../backend/internal/service/openai_live.go#L402) | 构造自身的协议头和缺失的会话、线程 ID |

**判断与状态：** 这些请求由网关生成，不能假设都有可原样转发的客户端头。A4 修复保留了它们的既有逻辑，因此不能宣称“开启透传后所有网关出站请求都不再包含规范身份”。

### D2. 修改 HTTP 头不能证明 TLS 指纹一致

**文件与位置：** [backend/internal/service/account.go:2377](../../backend/internal/service/account.go#L2377)，`IsTLSFingerprintEnabled` 的账号类型限制：

<!-- source: 06bb4df4c backend/internal/service/account.go:2377-2381 -->
```go
func (a *Account) IsTLSFingerprintEnabled() bool {
	// 仅支持 Anthropic OAuth/SetupToken 账号
	if !a.IsAnthropicOAuthOrSetupToken() {
		return false
	}
```

OpenAI 实际请求分发见 [backend/internal/service/openai_plugin_transport.go:11](../../backend/internal/service/openai_plugin_transport.go#L11)：

<!-- source: 06bb4df4c backend/internal/service/openai_plugin_transport.go:11-19 -->
```go
func (s *OpenAIGatewayService) doOpenAIUpstream(request *http.Request, proxyURL string, account *Account) (*http.Response, error) {
	if s.pluginManager != nil {
		response, handled, err := s.pluginManager.RoundTripOpenAIOAuth(request.Context(), request, proxyURL, account)
		if handled {
			return response, err
		}
	}
	return s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
}
```

**可确认：** 这处账号级 TLS 模板开关限制在 Anthropic OAuth/SetupToken；OpenAI 请求还可能由插件处理，否则走配置的 HTTPUpstream。仅查看 UA，不能证明实际 ClientHello、HTTP/2 行为或代理后的传输特征与官方 CLI 一样。

**当前结论：** 未做生产 TLS 抓取和对照，属于未验证项。不能把现有模板开关称为已经实现的 Codex TLS 等价方案。

## 7. 其他供应商及辅助请求中的固定 UA

这些位置也存在固定值或明文品牌标识，但属于不同协议或工具请求。未进行相应供应商的官方客户端对照，因此这里只确认代码行为，不标记为“已经证实不符合该供应商协议”。

| 场景 | 文件与位置 | 示例代码摘录 | 判断 |
| --- | --- | --- | --- |
| Grok OAuth 换码 / 刷新 | [grok_oauth_client.go:69](../../backend/internal/repository/grok_oauth_client.go#L69)、[第 101 行](../../backend/internal/repository/grok_oauth_client.go#L101) | `SetHeader("User-Agent", "sub2api-grok-oauth/1.0")` | 明文品牌 UA；不属于 Codex 推理流量 |
| Ollama Cloud 用量查询 | [ollama_cloud_usage.go:892](../../backend/internal/service/ollama_cloud_usage.go#L892) | `req.Header.Set("User-Agent", "sub2api-ollama-usage/1")` | 明文品牌 UA；网关的用量查询请求 |
| Gemini CLI 集成 | [geminicli/constants.go:50](../../backend/internal/pkg/geminicli/constants.go#L50) | `GeminiCLIUserAgent = "GeminiCLI/0.1.5 (Windows; AMD64)"` | 固定版本、OS 和架构，不代表任意实际客户端 |
| Claude 用量查询兜底 | [claude_usage_service.go:19](../../backend/internal/repository/claude_usage_service.go#L19) | `const defaultUsageUserAgent = "claude-code/2.1.7"` | 固定兜底版本；代码也可能使用传入的指纹 UA |
| Grok 账单查询 | [xai/billing.go:25](../../backend/internal/pkg/xai/billing.go#L25) | `billingCLIUserAgent = "grok-pager/" + CLIClientVersion + " grok-shell/" + CLIClientVersion + " (macos; aarch64)"` | 固定平台；此处 `CLIClientVersion` 为 `0.2.120` |
| OpenAI 图片下载兜底 | [openai_images.go:40](../../backend/internal/service/openai_images.go#L40)、[第 1602 行](../../backend/internal/service/openai_images.go#L1602) | `userAgent = openAIImageBackendUserAgent` | 无可用 UA 时使用带 Windows/x64、Chrome 131 的浏览器样式值 |
| 管理员代理质量探测 | [admin_service.go:679](../../backend/internal/service/admin_service.go#L679)、[admin_proxy.go:361](../../backend/internal/service/admin_proxy.go#L361) | `req.Header.Set("User-Agent", proxyQualityClientUserAgent)` | 固定 Windows/x64、Chrome 136 的探测 UA |

图片下载的完整常量示例：

<!-- source: 06bb4df4c backend/internal/service/openai_images.go:40-40 -->
```go
openAIImageBackendUserAgent            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
```

## 8. 当前修复范围与复核方式

**已经修复并验证：** A4、A5、A6、A7，代码已在本地提交 `06bb4df4c`。默认规范身份、桥接提示、保留工具别名、稳定哈希域及账号隔离没有被统一删除或更名。

身份透传使用现有配置，配置项见 [deploy/config.example.yaml:262](../../deploy/config.example.yaml#L262)：

```yaml
gateway:
  disable_codex_identity_enforcement: true
  force_codex_cli: false
```

账号级 UA 覆盖仍优先。缺失身份字段保持缺失；HTTP 库仍可能自行使用其默认 UA。该配置不关闭账号隔离或另行配置的收敛模式。

上一轮代码改动已通过完整后端单元测试（56 个包）、集成测试（50 个包）、golangci-lint v2.13.0、service go vet 及独立审查。具体命令、环境和范围见 [兼容性审计与验证记录](codex-request-compatibility.md)。

本次文档补充只核对源码摘录、文件链接、版本区分和结论依据，不改动运行时代码，不重新宣称完成新的生产抓取。
