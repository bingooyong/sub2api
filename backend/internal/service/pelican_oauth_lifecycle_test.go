//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type pelicanOAuthRepo struct {
	*openAIAccountTestRepo
	credentialWrites int
}

func (r *pelicanOAuthRepo) UpdateCredentials(_ context.Context, id int64, credentials map[string]any) error {
	r.credentialWrites++
	r.accountsByID[id].Credentials = shallowCopyMap(credentials)
	return nil
}

type pelicanOAuthCache struct {
	*openAITokenCacheStub
	deletes int
}

func (c *pelicanOAuthCache) DeleteAccessToken(ctx context.Context, key string) error {
	c.deletes++
	return c.openAITokenCacheStub.DeleteAccessToken(ctx, key)
}

func TestManualPelicanOAuthLifecycleUsesProviderWithoutHealthWrites(t *testing.T) {
	for _, mode := range []string{"cache", "cancelled_before_cache", "refresh", "missing_refresh", "refresh_error", "cancelled_refresh", "lock_held", "unconfigured_refresh"} {
		t.Run(mode, func(t *testing.T) {
			a := manualPelicanAccount(AccountTypeOAuth, false)
			a.Credentials["expires_at"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
			a.Credentials["refresh_token"] = "old-refresh"
			svc, observationRepo, upstream := manualPelicanService(a, newJSONResponse(http.StatusOK, "data: {\"type\":\"response.completed\"}\n\n"))
			repo := &pelicanOAuthRepo{openAIAccountTestRepo: observationRepo}
			svc.accountRepo = repo
			cache := &pelicanOAuthCache{openAITokenCacheStub: newOpenAITokenCacheStub()}
			provider := NewOpenAITokenProvider(repo, cache, nil)
			blocker := &runtimeBlockRecorder{}
			provider.SetAccountRuntimeBlocker(blocker)
			executor := &refreshAPIExecutorStub{needsRefresh: true, credentials: map[string]any{
				"access_token": "renewed-token", "refresh_token": "renewed-refresh",
				"expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
			}}
			provider.SetRefreshAPI(NewOAuthRefreshAPI(repo, cache), executor)
			svc.openaiGatewayService = &OpenAIGatewayService{accountRepo: repo, openAITokenProvider: provider}
			ctx, _ := newTestContext()
			requestCtx, cancel := context.WithCancel(ctx.Request.Context())
			defer cancel()
			ctx.Request = ctx.Request.WithContext(requestCtx)
			switch mode {
			case "cache", "cancelled_before_cache":
				cache.tokens[OpenAITokenCacheKey(a)] = "fresh-cached-token"
				if mode == "cancelled_before_cache" {
					cancel()
				}
			case "missing_refresh":
				delete(a.Credentials, "refresh_token")
			case "refresh_error":
				executor.err = errors.New("refresh rejected")
			case "cancelled_refresh":
				executor.onRefresh = cancel
			case "lock_held":
				cache.lockAcquired = false
			case "unconfigured_refresh":
				provider.SetRefreshAPI(nil, nil)
			}

			err := svc.TestManualPelicanAccountConnection(ctx, 1, "gpt-5.4", "Question", "high")
			if mode == "cache" || mode == "refresh" {
				require.NoError(t, err)
				require.Len(t, upstream.requests, 1)
				wantToken := "fresh-cached-token"
				if mode == "refresh" {
					wantToken = "renewed-token"
					require.Equal(t, 1, executor.refreshCalls)
					require.Equal(t, 1, repo.credentialWrites)
					require.Equal(t, "renewed-refresh", a.GetOpenAIRefreshToken())
					require.Positive(t, a.GetCredentialAsInt64("_token_version"))
					require.Equal(t, wantToken, cache.tokens[OpenAITokenCacheKey(a)])
				} else {
					require.Zero(t, executor.refreshCalls)
					require.Zero(t, repo.credentialWrites)
				}
				require.Equal(t, "Bearer "+wantToken, upstream.requests[0].Header.Get("Authorization"))
			} else {
				require.Error(t, err)
				require.Empty(t, upstream.requests, "failed credential resolution must not dispatch a stale token")
				require.Zero(t, repo.credentialWrites)
				require.Zero(t, cache.setCalled)
				if mode == "refresh_error" || mode == "cancelled_refresh" {
					require.Equal(t, 1, executor.refreshCalls)
				}
			}
			require.Zero(t, cache.deletes)
			require.Empty(t, blocker.accounts)
			require.Empty(t, blocker.clearedIDs)
			assertPelicanObservation(t, a, observationRepo)
		})
	}
}

func TestOpenAITokenProviderPelicanMissingRefreshDoesNotDisableAccount(t *testing.T) {
	a := manualPelicanAccount(AccountTypeOAuth, false)
	a.Credentials["expires_at"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	_, repo, _ := manualPelicanService(a)
	cache := &pelicanOAuthCache{openAITokenCacheStub: newOpenAITokenCacheStub()}
	provider := NewOpenAITokenProvider(repo, cache, nil)
	blocker := &runtimeBlockRecorder{}
	provider.SetAccountRuntimeBlocker(blocker)

	token, err := provider.GetAccessToken(withPelicanTestOptions(context.Background(), pelicanTestOptions{}), a)
	require.ErrorContains(t, err, "refresh_token is missing")
	require.Empty(t, token)
	require.Empty(t, blocker.accounts)
	require.Zero(t, cache.deletes)
	assertPelicanObservation(t, a, repo)
}

func TestOpenAITokenProviderPelicanRejectsRefreshFailureButForwardingKeepsFallback(t *testing.T) {
	for _, probe := range []bool{true, false} {
		a := manualPelicanAccount(AccountTypeOAuth, false)
		a.Credentials["refresh_token"] = "old-refresh"
		a.Credentials["expires_at"] = time.Now().Add(time.Minute).UTC().Format(time.RFC3339)
		_, observationRepo, _ := manualPelicanService(a)
		repo := &pelicanOAuthRepo{openAIAccountTestRepo: observationRepo}
		cache := newOpenAITokenCacheStub()
		provider := NewOpenAITokenProvider(repo, cache, nil)
		executor := &refreshAPIExecutorStub{needsRefresh: true, err: errors.New("refresh unavailable")}
		provider.SetRefreshAPI(NewOAuthRefreshAPI(repo, cache), executor)
		ctx := context.Background()
		if probe {
			ctx = withPelicanTestOptions(ctx, pelicanTestOptions{})
		}
		token, err := provider.GetAccessToken(ctx, a)
		require.Equal(t, 1, executor.refreshCalls)
		if probe {
			require.ErrorIs(t, err, executor.err)
			require.Empty(t, token)
			require.Zero(t, cache.setCalled)
		} else {
			require.NoError(t, err)
			require.Equal(t, "test-token", token)
			require.EqualValues(t, 1, cache.setCalled)
		}
		assertPelicanObservation(t, a, observationRepo)
	}
}

func TestPelicanOAuthInjectedGatewayPreservesConnectivityAndAgentIdentity(t *testing.T) {
	t.Run("ordinary_connectivity_keeps_snapshot", func(t *testing.T) {
		a := manualPelicanAccount(AccountTypeOAuth, false)
		svc, _, upstream := manualPelicanService(a, newJSONResponse(http.StatusOK, "data: {\"type\":\"response.completed\"}\n\n"))
		cache := newOpenAITokenCacheStub()
		cache.tokens[OpenAITokenCacheKey(a)] = "provider-token"
		svc.openaiGatewayService = &OpenAIGatewayService{openAITokenProvider: NewOpenAITokenProvider(nil, cache, nil)}
		ctx, _ := newTestContext()
		require.NoError(t, svc.TestAccountConnection(ctx, a.ID, "gpt-5.4", "", AccountTestModeDefault))
		require.Len(t, upstream.requests, 1)
		require.Equal(t, "Bearer test-token", upstream.requests[0].Header.Get("Authorization"))
		require.Zero(t, cache.getCalled)
	})
	t.Run("agent_identity_bypasses_bearer_provider", func(t *testing.T) {
		a := manualPelicanAccount(AccountTypeOAuth, false)
		key, privateKey := newTestAgentIdentityKey(t)
		a.Credentials = map[string]any{
			"auth_mode":        OpenAIAuthModeAgentIdentity,
			"agent_runtime_id": key.runtimeID, "agent_private_key": privateKey, "task_id": key.taskID,
		}
		svc, repo, upstream := manualPelicanService(a, newJSONResponse(http.StatusOK, "data: {\"type\":\"response.completed\"}\n\n"))
		cache := newOpenAITokenCacheStub()
		svc.openaiGatewayService = &OpenAIGatewayService{openAITokenProvider: NewOpenAITokenProvider(nil, cache, nil)}
		ctx, _ := newTestContext()
		require.NoError(t, svc.TestManualPelicanAccountConnection(ctx, a.ID, "gpt-5.4", "Question", "high"))
		require.Len(t, upstream.requests, 1)
		require.Contains(t, upstream.requests[0].Header.Get("Authorization"), "AgentAssertion ")
		require.NotContains(t, upstream.requests[0].Header.Get("Authorization"), privateKey)
		require.Zero(t, cache.getCalled)
		assertPelicanObservation(t, a, repo)
	})
}

func TestManualPelicanOAuthVersionCheckValidatesAdoptedCredentialExpiry(t *testing.T) {
	for _, mode := range []string{"expired", "fresh", "pat"} {
		t.Run(mode, func(t *testing.T) {
			snapshot := manualPelicanAccount(AccountTypeOAuth, false)
			snapshot.Credentials["expires_at"] = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
			snapshot.Credentials["_token_version"] = int64(1)
			latest := manualPelicanAccount(AccountTypeOAuth, false)
			latest.Credentials["access_token"] = "latest-token"
			latest.Credentials["_token_version"] = int64(2)
			latest.Credentials["expires_at"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
			if mode == "fresh" {
				latest.Credentials["expires_at"] = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
			} else if mode == "pat" {
				latest.Credentials["auth_mode"] = "personal_access_token"
			}
			svc, snapshotRepo, upstream := manualPelicanService(snapshot, newJSONResponse(http.StatusOK, "data: {\"type\":\"response.completed\"}\n\n"))
			_, latestRepo, _ := manualPelicanService(latest)
			cache := &pelicanOAuthCache{openAITokenCacheStub: newOpenAITokenCacheStub()}
			provider := NewOpenAITokenProvider(latestRepo, cache, nil)
			blocker := &runtimeBlockRecorder{}
			provider.SetAccountRuntimeBlocker(blocker)
			svc.openaiGatewayService = &OpenAIGatewayService{openAITokenProvider: provider}
			ctx, _ := newTestContext()

			err := svc.TestManualPelicanAccountConnection(ctx, snapshot.ID, "gpt-5.4", "Question", "high")
			if mode == "expired" {
				require.ErrorContains(t, err, "expired")
				require.Empty(t, upstream.requests)
			} else {
				require.NoError(t, err)
				require.Len(t, upstream.requests, 1)
				require.Equal(t, "Bearer latest-token", upstream.requests[0].Header.Get("Authorization"))
			}
			require.Zero(t, cache.setCalled, "a versioned token must not republish the stale snapshot")
			require.Zero(t, cache.deletes)
			require.Empty(t, blocker.accounts)
			assertPelicanObservation(t, snapshot, snapshotRepo)
			assertPelicanObservation(t, latest, latestRepo)
		})
	}
}
