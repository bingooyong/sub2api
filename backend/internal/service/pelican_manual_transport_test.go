//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func manualPelicanAccount(kind string, chat bool) *Account {
	a := &Account{ID: 1, Platform: PlatformOpenAI, Type: kind, Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token", "api_key": "test-key", "base_url": "https://example.test"},
		Extra:       map[string]any{"unchanged": true}, Schedulable: true, Status: StatusActive, GroupIDs: []int64{5}}
	if chat {
		a.Extra[openai_compat.ExtraKeyResponsesSupported] = false
	}
	return a
}

func manualPelicanService(a *Account, responses ...*http.Response) (*AccountTestService, *openAIAccountTestRepo, *queuedHTTPUpstream) {
	repo := &openAIAccountTestRepo{mockAccountRepoForGemini: mockAccountRepoForGemini{accountsByID: map[int64]*Account{a.ID: a}}}
	u := &queuedHTTPUpstream{responses: responses}
	return &AccountTestService{accountRepo: repo, httpUpstream: u, cfg: &config.Config{}}, repo, u
}

func assertPelicanObservation(t *testing.T, a *Account, repo *openAIAccountTestRepo) {
	t.Helper()
	require.Empty(t, repo.updatedExtra)
	require.Empty(t, repo.bulkUpdatedIDs)
	require.Zero(t, repo.rateLimitedID)
	require.Zero(t, repo.clearedErrorID)
	require.Zero(t, repo.setErrorID)
	require.Equal(t, map[string]any{"unchanged": true}, a.Extra, "probe headers must not mutate the in-memory snapshot either")
	require.True(t, a.Schedulable)
	require.Equal(t, StatusActive, a.Status)
	require.Equal(t, []int64{5}, a.GroupIDs)
}

func TestManualPelicanActualTransportUsesSharedPromptAndEffort(t *testing.T) {
	for _, tc := range []struct {
		name, kind, path, promptPath, effortPath string
		chat                                     bool
	}{
		{"oauth_responses", AccountTypeOAuth, "/backend-api/codex/responses", "input.0.content.0.text", "reasoning.effort", false},
		{"apikey_responses", AccountTypeAPIKey, "/v1/responses", "input.0.content.0.text", "reasoning.effort", false},
		{"apikey_chat", AccountTypeAPIKey, "/v1/chat/completions", "messages.0.content", "reasoning_effort", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := manualPelicanAccount(tc.kind, tc.chat)
			a.Credentials["model_mapping"] = map[string]any{"comparison": "gpt-5.4"}
			stream := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"<svg></svg>\"}\n\ndata: {\"type\":\"response.completed\"}\n\n"
			if tc.chat {
				stream = "data: {\"choices\":[{\"delta\":{\"content\":\"<svg></svg>\"}}]}\n\ndata: [DONE]\n\n"
			}
			resp := newJSONResponse(http.StatusOK, stream)
			resp.Header.Set("x-codex-primary-used-percent", "42")
			resp.Header.Set("x-codex-primary-window-minutes", "300")
			resp.Header.Set("x-codex-primary-reset-after-seconds", "60")
			svc, repo, upstream := manualPelicanService(a, resp)
			ctx, recorder := newTestContext()
			err := svc.TestManualPelicanAccountConnection(ctx, 1, "comparison", "Draw a pelican cycling in SVG.", "high")
			require.NoError(t, err)
			require.Len(t, upstream.requests, 1)
			req := upstream.requests[0]
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, tc.path, req.URL.Path)
			require.Equal(t, "gpt-5.4", gjson.GetBytes(body, "model").String())
			require.Equal(t, "Draw a pelican cycling in SVG.", gjson.GetBytes(body, tc.promptPath).String())
			require.Equal(t, "high", gjson.GetBytes(body, tc.effortPath).String())
			require.NotContains(t, strings.ToLower(string(body)), "sub2api")
			for key, values := range req.Header {
				require.NotContains(t, strings.ToLower(key+strings.Join(values, " ")), "sub2api")
			}
			require.Contains(t, recorder.Body.String(), `"success":true`)
			if tc.chat {
				delete(a.Extra, openai_compat.ExtraKeyResponsesSupported)
			}
			assertPelicanObservation(t, a, repo)
		})
	}
}

func TestManualPelicanErrorsDoNotMutateAccount(t *testing.T) {
	for _, chat := range []bool{false, true} {
		for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests} {
			t.Run(http.StatusText(status)+map[bool]string{true: "_chat", false: "_responses"}[chat], func(t *testing.T) {
				a := manualPelicanAccount(AccountTypeOAuth, false)
				if chat {
					a = manualPelicanAccount(AccountTypeAPIKey, true)
				}
				resp := newJSONResponse(status, `{"error":{"message":"usage_limit_reached","resets_in_seconds":60}}`)
				resp.Header.Set("x-codex-primary-used-percent", "100")
				resp.Header.Set("x-codex-primary-window-minutes", "300")
				resp.Header.Set("x-codex-primary-reset-after-seconds", "60")
				resp.Header.Set("retry-after", "60")
				svc, repo, _ := manualPelicanService(a, resp)
				ctx, recorder := newTestContext()
				require.Error(t, svc.TestManualPelicanAccountConnection(ctx, 1, "gpt-5.4", "Question", "medium"))
				require.Contains(t, recorder.Body.String(), `"type":"error"`)
				if chat {
					delete(a.Extra, openai_compat.ExtraKeyResponsesSupported)
				}
				assertPelicanObservation(t, a, repo)
			})
		}
	}
}

