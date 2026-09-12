package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// These tests change the process-wide enforcement setting and must remain serial.
func TestOpenAIGatewayService_IdentityOptOutPreservesClientHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })

	for _, tc := range []struct {
		name       string
		userAgent  string
		originator string
		version    string
	}{
		{
			name:       "exec without version header",
			userAgent:  "codex_exec/0.153.4 (Mac OS 26.6.2; arm64) iTerm.app/3.4.22 (codex_exec; 0.153.4)",
			originator: "codex_exec",
		},
		{
			name:       "old client version is not elevated",
			userAgent:  "codex-tui/0.125.0 (Linux; aarch64) terminal",
			originator: "codex-tui",
			version:    "0.125.0",
		},
		{
			name:       "prerelease version",
			userAgent:  "codex-tui/0.154.0-alpha.2 (Mac OS 26.6.2; arm64) terminal",
			originator: "codex-tui",
			version:    "0.154.0-alpha.2",
		},
		{
			name:       "custom originator prefix",
			userAgent:  "workbench/0.153.4 (Linux; x86_64) screen (codex-tui; 0.153.4)",
			originator: "workbench",
			version:    "0.153.4",
		},
		{
			name:       "independently declared version",
			userAgent:  "Codex Desktop/1.2.3 (Mac OS 26.6.2; arm64) unknown",
			originator: "Codex Desktop",
			version:    "0.153.4",
		},
		{
			name:       "unknown client metadata",
			userAgent:  "warehouse-adapter/2.0",
			originator: "warehouse",
			version:    "build-2026_09",
		},
		{name: "all identity headers absent"},
		{name: "only user agent", userAgent: "warehouse-adapter/2.0"},
		{name: "only originator", originator: "workbench"},
		{name: "only version", version: "0.153.4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := make(http.Header)
			for name, value := range map[string]string{
				"User-Agent": tc.userAgent, "Originator": tc.originator, "Version": tc.version,
			} {
				if value != "" {
					input.Set(name, value)
				}
			}
			for _, path := range []string{"http", "compact", "passthrough", "passthrough compact", "websocket", "alpha search", "alpha responses", "messages"} {
				t.Run(path, func(t *testing.T) {
					account := identityHeaderTestAccount()
					headers := buildIdentityTestHeaders(t, &OpenAIGatewayService{}, account, path, input)
					for _, name := range []string{"User-Agent", "Originator", "Version"} {
						require.Equal(t, input.Values(name), headers.Values(name), "%s must retain caller metadata", name)
					}
					require.Equal(t, "Bearer test-token", headers.Get("Authorization"))
				})
			}
		})
	}
}

func TestOpenAIGatewayService_IdentityOptOutDiscardsMalformedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })

	for _, name := range []string{"User-Agent", "Originator", "Version"} {
		t.Run(name, func(t *testing.T) {
			for _, invalid := range []struct {
				name  string
				value string
			}{
				{name: "CRLF", value: "valid\r\nInjected: value"},
				{name: "NUL", value: "\x00invalid"},
				{name: "DEL", value: "valid\x7f"},
			} {
				for _, path := range []string{"http", "compact", "passthrough", "passthrough compact", "websocket", "alpha search", "alpha responses", "messages"} {
					t.Run(path+"/"+invalid.name, func(t *testing.T) {
						input := make(http.Header)
						input.Set(name, invalid.value)
						headers := buildIdentityTestHeaders(t, &OpenAIGatewayService{}, identityHeaderTestAccount(), path, input)
						for _, identityName := range []string{"User-Agent", "Originator", "Version"} {
							require.Empty(t, headers.Values(identityName), "invalid or absent %s must not gain a fabricated value", identityName)
						}
					})
				}
			}
		})
	}
}

func TestOpenAIGatewayService_IdentityOptOutDoesNotChangeAPIKeyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })
	input := http.Header{"User-Agent": {"workbench/1.0"}, "Originator": {"workbench"}, "Version": {"0.153.4"}}
	for _, path := range []string{"http", "compact", "passthrough", "passthrough compact", "websocket", "alpha search"} {
		t.Run(path, func(t *testing.T) {
			account := identityHeaderTestAccount()
			account.Type = AccountTypeAPIKey
			SetCodexIdentityEnforcementEnabled(true)
			legacy := buildIdentityTestHeaders(t, &OpenAIGatewayService{}, account, path, input)
			SetCodexIdentityEnforcementEnabled(false)
			optOut := buildIdentityTestHeaders(t, &OpenAIGatewayService{}, account, path, input)
			require.Equal(t, legacy, optOut, "the Codex OAuth opt-out must not change API-key header policy")
		})
	}
}

func TestOpenAIGatewayService_IdentityOptOutPreservesSyntheticHeaders(t *testing.T) {
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })
	for _, tc := range []struct {
		name    string
		context *gin.Context
	}{
		{name: "no context"},
		{name: "no incoming request", context: &gin.Context{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			headers := make(http.Header)
			ensureCodexIdentityHeaders(headers)
			expected := headers.Clone()
			svc := &OpenAIGatewayService{}
			svc.preserveCodexClientIdentityHeaders(tc.context, identityHeaderTestAccount(), headers)
			require.Equal(t, expected, headers)
		})
	}
}

