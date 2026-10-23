package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// productionInvalidEncryptedContentBody 是 2026-10-23 在 160 生产环境取到的真实错误体：
// error.code 为 null，语义只体现在 message 文案里。
const productionInvalidEncryptedContentBody = `{"error":{"message":"The encrypted content gAAA...3WpB could not be verified. Reason: Encrypted content could not be decrypted or parsed. (request id: 2026102307090876042774j8bHcbc8) [trace_id: 5521fc999b5fec4c9663eef7ddb1e9b5]","type":"invalid_request_error","param":"","code":null}}`

func TestIsOpenAIInvalidEncryptedContentUpstreamError(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
		msg  string
		want bool
	}{
		{
			name: "production shape with null code",
			code: "",
			msg:  "The encrypted content gAAA...3WpB could not be verified. Reason: Encrypted content could not be decrypted or parsed.",
			want: true,
		},
		{
			name: "legacy invalid_encrypted_content code",
			code: "invalid_encrypted_content",
			msg:  "bad encrypted content",
			want: true,
		},
		{
			name: "legacy thinking_signature_invalid code",
			code: "thinking_signature_invalid",
			msg:  "The encrypted content abc could not be verified. Reason: Encrypted content could not be decrypted or parsed.",
			want: true,
		},
		{
			name: "code null in json but message carries the error",
			code: "null",
			msg:  "The encrypted content could not be decrypted or parsed.",
			want: true,
		},
		{
			name: "unrelated signature error must not match",
			code: "thinking_signature_invalid",
			msg:  "Invalid signature",
			want: false,
		},
		{
			name: "ordinary bad request must not match",
			code: "invalid_request_error",
			msg:  "bad request",
			want: false,
		},
		{
			name: "encrypted content without failure wording must not match",
			code: "invalid_request_error",
			msg:  "Missing required parameter 'input[0].encrypted_content'",
			want: false,
		},
		{
			name: "empty input",
			code: "",
			msg:  "",
			want: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isOpenAIInvalidEncryptedContentUpstreamError(tc.code, tc.msg))
		})
	}
}

// TestOpenAIGatewayService_Forward_RecoversFromNullCodeInvalidEncryptedContent
// 锁定生产形态（error.code=null）的自愈：剥掉失效密文后重试一次并成功返回。
func TestOpenAIGatewayService_Forward_RecoversFromNullCodeInvalidEncryptedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(productionInvalidEncryptedContentBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{ID: 10, Name: "openai-apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://example.com"},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.6-sol","stream":false,"previous_response_id":"resp_stale","input":[{"type":"reasoning","id":"rs_stale","encrypted_content":"gAAAstale","summary":[{"type":"summary_text","text":"keep"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2, "失效密文必须只重试一次")
	require.Equal(t, "gAAAstale", gjson.GetBytes(upstream.bodies[0], "input.0.encrypted_content").String())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.encrypted_content").Exists(), "重试请求必须剥掉失效密文")
	require.Equal(t, "keep", gjson.GetBytes(upstream.bodies[1], "input.0.summary.0.text").String(), "推理摘要必须保留")
	require.Equal(t, "hello", gjson.GetBytes(upstream.bodies[1], "input.1.content.0.text").String(), "普通消息必须保留")
	require.Equal(t, "resp_stale", gjson.GetBytes(upstream.bodies[1], "previous_response_id").String(), "HTTP 路径重试必须保留 previous_response_id 续链锚点")
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentKeepsPreviousResponseIDWithFunctionCallOutput
// 含 function_call_output 时 previous_response_id 是工具续链锚点，必须保留。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentKeepsPreviousResponseIDWithFunctionCallOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(productionInvalidEncryptedContentBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{ID: 10, Name: "openai-apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://example.com"},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.6-sol","stream":false,"previous_response_id":"resp_anchor","input":[{"type":"reasoning","id":"rs_stale","encrypted_content":"gAAAstale","summary":[]},{"type":"function_call_output","call_id":"call_1","output":"result"}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2)
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.encrypted_content").Exists())
	require.Equal(t, "resp_anchor", gjson.GetBytes(upstream.bodies[1], "previous_response_id").String(), "含 function_call_output 时必须保留 previous_response_id")
	require.Equal(t, "result", gjson.GetBytes(upstream.bodies[1], "input.1.output").String())
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentRetriesOnlyOnce
// 第二次仍然 400 时必须终止，不得形成重试风暴。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentRetriesOnlyOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(productionInvalidEncryptedContentBody)),
		},
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(productionInvalidEncryptedContentBody)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{ID: 10, Name: "openai-apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://example.com"},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.6-sol","stream":false,"input":[{"type":"reasoning","id":"rs_stale","encrypted_content":"gAAAstale","summary":[]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`
	_, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.Error(t, err)
	require.Len(t, upstream.bodies, 2, "只允许重试一次")
}

