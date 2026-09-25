package repository

import (
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

// applyAccountPlanTierFilter appends a platform-scoped plan_tier predicate.
// plan_tier without platform (or unknown IDs) is ignored so old clients stay safe.
//
// OpenAI / Antigravity: credentials.plan_type, with parent plan_type fallback when
// parent_account_id is set and the child value is empty (spark shadows).
// Grok: credentials.subscription_tier OR credentials.plan_type.
// Snapshot-based Grok Heavy inference is intentionally not replicated in SQL.
func applyAccountPlanTierFilter(q *dbent.AccountQuery, platform, planTier string) *dbent.AccountQuery {
	aliases := service.PlanTierAliases(platform, planTier)
	if len(aliases) == 0 {
		return q
	}
	platform = strings.TrimSpace(strings.ToLower(platform))
	return q.Where(accountPlanTierPredicate(platform, aliases))
}

func accountPlanTierPredicate(platform string, aliases []string) dbpredicate.Account {
	return dbpredicate.Account(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			switch platform {
			case service.PlatformGrok:
				writeNormalizedJSONTextIn(b, s, dbaccount.FieldCredentials, "subscription_tier", aliases)
				b.WriteString(" OR ")
				writeNormalizedJSONTextIn(b, s, dbaccount.FieldCredentials, "plan_type", aliases)
			default:
				// OpenAI / Antigravity (and any future platform using plan_type + parent fallback)
				writeEffectivePlanTypeNormalizedIn(b, s, aliases)
			}
		}))
	})
}

// writeNormalizedJSONTextIn writes:
//
//	regexp_replace(lower(btrim(coalesce(<col>->>'key', ''))), '[\s_-]+', '', 'g') IN (...)
func writeNormalizedJSONTextIn(b *entsql.Builder, s *entsql.Selector, field, jsonKey string, aliases []string) {
	b.WriteString("regexp_replace(lower(btrim(coalesce(")
	b.Ident(s.C(field))
	b.WriteString("->>'")
	b.WriteString(jsonKey)
	b.WriteString("', ''))), '[\\s_-]+', '', 'g') IN (")
	writeStringArgs(b, aliases)
	b.WriteByte(')')
}

// writeEffectivePlanTypeNormalizedIn matches child plan_type, or parent plan_type
// when parent_account_id is set and the child value is blank.
func writeEffectivePlanTypeNormalizedIn(b *entsql.Builder, s *entsql.Selector, aliases []string) {
	b.WriteString("regexp_replace(lower(btrim(coalesce(")
	b.WriteString("NULLIF(BTRIM(")
	b.Ident(s.C(dbaccount.FieldCredentials))
	b.WriteString("->>'plan_type'), ''), ")
	b.WriteString("(CASE WHEN ")
	b.Ident(s.C(dbaccount.FieldParentAccountID))
	b.WriteString(" IS NOT NULL THEN (SELECT NULLIF(BTRIM(p.credentials->>'plan_type'), '') FROM ")
	b.Ident(dbaccount.Table)
	b.WriteString(" AS p WHERE p.id = ")
	b.Ident(s.C(dbaccount.FieldParentAccountID))
	b.WriteString(" AND p.deleted_at IS NULL) ELSE NULL END)")
	b.WriteString(", ''))), '[\\s_-]+', '', 'g') IN (")
	writeStringArgs(b, aliases)
	b.WriteByte(')')
}

func writeStringArgs(b *entsql.Builder, values []string) {
	for i, value := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.Arg(value)
	}
}
