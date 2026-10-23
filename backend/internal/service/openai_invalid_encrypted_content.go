package service

import (
	"regexp"
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
