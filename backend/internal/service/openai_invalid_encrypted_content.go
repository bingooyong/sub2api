package service

import "strings"

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
