package repository

import (
	"strings"
	"testing"

	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
)

func TestAccountPlanTierPredicate_OpenAIUsesParentFallback(t *testing.T) {
	pred := accountPlanTierPredicate(service.PlatformOpenAI, []string{"prolite"})
	selector := entsql.Dialect("postgres").Select("*").From(entsql.Table(dbaccount.Table))
	pred(selector)
	query, args := selector.Query()
	normalized := strings.ToLower(query)
	require.Contains(t, normalized, "plan_type")
	require.Contains(t, normalized, "parent_account_id")
	require.Contains(t, normalized, "credentials->>'plan_type'")
	require.Contains(t, args, "prolite")
}

func TestAccountPlanTierPredicate_GrokUsesSubscriptionOrPlan(t *testing.T) {
	pred := accountPlanTierPredicate(service.PlatformGrok, []string{"supergrokheavy", "heavy"})
	selector := entsql.Dialect("postgres").Select("*").From(entsql.Table(dbaccount.Table))
	pred(selector)
	query, args := selector.Query()
	normalized := strings.ToLower(query)
	require.Contains(t, normalized, "subscription_tier")
	require.Contains(t, normalized, "plan_type")
	require.Contains(t, normalized, " or ")
	require.NotContains(t, normalized, "parent_account_id")
	require.Contains(t, args, "supergrokheavy")
	require.Contains(t, args, "heavy")
}

func TestApplyAccountPlanTierFilter_IgnoresWithoutPlatform(t *testing.T) {
	// Nil query path: aliases empty → no panic / no predicate applied via helper.
	require.Nil(t, service.PlanTierAliases("", "free"))
	require.Nil(t, service.PlanTierAliases(service.PlatformOpenAI, ""))
}