type cancellingPelicanUpstream struct {
	queuedHTTPUpstream
	started chan struct{}
}

func (u *cancellingPelicanUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	close(u.started)
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestManualPelicanCancellationReachesUpstreamWithoutMutation(t *testing.T) {
	a := manualPelicanAccount(AccountTypeOAuth, false)
	svc, repo, _ := manualPelicanService(a)
	u := &cancellingPelicanUpstream{started: make(chan struct{})}
	svc.httpUpstream = u
	ctx, _ := newTestContext()
	requestCtx, cancel := context.WithCancel(ctx.Request.Context())
	defer cancel()
	ctx.Request = ctx.Request.WithContext(requestCtx)
	done := make(chan error, 1)
	go func() { done <- svc.TestManualPelicanAccountConnection(ctx, 1, "gpt-5.4", "Question", "high") }()
	select {
	case <-u.started:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorContains(t, err, context.Canceled.Error())
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled probe did not return")
	}
	assertPelicanObservation(t, a, repo)
}

func TestManualPelicanRejectsUnsupportedProviderAndMappedMedia(t *testing.T) {
	for _, tc := range []struct{ name, platform, target string }{
		{"provider", PlatformAnthropic, "gpt-5.4"},
		{"mapped_image", PlatformOpenAI, "gpt-image-2"},
		{"mapped_audio", PlatformOpenAI, "gpt-audio"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := manualPelicanAccount(AccountTypeAPIKey, false)
			a.Platform = tc.platform
			a.Credentials["model_mapping"] = map[string]any{"comparison": tc.target}
			svc, _, upstream := manualPelicanService(a)
			ctx, _ := newTestContext()
			require.Error(t, svc.TestManualPelicanAccountConnection(ctx, 1, "comparison", "Question", "medium"))
			require.Empty(t, upstream.requests)
		})
	}
}

func TestPelicanTransportKeepsOrdinaryConnectivityDefaultAndUserText(t *testing.T) {
	a := manualPelicanAccount(AccountTypeOAuth, false)
	for _, manual := range []bool{false, true} {
		svc, _, upstream := manualPelicanService(a, newJSONResponse(http.StatusOK, "data: {\"type\":\"response.completed\"}\n\n"))
		ctx, _ := newTestContext()
		var err error
		if manual {
			err = svc.TestManualPelicanAccountConnection(ctx, 1, "gpt-5.4", "Explain the user project sub2api", "high")
		} else {
			err = svc.TestAccountConnection(ctx, 1, "gpt-5.4", "unused connectivity prompt", "")
		}
		require.NoError(t, err)
		body, err := io.ReadAll(upstream.requests[0].Body)
		require.NoError(t, err)
		if manual {
			require.Equal(t, "Explain the user project sub2api", gjson.GetBytes(body, "input.0.content.0.text").String())
		} else {
			require.Equal(t, "hi", gjson.GetBytes(body, "input.0.content.0.text").String())
			require.False(t, gjson.GetBytes(body, "reasoning").Exists())
		}
	}
}

func TestManualPelicanValidationBeforeAccountLookup(t *testing.T) {
	for _, tc := range []struct{ model, prompt, effort string }{
		{"", "Question", "high"}, {"gpt-5.4", " \n ", "high"}, {"gpt-5.4", "Question", "invalid"},
		{strings.Repeat("a", 101), "Question", "high"}, {"gpt-5.4", strings.Repeat("题", 32001), "high"},
	} {
		ctx, _ := newTestContext()
		svc := &AccountTestService{}
		err := svc.TestManualPelicanAccountConnection(ctx, 1, tc.model, tc.prompt, tc.effort)
		require.Error(t, err)
	}
}

func TestManualPelicanTransportAndTruncatedStreamFailuresRemainObservational(t *testing.T) {
	for _, tc := range []struct {
		name      string
		responses []*http.Response
	}{
		{"transport_failure", nil},
		{"truncated_stream", []*http.Response{newJSONResponse(http.StatusOK, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := manualPelicanAccount(AccountTypeOAuth, false)
			svc, repo, _ := manualPelicanService(a, tc.responses...)
			ctx, recorder := newTestContext()
			require.Error(t, svc.TestManualPelicanAccountConnection(ctx, 1, "gpt-5.4", "Question", "medium"))
			require.Contains(t, recorder.Body.String(), `"type":"error"`)
			assertPelicanObservation(t, a, repo)
		})
	}
}

func TestScheduledPelicanActualTransportHonorsOptionsWithoutHealthWrites(t *testing.T) {
	a := manualPelicanAccount(AccountTypeOAuth, false)
	resp := newJSONResponse(http.StatusOK, "data: {\"type\":\"response.completed\"}\n\n")
	resp.Header.Set("x-codex-primary-used-percent", "60")
	resp.Header.Set("x-codex-primary-window-minutes", "300")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "60")
	svc, repo, upstream := manualPelicanService(a, resp)
	ctx, _ := newTestContext()
	require.NoError(t, svc.TestPelicanAccountConnection(ctx, 1, "gpt-5.4", "Scheduled question", "high"))
	body, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "Scheduled question", gjson.GetBytes(body, "input.0.content.0.text").String())
	require.Equal(t, "high", gjson.GetBytes(body, "reasoning.effort").String())
	assertPelicanObservation(t, a, repo)
}
