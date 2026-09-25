package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePlanType(t *testing.T) {
	require.Equal(t, "selfservebusinessprolite", NormalizePlanType(" self_serve_business_prolite "))
	require.Equal(t, "prolite", NormalizePlanType("PRO_LITE"))
	require.Equal(t, "chatgptpro", NormalizePlanType("ChatGPT-Pro"))
	require.Equal(t, "", NormalizePlanType("   "))
}

func TestPlanTierAliases_OpenAI(t *testing.T) {
	require.Equal(t, []string{"prolite"}, PlanTierAliases(PlatformOpenAI, PlanTierPro5x))
	require.Equal(t, []string{"pro", "chatgptpro"}, PlanTierAliases(PlatformOpenAI, PlanTierPro20x))
	require.Equal(t, []string{"team"}, PlanTierAliases(PlatformOpenAI, PlanTierBusinessStandard))
	require.Equal(t, []string{"selfservebusinessprolite"}, PlanTierAliases(PlatformOpenAI, PlanTierBusinessPremium))
	require.Equal(t, []string{"free"}, PlanTierAliases(PlatformOpenAI, PlanTierFree))
	require.Equal(t, []string{"plus"}, PlanTierAliases(PlatformOpenAI, PlanTierPlus))
}

func TestPlanTierAliases_IgnoresWithoutPlatform(t *testing.T) {
	require.Nil(t, PlanTierAliases("", PlanTierFree))
	require.Nil(t, PlanTierAliases(PlatformOpenAI, ""))
	require.Nil(t, PlanTierAliases("gemini", PlanTierFree))
}

func TestPlanTierAliases_PlatformScoped(t *testing.T) {
	// OpenAI prolite must not leak into Grok/Antigravity catalogs.
	require.Nil(t, PlanTierAliases(PlatformGrok, PlanTierPro5x))
	require.Nil(t, PlanTierAliases(PlatformAntigravity, PlanTierPro20x))
	require.Equal(t, []string{"pro"}, PlanTierAliases(PlatformAntigravity, PlanTierPro))
	require.Equal(t, []string{"ultra"}, PlanTierAliases(PlatformAntigravity, PlanTierUltra))
	require.Equal(t, []string{"free", "basic", "xbasic"}, PlanTierAliases(PlatformGrok, PlanTierFree))
	require.Equal(t, []string{"supergrok", "supergrokpro"}, PlanTierAliases(PlatformGrok, PlanTierSuperGrok))
	require.Equal(t, []string{"supergrokheavy", "heavy"}, PlanTierAliases(PlatformGrok, PlanTierSuperGrokHeavy))
}

func TestMatchesPlanTier_OpenAIAliases(t *testing.T) {
	require.True(t, MatchesPlanTier(PlatformOpenAI, PlanTierPro5x, "pro_lite"))
	require.True(t, MatchesPlanTier(PlatformOpenAI, PlanTierPro20x, "ChatGPT_Pro"))
	require.True(t, MatchesPlanTier(PlatformOpenAI, PlanTierBusinessPremium, "self_serve_business_prolite"))
	require.True(t, MatchesPlanTier(PlatformOpenAI, PlanTierBusinessStandard, "team"))
	require.False(t, MatchesPlanTier(PlatformOpenAI, PlanTierPro5x, "pro"))
	require.False(t, MatchesPlanTier(PlatformGrok, PlanTierPro20x, "pro"))
}
