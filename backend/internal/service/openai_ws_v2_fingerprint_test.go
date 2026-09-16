package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type fingerprintPassthroughDialer struct {
	conn    *stagedPassthroughConn
	headers chan http.Header
}

func (d *fingerprintPassthroughDialer) Dial(_ context.Context, _ string, headers http.Header, _ string) (openAIWSClientConn, int, http.Header, error) {
	d.headers <- headers.Clone()
	return d.conn, http.StatusSwitchingProtocols, http.Header{}, nil
}

// Exercise native ingress, the real v2 adapter, and its subsequent-frame filter.
// Inspect actual writes rather than testing the fingerprint helper in isolation.
func TestPassthroughFingerprint_HeadersAndEveryFrame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		mode   codexFingerprintMode
		apiKey bool
	}{
		{"off", codexFingerprintOff, false},
		{"device", codexFingerprintDevice, false},
		{"session", codexFingerprintSession, false},
		{"full", codexFingerprintFull, false},
		{"api_key", codexFingerprintFull, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			upstream := newStagedPassthroughConn()
			cfg := passthroughLifecycleConfig()
			cfg.Gateway.OpenAIWS.OAuthEnabled = true
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 10
			cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 10
			account := passthroughLifecycleAccount()
			if !tc.apiKey {
				account.Type = AccountTypeOAuth
				account.Credentials = map[string]any{"access_token": "test-token", "chatgpt_account_id": "fingerprint-account"}
				account.Extra["openai_oauth_responses_websockets_v2_mode"] = OpenAIWSIngressModePassthrough
			}
			account.Extra[codexFingerprintModeExtraKey] = string(tc.mode)
			account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			svc := newPassthroughLifecycleService(cfg, upstream)
			dialer := &fingerprintPassthroughDialer{conn: upstream, headers: make(chan http.Header, 1)}
			svc.openaiWSPassthroughDialer = dialer
			// Seed stale state as if the handler had already tried another account.
			// Native ingress must replace it, including when the selected mode is off.
			server, serverErr := startPassthroughLifecycleServerWithHooks(t, ctx, svc, account, func(c *gin.Context) *OpenAIWSIngressHooks {
				stageCodexFingerprintIDs(c, &codexFingerprintIDs{accountID: account.ID, mode: codexFingerprintFull, installationID: "stale-device", sessionID: "stale-session"})
				return nil
			})
			defer server.Close()
			headers := http.Header{}
			headers.Set("session-id", "caller-session")
			headers.Set("thread-id", "caller-thread")
			headers.Set("x-codex-installation-id", "caller-installation")
			headers.Set(openAIWSTurnMetadataHeader, `{"session_id":"caller-session","turn_id":"caller-turn","installation_id":"caller-installation"}`)
			client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), &coderws.DialOptions{HTTPHeader: headers})
			require.NoError(t, err)
			defer func() { _ = client.CloseNow() }()
			var handshake http.Header
			for i, frame := range []struct{ event, session, cache string }{
				{"response.create", "body-session-1", "body-session-1"},
				{"response.create", "body-session-2", "body-session-2"},
				{"response.create", "body-session-3", "custom-cache"},
				{"session.update", "body-session-4", "body-session-4"},
			} {
				raw := fmt.Sprintf(`{"type":%q,"model":"gpt-5.1","input":[],"prompt_cache_key":%q,"client_metadata":{"session_id":%q,"thread_id":"caller-thread","x-codex-installation-id":"caller-installation","x-codex-turn-metadata":"{\"session_id\":\"%s\",\"turn_id\":\"caller-turn\",\"installation_id\":\"caller-installation\"}"}}`, frame.event, frame.cache, frame.session, frame.session)
				require.NoError(t, client.Write(ctx, coderws.MessageText, []byte(raw)))
				sent := requirePassthroughUpstreamWrite(t, upstream, 3*time.Second)
				if i == 0 {
					handshake = <-dialer.headers
				}
				installation := gjson.GetBytes(sent, "client_metadata.x-codex-installation-id").String()
				require.Equal(t, handshake.Get("x-codex-installation-id"), installation)
				require.NotEqual(t, "stale-device", installation)
				session := gjson.GetBytes(sent, "client_metadata.session_id").String()
				cache := gjson.GetBytes(sent, "prompt_cache_key").String()
				converged := !tc.apiKey && (tc.mode == codexFingerprintSession || tc.mode == codexFingerprintFull)
				if converged {
					require.Equal(t, handshake.Get("session-id"), session)
					require.Equal(t, handshake.Get("thread-id"), gjson.GetBytes(sent, "client_metadata.thread_id").String())
					embedded := gjson.GetBytes(sent, "client_metadata.x-codex-turn-metadata").String()
					require.Equal(t, session, gjson.Get(embedded, "session_id").String())
					require.Equal(t, gjson.Get(handshake.Get(openAIWSTurnMetadataHeader), "turn_id").String(), gjson.GetBytes(sent, "client_metadata.turn_id").String())
				} else {
					require.Equal(t, scopeCodexAccountIdentityValue(account, 0, "session", frame.session), session)
				}
				if frame.cache == frame.session {
					require.Equal(t, session, cache)
				} else {
					require.Equal(t, scopeCodexAccountIdentityValue(account, 0, "prompt-cache", frame.cache), cache)
				}
				if frame.event == "response.create" {
					upstream.Send(fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_%d","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`, i))
					_, err := readPassthroughLifecycleFrame(t, client, 3*time.Second)
					require.NoError(t, err)
				}
			}
			require.NoError(t, client.CloseNow())
			select {
			case <-serverErr:
			case <-time.After(3 * time.Second):
				t.Fatal("passthrough did not stop")
			}
		})
	}
}

func TestWSFingerprint_FrameSnapshotAndAccountGuard(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{
		codexFingerprintModeExtraKey: string(codexFingerprintSession),
		codexFingerprintSeedExtraKey: "11111111-1111-4111-8111-111111111111",
	}}
	ids := resolveCodexFingerprintIDs(account, "caller-session", codexFingerprintSession)
	require.NotNil(t, ids)
	// Even if an earlier path captured the first body's session, a new frame
	// must match its own body session without changing the handshake snapshot.
	ids.originalBodySessionID = "first-body-session"
	ids.originalBodySessionIDCaptured = true
	before := *ids
	stageCodexFingerprintIDs(c, ids)
	body := []byte(`{"client_metadata":{"session_id":"next-body-session"},"prompt_cache_key":"next-body-session"}`)
	out, changed, err := applyOpenAIWSFingerprintClientMetadata(c, account, body)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, ids.sessionID, gjson.GetBytes(out, "prompt_cache_key").String())
	require.Equal(t, before, *ids)
	for _, other := range []*Account{nil, {ID: 43, Platform: PlatformOpenAI, Type: AccountTypeOAuth}, {ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}} {
		out, changed, err = applyOpenAIWSFingerprintClientMetadata(c, other, body)
		require.NoError(t, err)
		require.False(t, changed)
		require.Equal(t, body, out)
	}
	stageCodexFingerprintIDs(c, nil)
	out, changed, err = applyOpenAIWSFingerprintClientMetadata(c, account, body)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, body, out)
}
