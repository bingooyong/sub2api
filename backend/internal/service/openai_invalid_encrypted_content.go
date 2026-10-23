package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// isOpenAIInvalidEncryptedContentUpstreamError 判定上游错误是否表示
// 「客户端回放的加密推理密文（reasoning.encrypted_content）已失效」。
//
// 判定不能只依赖 error.code。上游（OpenAI 官方与多数中转）对该错误的常见形态是
// code=null、type=invalid_request_error，错误语义只体现在 message 文案里：
//
//	{"error":{"message":"The encrypted content gAAA... could not be verified.
//	 Reason: Encrypted content could not be decrypted or parsed.",
//	 "type":"invalid_request_error","param":"","code":null}}
//
// extractUpstreamErrorCode 在 error.code 为 null 时返回空串，只认 code 的判定会漏判，
// 导致自愈重试不触发、400 原样回客户端，同一份历史被客户端反复重放 → 对话卡死。
//
// 判定口径与 WS 路径共用本函数，避免两条路径再次漂移（同语义不同判定是本缺陷的直接成因）。
func isOpenAIInvalidEncryptedContentUpstreamError(code, msg string) bool {
	if strings.EqualFold(strings.TrimSpace(code), "invalid_encrypted_content") {
		return true
	}
	lowered := strings.ToLower(strings.TrimSpace(msg))
	if strings.Contains(lowered, "invalid_encrypted_content") {
		return true
	}
	if !strings.Contains(lowered, "encrypted content") {
		return false
	}
	// 要求同时出现「密文对象」与「不可验证/不可解密」两个语义要素，
	// 避免把无关的签名类错误（Invalid signature）误判为密文失效。
	return strings.Contains(lowered, "could not be verified") ||
		strings.Contains(lowered, "could not be decrypted")
}

// openAIInvalidEncryptedContentItemIDPattern 匹配上游错误文案里的失效 item id，
// 形如 "The encrypted content for item amsg_01a09b80-... could not be verified"。
var openAIInvalidEncryptedContentItemIDPattern = regexp.MustCompile(`(?i)encrypted content for item ([A-Za-z0-9_.-]+)`)

// openAIInvalidEncryptedContentItemID 从上游错误文案中解析出失效的 item id。
// 上游在密文解不开时会点名是哪个 item（message 用 amsg_ 前缀，reasoning 用 rs_ 前缀）。
// 解析不出来时返回空串。
func openAIInvalidEncryptedContentItemID(msg string) string {
	match := openAIInvalidEncryptedContentItemIDPattern.FindStringSubmatch(msg)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

// stripOpenAIInvalidEncryptedContentItemID 从请求体的 input 中找到 id 等于失效
// item id 的项，并删掉它的 id 字段（保留其余内容）。
//
// 为什么要删 id 而不是删整项：上游对已知 id 会做密文 lookup（hydration），
// lookup 失败才报 400。删掉 id 后上游不再尝试 hydrate 旧密文，item 的可见内容
// （message 正文、summary 等）仍然保留，对话上下文不丢。这与 codex 传输层
// filterCodexInputWithOptions 剥 reasoning id 的做法同源。
//
// 返回是否发生了修改。
func stripOpenAIInvalidEncryptedContentItemID(reqBody map[string]any, itemID string) bool {
	if reqBody == nil || strings.TrimSpace(itemID) == "" {
		return false
	}
	inputValue, has := reqBody["input"]
	if !has {
		return false
	}
	items, ok := inputValue.([]any)
	if !ok {
		return false
	}
	changed := false
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := item["id"].(string)
		if strings.TrimSpace(id) != itemID {
			continue
		}
		delete(item, "id")
		changed = true
	}
	return changed
}

// trimOpenAIInvalidEncryptedContentItems 剥除请求体中所有会导致上游 hydration 失败的
// 加密内容，返回是否发生了修改。
//
// 与 trimOpenAIEncryptedReasoningItems 的差异：后者只处理 reasoning 与
// compaction/compaction_summary 两类项，而 OpenAI Responses 在 store=false 的
// 回放场景下会给 **assistant message** 项也带上 encrypted_content。message 的
// 密文失效时上游报的是：
//
//	The encrypted content for item amsg_xxx could not be verified.
//	Reason: encrypted content hydration failed: Encrypted content could not be decrypted or parsed.
//
// 只剥 reasoning 会让这类错误「判定命中但无可剥之物」，从而放弃自愈、把 400
// 原样回给客户端，客户端重放同一份历史 → 对话卡死。
//
// 剥离语义：删掉 message 项的 encrypted_content 字段，保留 id 与可见内容。
func trimOpenAIInvalidEncryptedContentItems(reqBody map[string]any) bool {
	if len(reqBody) == 0 {
		return false
	}
	inputValue, has := reqBody["input"]
	if !has {
		return false
	}
	items, ok := inputValue.([]any)
	if !ok {
		return false
	}
	changed := false
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		itemType, _ := item["type"].(string)
		if strings.TrimSpace(itemType) != "message" {
			continue
		}
		if _, has := item["encrypted_content"]; !has {
			continue
		}
		delete(item, "encrypted_content")
		changed = true
	}
	return changed
}