func TestOpenAIGatewayService_IdentityOptOutKeepsBridgeOriginatorOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })

	input := http.Header{"User-Agent": {"workbench/1.0"}, "Originator": {"workbench"}, "Version": {"2.0"}}
	headers := buildIdentityTestHeaders(t, &OpenAIGatewayService{}, identityHeaderTestAccount(), "http bridge", input)
	require.Equal(t, input.Values("User-Agent"), headers.Values("User-Agent"))
	require.Equal(t, input.Values("Version"), headers.Values("Version"))
	require.Empty(t, headers.Values("Originator"), "the inner bridge builder deliberately omits originator")
}

func TestOpenAIGatewayService_IdentityOptOutKeepsExplicitUserAgentOverrides(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := codexIdentityEnforcement.Load()
	SetCodexIdentityEnforcementEnabled(false)
	t.Cleanup(func() { SetCodexIdentityEnforcementEnabled(previous) })

	const accountUA = "codex_vscode/0.149.2 (Mac OS 26.6.2; arm64) vscode"
	for _, force := range []bool{false, true} {
		name := "account user agent"
		if force {
			name = "force CLI takes precedence"
		}
		t.Run(name, func(t *testing.T) {
			for _, path := range []string{"http", "compact", "passthrough", "passthrough compact", "websocket", "alpha search", "alpha responses", "messages"} {
				t.Run(path, func(t *testing.T) {
					cfg := &config.Config{}
					cfg.Gateway.ForceCodexCLI = force
					account := identityHeaderTestAccount()
					account.Credentials["user_agent"] = accountUA
					input := http.Header{"User-Agent": {"workbench/1.0"}, "Originator": {"workbench"}, "Version": {"2.0"}}
					headers := buildIdentityTestHeaders(t, &OpenAIGatewayService{cfg: cfg}, account, path, input)
					wantUA, wantOriginator := accountUA, "codex_vscode"
					if force {
						wantUA, wantOriginator = codexCLIUserAgent, "codex-tui"
					}
					require.Equal(t, wantUA, headers.Get("User-Agent"))
					require.Equal(t, wantOriginator, headers.Get("Originator"))
					wantVersion := ""
					switch path {
					case "compact", "passthrough compact", "messages":
						wantVersion = codexCLIVersion
					case "alpha search", "alpha responses":
						wantVersion = "2.0"
					}
					require.Equal(t, wantVersion, headers.Get("Version"), "explicit overrides retain their legacy version policy")
				})
			}
		})
	}
}

func identityHeaderTestAccount() *Account {
	return &Account{
		ID:       51,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "test-token",
			"chatgpt_account_id": "test-account",
		},
	}
}

func buildIdentityTestHeaders(t *testing.T, svc *OpenAIGatewayService, account *Account, path string, input http.Header) http.Header {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	requestPath := "/v1/responses"
	if path == "compact" || path == "passthrough compact" {
		requestPath += "/compact"
	}
	c.Request = httptest.NewRequest(http.MethodPost, requestPath, nil)
	c.Request.Header = input.Clone()
	if path == "http bridge" {
		setOpenAICompatMessagesBridgeContext(c, true)
	}
	body := []byte(`{"model":"gpt-5","input":"hello"}`)
	if path == "websocket" {
		headers, _, err := svc.buildOpenAIWSHeaders(
			context.Background(), c, account, "test-token",
			OpenAIWSProtocolDecision{Transport: OpenAIUpstreamTransportResponsesWebsocketV2},
			true, "", "", "", "", "",
		)
		require.NoError(t, err)
		return headers
	}
	var req *http.Request
	var err error
	switch path {
	case "passthrough", "passthrough compact":
		req, err = svc.buildUpstreamRequestOpenAIPassthrough(context.Background(), c, account, body, "test-token")
	case "alpha search":
		req, err = svc.buildOpenAIAlphaSearchRequest(context.Background(), c, account, []byte(`{"query":"hello"}`), "test-token")
	case "alpha responses":
		req, err = svc.buildOpenAIAlphaSearchResponsesWebSearchRequest(context.Background(), c, account, []byte(`{"query":"hello"}`), body, "test-token")
	case "messages":
		body = []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
		c.Request.Header = input.Clone()
		upstream := &httpUpstreamRecorder{resp: openAICompatSSECompletedResponse("resp_identity", "gpt-5.4")}
		svc.httpUpstream = upstream
		if svc.cfg == nil {
			svc.cfg = &config.Config{}
		}
		result, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "gpt-5.4")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, upstream.lastReq)
		return upstream.lastReq.Header
	default:
		req, err = svc.buildUpstreamRequest(context.Background(), c, account, body, "test-token", true, "", true)
	}
	require.NoError(t, err)
	return req.Header
}
