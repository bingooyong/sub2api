package admin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestPelicanHandlerRejectsInvalidRequestBeforeDispatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct{ name, id, body string }{
		{"negative_id", "-1", `{"model_id":"gpt-5.4","prompt":"Question"}`},
		{"invalid_id", "bad", `{"model_id":"gpt-5.4","prompt":"Question"}`},
		{"missing_model", "1", `{"prompt":"Question"}`},
		{"blank_prompt", "1", `{"model_id":"gpt-5.4","prompt":"  "}`},
		{"invalid_effort", "1", `{"model_id":"gpt-5.4","prompt":"Question","reasoning_effort":"unsupported"}`},
		{"unsupported_mode", "1", `{"model_id":"gpt-5.4","prompt":"Question","mode":"compact"}`},
		{"long_model", "1", `{"model_id":"` + strings.Repeat("x", 101) + `","prompt":"Question"}`},
		{"long_prompt", "1", `{"model_id":"gpt-5.4","prompt":"` + strings.Repeat("a", 32001) + `"}`},
		{"oversized_body", "1", `{"model_id":"gpt-5.4","prompt":"` + strings.Repeat("a", 256<<10) + `"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Params = gin.Params{{Key: "id", Value: tc.id}}
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			// No service dependency: invalid input must stop before dispatch.
			(&AccountHandler{}).PelicanTest(ctx)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.NotContains(t, recorder.Body.String(), `"type":"test_start"`)
		})
	}
}

type pelicanHandlerRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r *pelicanHandlerRepo) GetByID(_ context.Context, _ int64) (*service.Account, error) {
	return r.account, nil
}

type pelicanHandlerUpstream struct {
	service.HTTPUpstream
	body    []byte
	headers http.Header
}

func (u *pelicanHandlerUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	var err error
	u.body, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	u.headers = req.Header.Clone()
	return &http.Response{StatusCode: http.StatusOK,
		Header: http.Header{
			"X-Codex-Primary-Used-Percent":        {"45"},
			"X-Codex-Primary-Window-Minutes":      {"300"},
			"X-Codex-Primary-Reset-After-Seconds": {"60"},
		},
		Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"<svg></svg>\"}\n\ndata: {\"type\":\"response.completed\"}\n\n")),
	}, nil
}

func TestPelicanHandlerTransmitsQuestionWithoutAccountWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pelicanHandlerRepo{account: &service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "test-token"}}}
	upstream := &pelicanHandlerUpstream{}
	handler := &AccountHandler{accountTestService: service.NewAccountTestService(repo, nil, nil, nil, nil, upstream, nil, nil)}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"model_id":"gpt-5.4","prompt":"Draw a cycling pelican","reasoning_effort":"high"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.PelicanTest(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Equal(t, "Draw a cycling pelican", gjson.GetBytes(upstream.body, "input.0.content.0.text").String())
	require.Equal(t, "high", gjson.GetBytes(upstream.body, "reasoning.effort").String())
	require.NotContains(t, strings.ToLower(string(upstream.body)), "sub2api")
	for key, values := range upstream.headers {
		require.NotContains(t, strings.ToLower(key+strings.Join(values, " ")), "sub2api")
	}
	// The repo embeds an unset write interface. Any health write would panic,
	// including the usage-header update normally performed by connectivity tests.
}