// openAIInputShapeSummary 概括请求体 input 的结构，用于诊断密文失效问题。
//
// 只输出每个 item 的类型，以及是否带 id / 是否带 encrypted_content 字段，
// **绝不输出密文本身**（密文是敏感数据，且日志会被收集到远端）。
// 形如：[message(id,enc) reasoning(enc) message(-)]
func openAIInputShapeSummary(reqBody map[string]any) string {
	if reqBody == nil {
		return ""
	}
	inputValue, has := reqBody["input"]
	if !has {
		return "no-input"
	}
	switch input := inputValue.(type) {
	case []any:
		parts := make([]string, 0, len(input))
		for _, raw := range input {
			item, ok := raw.(map[string]any)
			if !ok {
				parts = append(parts, "?")
				continue
			}
			itemType, _ := item["type"].(string)
			if strings.TrimSpace(itemType) == "" {
				itemType = "?"
			}
			idValue, _ := item["id"].(string)
			if len(idValue) > 24 {
				idValue = idValue[:24] + "…"
			}
			role, _ := item["role"].(string)
			_, hasEnc := item["encrypted_content"]
			parts = append(parts, fmt.Sprintf("%s(role=%s,id=%s,enc=%t)", itemType, role, idValue, hasEnc))
		}
		return "[" + strings.Join(parts, " ") + "]"
	case string:
		return "string"
	default:
		return "other"
	}
}

// openAIInputShapeSummaryFromBytes 从原始请求体字节概括 input 结构（诊断用）。
func openAIInputShapeSummaryFromBytes(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "unparsable"
	}
	return openAIInputShapeSummary(decoded)
}

// stripOpenAIAssistantHistoryForInvalidEncryptedContent 是密文失效的最后一道兜底：
// 当上游坚持认为历史里的密文解不开、而前两级自愈（剥 reasoning/compaction 密文、
// 剥 message 密文）都无从下手时，丢弃 assistant 侧的历史，只保留 user 消息与
// 工具调用/工具输出。
//
// 代价是模型看不到之前的 assistant 回答与加密推理；收益是请求能成功、对话能继续。
// 对话卡死（客户端反复重放同一份历史）比丢一段历史更糟，所以这个取舍是划算的。
//
// 只在「确认是密文失效错误」时调用，不影响任何其它 400。
func stripOpenAIAssistantHistoryForInvalidEncryptedContent(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	inputValue, has := reqBody["input"]
	if !has {
		return false
	}
	items, ok := inputValue.([]any)
	if !ok {
		return false
	}
	kept := make([]any, 0, len(items))
	changed := false
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			kept = append(kept, raw)
			continue
		}
		itemType := strings.TrimSpace(strings.ToLower(firstNonEmptyString(item["type"])))
		role := strings.TrimSpace(strings.ToLower(firstNonEmptyString(item["role"])))
		switch {
		case itemType == "reasoning",
			itemType == "compaction",
			itemType == "compaction_summary",
			role == "assistant":
			changed = true
			continue
		}
		kept = append(kept, raw)
	}
	if !changed {
		return false
	}
	if len(kept) == 0 {
		delete(reqBody, "input")
		return true
	}
	reqBody["input"] = kept
	return true
}

// trimOpenAIInputToLastUserMessage 是密文失效自愈的终极兜底：把 input 裁剪成
// 「只保留最后一条 user 消息」。
//
// 适用场景：上游坚持认为历史里的密文解不开，而前面的自愈动作（剥 reasoning 密文、
// 剥 message 密文、丢 previous_response_id、删 assistant 侧历史）都无从下手——
// 典型是客户端回放的 item 不带 role、也不带 encrypted_content，但上游仍按某个
// 服务端 id 去 hydrate。
//
// 代价：模型几乎看不到之前的对话上下文。收益：请求一定发得出去，对话不会卡死。
// 客户端下一轮会把完整历史再带回来，届时若上游仍失败会再次触发本兜底。
func trimOpenAIInputToLastUserMessage(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	inputValue, has := reqBody["input"]
	if !has {
		return false
	}
	items, ok := inputValue.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	lastUserIndex := -1
	for i := len(items) - 1; i >= 0; i-- {
		item, ok := items[i].(map[string]any)
		if !ok {
			continue
		}
		role := strings.TrimSpace(strings.ToLower(firstNonEmptyString(item["role"])))
		itemType := strings.TrimSpace(strings.ToLower(firstNonEmptyString(item["type"])))
		if role == "user" || (itemType == "message" && role == "") {
			lastUserIndex = i
			break
		}
	}
	if lastUserIndex < 0 {
		return false
	}
	kept := make([]any, 0, 1)
	kept = append(kept, items[lastUserIndex])
	if len(kept) == len(items) {
		return false
	}
	reqBody["input"] = kept
	return true
}

// openAIOutboundBodySummary 概括**出站**请求体：顶层 key 列表 + 关键续链字段 +
// input 的结构摘要。绝不输出 instructions 正文与用户对话内容。
//
// 诊断要点：必须看**出站** body。客户端原始 body 会被 codex transform 改写
// （剥掉 item 的 id、剥掉 reasoning 项），用原始 body 判断会得出错误结论。
func openAIOutboundBodySummary(body []byte) string {
	if len(body) == 0 {
		return "empty"
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "unparsable"
	}
	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		if key == "instructions" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{"keys=" + strings.Join(keys, ",")}
	for _, key := range []string{"previous_response_id", "conversation", "include", "store", "reasoning", "model"} {
		value, ok := decoded[key]
		if !ok {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			continue
		}
		text := string(raw)
		if len(text) > 120 {
			text = text[:120] + "…"
		}
		parts = append(parts, key+"="+text)
	}
	parts = append(parts, "input="+openAIInputShapeSummary(decoded))
	return strings.Join(parts, " ")
}