// TestOpenAIGatewayService_Forward_DoesNotRetryUnrelatedBadRequest
// 与密文无关的 400 不得触发重试。
func TestOpenAIGatewayService_Forward_DoesNotRetryUnrelatedBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_request_error","type":"invalid_request_error","message":"bad request"}}`)),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{ID: 10, Name: "openai-apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://example.com"},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.6-sol","stream":false,"input":[{"type":"reasoning","id":"rs_stale","encrypted_content":"gAAAstale","summary":[]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`
	_, _ = svc.Forward(context.Background(), c, account, []byte(body))
	require.Len(t, upstream.bodies, 1, "无关 400 不得触发重试")
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentDropsAnchorWhenNoItemIDMatches
// 密文在服务端、input 里也没有可匹配的 id 时，最后手段是断开续链。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentDropsAnchorWhenNoItemIDMatches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverSideBody := `{"error":{"code":"invalid_encrypted_content","message":"The encrypted content for item amsg_missing could not be verified. Reason: encrypted content hydration failed: Encrypted content could not be decrypted or parsed.","param":"input","type":"invalid_request_error"}}`
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(serverSideBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	// apikey 路径：没有 codex transform，previous_response_id 与 item id 都会保留，
	// 因此「丢锚点 / 保留锚点」的兜底动作只在这一路径上可验证。
	account := &Account{ID: 6642, Name: "apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.com",
		},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.1","stream":false,"previous_response_id":"resp_server_side","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2, "无法匹配 item id 时应靠丢锚点恢复")
	require.False(t, gjson.GetBytes(upstream.bodies[1], "previous_response_id").Exists(), "重试必须丢弃 previous_response_id 断开续链")
	require.Equal(t, "hello", gjson.GetBytes(upstream.bodies[1], "input.0.content.0.text").String(), "对话内容必须保留")
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentKeepsAnchorWithFunctionCallOutput
// 含 function_call_output 时锚点是工具续链依据，不能丢。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentKeepsAnchorWithFunctionCallOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverSideBody := `{"error":{"code":"invalid_encrypted_content","message":"The encrypted content for item amsg_x could not be verified. Reason: encrypted content hydration failed: Encrypted content could not be decrypted or parsed.","param":"input","type":"invalid_request_error"}}`
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(serverSideBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	// apikey 路径：没有 codex transform，previous_response_id 与 item id 都会保留，
	// 因此「丢锚点 / 保留锚点」的兜底动作只在这一路径上可验证。
	account := &Account{ID: 6642, Name: "apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.com",
		},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.1","stream":false,"previous_response_id":"resp_anchor","input":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"earlier"}],"encrypted_content":"gAAAencrypted"},{"type":"function_call_output","call_id":"call_1","output":"result"}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2)
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.encrypted_content").Exists(), "message 密文必须被剥掉")
	require.Equal(t, "resp_anchor", gjson.GetBytes(upstream.bodies[1], "previous_response_id").String(), "含 function_call_output 时必须保留锚点")
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentTrimsMessageEncryptedContent
// 真实生产形态：store=false 回放场景下 assistant message 项带 encrypted_content，
// 上游对其 hydration 失败并点名 item id（amsg_ 前缀）。此时唯一的自愈是删掉
// message 的 encrypted_content（保留 id 与可见内容）后重试。
//
// 这是 2026-10-23 oauth 账号（6854）遇到的实际形态：判定命中但既有的
// trimOpenAIEncryptedReasoningItems 不处理 message，导致无可剥之物而放弃自愈。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentTrimsMessageEncryptedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverSideBody := `{"error":{"code":"invalid_encrypted_content","message":"The encrypted content for item amsg_01a09b80-e59f-7500-a42f-a5621a1194f8 could not be verified. Reason: encrypted content hydration failed: Encrypted content could not be decrypted or parsed.","param":"input","type":"invalid_request_error"}}`
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(serverSideBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	// apikey 路径：没有 codex transform，previous_response_id 与 item id 都会保留，
	// 因此「丢锚点 / 保留锚点」的兜底动作只在这一路径上可验证。
	account := &Account{ID: 6642, Name: "apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.com",
		},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.1","stream":false,"input":[{"type":"message","id":"amsg_01a09b80-e59f-7500-a42f-a5621a1194f8","role":"assistant","content":[{"type":"output_text","text":"earlier answer"}],"encrypted_content":"gAAAencrypted"},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2, "message 密文失效必须重试一次")
	require.Equal(t, "gAAAencrypted", gjson.GetBytes(upstream.bodies[0], "input.0.encrypted_content").String(), "首次请求应保留密文")
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.encrypted_content").Exists(), "重试必须剥掉 message 的失效密文")
	require.Equal(t, "earlier answer", gjson.GetBytes(upstream.bodies[1], "input.0.content.0.text").String(), "message 可见内容必须保留")
}

// TestClassifyOpenAIWSErrorEventFromRaw_InvalidEncryptedContentSharedWithHTTPPath
// WS 路径与 HTTP 路径必须共用同一判定：生产形态（code 空/ null）都要归类为
// invalid_encrypted_content。
func TestClassifyOpenAIWSErrorEventFromRaw_InvalidEncryptedContentSharedWithHTTPPath(t *testing.T) {
	msg := "The encrypted content gAAA...3WpB could not be verified. Reason: Encrypted content could not be decrypted or parsed."
	for _, code := range []string{"", "null", "invalid_request_error"} {
		reason, matched := classifyOpenAIWSErrorEventFromRaw(code, "invalid_request_error", msg)
		require.True(t, matched, "code=%q 必须被识别为密文失效", code)
		require.Equal(t, "invalid_encrypted_content", reason)
		require.True(t, isOpenAIInvalidEncryptedContentUpstreamError(code, msg))
	}

	reason, matched := classifyOpenAIWSErrorEventFromRaw("invalid_request_error", "invalid_request_error", "Invalid signature")
	require.False(t, matched || reason == "invalid_encrypted_content", "无关签名错误不得归类为密文失效")
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentDropsAssistantHistoryAsLastResort
// 最后一道兜底：前三级自愈都做不了时，丢弃 assistant 侧历史，保留 user 消息与工具输出。
// 对话卡死（客户端反复重放同一份历史）比丢一段 assistant 历史更糟。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentDropsAssistantHistoryAsLastResort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverSideBody := `{"error":{"code":"invalid_encrypted_content","message":"The encrypted content for item amsg_zzz could not be verified. Reason: encrypted content hydration failed: Encrypted content could not be decrypted or parsed.","param":"input","type":"invalid_request_error"}}`
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(serverSideBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{ID: 6854, Name: "oauth-codex", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Concurrency: 1, Credentials: map[string]any{
			"access_token":       "test-token",
			"chatgpt_account_id": "test-account",
		}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := `{"model":"gpt-5.1","stream":false,"input":[{"type":"reasoning","id":"rs_1","summary":[{"type":"summary_text","text":"old"}]},{"type":"message","id":"amsg_zzz","role":"assistant","content":[{"type":"output_text","text":"old answer"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]},{"type":"function_call_output","call_id":"call_1","output":"result"}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2, "最后一道兜底也必须重试一次")
	require.Equal(t, "reasoning", gjson.GetBytes(upstream.bodies[0], "input.0.type").String(), "首次请求应保留原历史")
	// 重试后：reasoning 与 assistant message 被丢弃，user 消息与工具输出保留
	types := []string{}
	gjson.GetBytes(upstream.bodies[1], "input").ForEach(func(_, v gjson.Result) bool {
		types = append(types, v.Get("type").String())
		return true
	})
	require.NotContains(t, types, "reasoning", "兜底必须丢弃 reasoning 历史")
	// codex transform 本身就会把 function_call_output 规范化掉（首次请求体里已不存在），
	// 因此这里只断言 user 消息被保留、assistant 侧历史被丢弃。
	require.Equal(t, []string{"message"}, types, "只应保留 user 消息")
	require.Equal(t, "hello", gjson.GetBytes(upstream.bodies[1], "input.0.content.0.text").String(), "user 消息必须保留")
	require.Equal(t, "user", gjson.GetBytes(upstream.bodies[1], "input.0.role").String())
}

// TestStripOpenAIAssistantHistoryForInvalidEncryptedContent_NoAssistantHistoryIsNoop
// 没有 assistant 侧历史时不应改动请求体。
func TestStripOpenAIAssistantHistoryForInvalidEncryptedContent_NoAssistantHistoryIsNoop(t *testing.T) {
	body := map[string]any{
		"input": []any{
			map[string]any{"type": "message", "role": "user", "content": "hi"},
			map[string]any{"type": "function_call_output", "call_id": "c1", "output": "r"},
		},
	}
	require.False(t, stripOpenAIAssistantHistoryForInvalidEncryptedContent(body))
	require.Len(t, body["input"], 2)
}

// TestOpenAIGatewayService_Forward_InvalidEncryptedContentTrimsToLastUserMessage
// 终极兜底：连 assistant 侧历史都识别不出来时（客户端回放的 item 不带 role），
// 把 input 裁剪成只保留最后一条 user 消息，保证请求发得出去。
func TestOpenAIGatewayService_Forward_InvalidEncryptedContentTrimsToLastUserMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverSideBody := `{"error":{"code":"invalid_encrypted_content","message":"The encrypted content for item amsg_x could not be verified. Reason: encrypted content hydration failed: Encrypted content could not be decrypted or parsed.","param":"input","type":"invalid_request_error"}}`
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(serverSideBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_recovered","usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{ID: 6642, Name: "apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Concurrency: 1, Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.com",
		},
		Extra: map[string]any{"use_responses_api": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	// item 不带 role：删 assistant 侧历史识别不出来，只能走终极兜底。
	body := `{"model":"gpt-5.1","stream":false,"input":[{"type":"message","content":[{"type":"output_text","text":"old"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2, "终极兜底也必须重试一次")
	require.Equal(t, "hello", gjson.GetBytes(upstream.bodies[1], "input.0.content.0.text").String(), "只保留最后一条 user 消息")
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.1").Exists(), "其它历史必须被裁掉")
}
