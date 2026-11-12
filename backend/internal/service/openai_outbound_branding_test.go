package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Capture the final request sent by Forward, including compatibility-generated
// instructions. Internal hash namespaces are deliberately not renamed.
func TestOpenAIOutboundBranding_HTTPGeneratedMetadataAndCallerContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })
	for _, tc := range []struct {
		name   string
		model  string
		raw    bool
		bridge bool
	}{
		{name: "responses image bridge", model: "gpt-5.2", bridge: true},
		{name: "responses spark instructions", model: "gpt-5.3-codex-spark"},
		{name: "raw responses", model: "gpt-5.2", raw: true},
		{name: "messages bridge", model: "gpt-5.4"},
	} {
		for _, prompt := range []string{"Draw a pelican riding a bicycle.", "Review the sub2api source."} {
			t.Run(tc.name+"/"+prompt, func(t *testing.T) {
				body, err := json.Marshal(map[string]any{
					"model": tc.model, "stream": false,
					"input": []map[string]any{{"type": "message", "role": "user", "content": prompt}},
				})
				require.NoError(t, err)
				path := "/v1/responses"
				if tc.name == "messages bridge" {
					path = "/v1/messages"
					body, err = json.Marshal(map[string]any{
						"model": "claude-sonnet-4-5", "max_tokens": 128,
						"messages": []map[string]any{{"role": "user", "content": prompt}},
					})
					require.NoError(t, err)
				}
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
				c.Request.Header.Set("User-Agent", codexCLIUserAgent)
				c.Request.Header.Set("Originator", "codex_cli_rs")
				c.Request.Header.Set("X-Sub2api-Admin-Ui", "1")
				upstream := &httpUpstreamRecorder{resp: openAICompatSSECompletedResponse("resp_branding", tc.model)}
				cfg := &config.Config{}
				cfg.Gateway.CodexImageGenerationBridgeEnabled = tc.bridge
				svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream, toolCorrector: NewCodexToolCorrector()}
				account := identityHeaderTestAccount()
				account.Extra = map[string]any{"openai_oauth_passthrough": tc.raw}
				account.Status, account.Schedulable, account.Concurrency = StatusActive, true, 1
				if tc.name == "messages bridge" {
					_, err = svc.ForwardAsAnthropic(context.Background(), c, account, body, "", tc.model)
				} else {
					_, err = svc.Forward(context.Background(), c, account, body)
				}
				require.NoError(t, err)
				require.NotNil(t, upstream.lastReq)
				var headers bytes.Buffer
				require.NoError(t, upstream.lastReq.Header.Write(&headers))
				require.NotContains(t, strings.ToLower(headers.String()), "sub2api", "local diagnostic headers must not reach GPT")
				require.Contains(t, string(upstream.lastBody), prompt, "caller-authored content must remain intact")
				if !strings.Contains(prompt, "sub2api") {
					require.NotContains(t, strings.ToLower(string(upstream.lastBody)), "sub2api", "generated prompts and metadata must remain unbranded")
				}
				if tc.bridge {
					require.Contains(t, string(upstream.lastBody), "codex-image-generation-bridge", "exercise the generated instruction branch")
				}
				if tc.model == "gpt-5.3-codex-spark" {
					require.Contains(t, string(upstream.lastBody), "codex-spark-image-unsupported", "exercise the generated Spark instruction branch")
				}
			})
		}
	}
}

func TestOpenAIOutboundBranding_WSHandshakeAndActualFrames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	upstream := newStagedPassthroughConn()
	cfg := passthroughLifecycleConfig()
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 10
	cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 10
	account := passthroughLifecycleAccount()
	account.Type = AccountTypeOAuth
	account.Credentials = map[string]any{"access_token": "test-token", "chatgpt_account_id": "test-account"}
	account.Extra = map[string]any{
		"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough,
		codexFingerprintModeExtraKey:                "full",
		codexFingerprintSeedExtraKey:                "11111111-1111-4111-8111-111111111111",
	}
	svc := newPassthroughLifecycleService(cfg, upstream)
	dialer := &fingerprintPassthroughDialer{conn: upstream, headers: make(chan http.Header, 1)}
	svc.openaiWSPassthroughDialer = dialer
	server, serverErr := startPassthroughLifecycleServer(t, ctx, svc, account)
	defer server.Close()
	headers := http.Header{
		"User-Agent": {"codex_exec/0.153.4"}, "Originator": {"codex_exec"},
		"Session-Id": {"caller-session"}, "X-Sub2api-Admin-Ui": {"1"},
	}
	client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), &coderws.DialOptions{HTTPHeader: headers})
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()
	for i, prompt := range []string{"Draw a pelican riding a bicycle.", "Review the sub2api source."} {
		body, err := json.Marshal(map[string]any{
			"type": "response.create", "model": "gpt-5.2",
			"input":           []map[string]any{{"type": "message", "role": "user", "content": prompt}},
			"client_metadata": map[string]any{"session_id": fmt.Sprintf("body-session-%d", i)},
		})
		require.NoError(t, err)
		require.NoError(t, client.Write(ctx, coderws.MessageText, body))
		sent := requirePassthroughUpstreamWrite(t, upstream, 3*time.Second)
		if i == 0 {
			var handshake bytes.Buffer
			headers := <-dialer.headers
			require.NoError(t, headers.Write(&handshake))
			require.NotContains(t, strings.ToLower(handshake.String()), "sub2api")
			require.Equal(t, "codex_exec/0.153.4", headers.Get("User-Agent"))
			require.Equal(t, "codex_exec", headers.Get("Originator"))
			require.Empty(t, headers.Values("Version"))
			require.NotEqual(t, "caller-session", headers.Get("session-id"), "exercise stable hash-derived fingerprint output")
		}
		require.Contains(t, string(sent), prompt)
		if i == 0 {
			require.NotContains(t, strings.ToLower(string(sent)), "sub2api")
		}
		upstream.Send(fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_branding_%d","model":"gpt-5.2","usage":{"input_tokens":1,"output_tokens":1}}}`, i))
		_, err = readPassthroughLifecycleFrame(t, client, 3*time.Second)
		require.NoError(t, err)
	}
	require.NoError(t, client.CloseNow())
	select {
	case shutdownErr := <-serverErr:
		t.Logf("passthrough shutdown result: %v", shutdownErr)
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough did not stop")
	}
}
