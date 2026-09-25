package service

import (
	"strings"
	"unicode"
)

// Plan-tier filter IDs for admin account list (?plan_tier=).
// Alias tables mirror frontend/src/utils/planTierFilter.ts and must stay in sync
// with PlatformTypeBadge / openAIPlanTypeLabel cases.
//
// MVP limit: Grok snapshot-based Heavy upgrades are not replicated in SQL;
// filtering uses credentials.subscription_tier / plan_type only.

const (
	PlanTierFree             = "free"
	PlanTierPlus             = "plus"
	PlanTierPro5x            = "pro_5x"
	PlanTierPro20x           = "pro_20x"
	PlanTierBusinessStandard = "business_standard"
	PlanTierBusinessPremium  = "business_premium"
	PlanTierPro              = "pro"
	PlanTierUltra            = "ultra"
	PlanTierSuperGrok        = "supergrok"
	PlanTierSuperGrokLite    = "supergrok_lite"
	PlanTierSuperGrokPlus    = "supergrok_plus"
	PlanTierSuperGrokHeavy   = "supergrok_heavy"
)

// NormalizePlanType matches frontend normalizePlanType: trim, lower-case, strip
// spaces / underscores / hyphens.
func NormalizePlanType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var openAIPlanTierAliases = map[string][]string{
	PlanTierFree:             {"free"},
	PlanTierPlus:             {"plus"},
	PlanTierPro5x:            {"prolite"},
	PlanTierPro20x:           {"pro", "chatgptpro"},
	PlanTierBusinessStandard: {"team"},
	PlanTierBusinessPremium:  {"selfservebusinessprolite"},
}

var antigravityPlanTierAliases = map[string][]string{
	PlanTierFree:  {"free"},
	PlanTierPro:   {"pro"},
	PlanTierUltra: {"ultra"},
}

var grokPlanTierAliases = map[string][]string{
	PlanTierFree:           {"free", "basic", "xbasic"},
	PlanTierSuperGrok:      {"supergrok", "supergrokpro"}, // ambiguous paid bucket
	PlanTierSuperGrokLite:  {"supergroklite"},
	PlanTierSuperGrokPlus:  {"supergrokplus"},
	PlanTierSuperGrokHeavy: {"supergrokheavy", "heavy"},
}

// PlanTierAliases returns normalized stored aliases for a platform-scoped filter
// ID. Empty platform, empty tier, or unknown combinations return nil so callers
// can ignore the filter (plan_tier without platform is a no-op).
func PlanTierAliases(platform, planTier string) []string {
	platform = strings.TrimSpace(strings.ToLower(platform))
	planTier = strings.TrimSpace(strings.ToLower(planTier))
	if platform == "" || planTier == "" {
		return nil
	}
	var catalog map[string][]string
	switch platform {
	case PlatformOpenAI:
		catalog = openAIPlanTierAliases
	case PlatformAntigravity:
		catalog = antigravityPlanTierAliases
	case PlatformGrok:
		catalog = grokPlanTierAliases
	default:
		return nil
	}
	aliases, ok := catalog[planTier]
	if !ok || len(aliases) == 0 {
		return nil
	}
	out := make([]string, len(aliases))
	copy(out, aliases)
	return out
}

// MatchesPlanTier reports whether any stored credential value matches the
// platform-scoped filter aliases after NormalizePlanType.
func MatchesPlanTier(platform, planTier string, storedValues ...string) bool {
	aliases := PlanTierAliases(platform, planTier)
	if len(aliases) == 0 {
		return false
	}
	aliasSet := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		aliasSet[alias] = struct{}{}
	}
	for _, value := range storedValues {
		normalized := NormalizePlanType(value)
		if normalized == "" {
			continue
		}
		if _, ok := aliasSet[normalized]; ok {
			return true
		}
	}
	return false
}
