/**
 * Account-list plan-tier filter catalogs.
 *
 * Filter IDs are stable query values (`plan_tier=`). Labels match PlatformTypeBadge
 * copy. Alias matching uses the same normalization as `normalizePlanType`.
 *
 * Platform vocabularies are intentionally separate: OpenAI Pro 5x/20x labels must
 * never be applied when filtering Grok or Antigravity.
 */

import { normalizePlanType } from '@/utils/planType'

export type PlanTierOption = { value: string; label: string }

/** OpenAI filter ID → stored aliases (already normalized). */
const OPENAI_PLAN_TIER_ALIASES: Record<string, string[]> = {
  free: ['free'],
  plus: ['plus'],
  pro_5x: ['prolite'],
  pro_20x: ['pro', 'chatgptpro'],
  business_standard: ['team'],
  business_premium: ['selfservebusinessprolite']
}

const OPENAI_PLAN_TIER_OPTIONS: PlanTierOption[] = [
  { value: 'free', label: 'Free' },
  { value: 'plus', label: 'Plus' },
  { value: 'pro_5x', label: 'Pro 5x' },
  { value: 'pro_20x', label: 'Pro 20x' },
  { value: 'business_standard', label: 'Business Standard' },
  { value: 'business_premium', label: 'Business Premium' }
]

const ANTIGRAVITY_PLAN_TIER_ALIASES: Record<string, string[]> = {
  free: ['free'],
  pro: ['pro'],
  ultra: ['ultra']
}

const ANTIGRAVITY_PLAN_TIER_OPTIONS: PlanTierOption[] = [
  { value: 'free', label: 'Free' },
  { value: 'pro', label: 'Pro' },
  { value: 'ultra', label: 'Ultra' }
]

const GROK_PLAN_TIER_ALIASES: Record<string, string[]> = {
  free: ['free', 'basic', 'xbasic'],
  // Ambiguous paid bucket: treat SuperGrokPro as SuperGrok (not Heavy).
  // Snapshot-based Heavy upgrades are an accepted MVP limit — not mirrored in SQL.
  supergrok: ['supergrok', 'supergrokpro'],
  supergrok_lite: ['supergroklite'],
  supergrok_plus: ['supergrokplus'],
  supergrok_heavy: ['supergrokheavy', 'heavy']
}

const GROK_PLAN_TIER_OPTIONS: PlanTierOption[] = [
  { value: 'free', label: 'Grok Free' },
  { value: 'supergrok', label: 'SuperGrok' },
  { value: 'supergrok_lite', label: 'SuperGrok Lite' },
  { value: 'supergrok_plus', label: 'SuperGrok Plus' },
  { value: 'supergrok_heavy', label: 'SuperGrok Heavy' }
]

function catalogForPlatform(platform: string): {
  options: PlanTierOption[]
  aliases: Record<string, string[]>
} | null {
  switch ((platform || '').trim().toLowerCase()) {
    case 'openai':
      return { options: OPENAI_PLAN_TIER_OPTIONS, aliases: OPENAI_PLAN_TIER_ALIASES }
    case 'antigravity':
      return { options: ANTIGRAVITY_PLAN_TIER_OPTIONS, aliases: ANTIGRAVITY_PLAN_TIER_ALIASES }
    case 'grok':
      return { options: GROK_PLAN_TIER_OPTIONS, aliases: GROK_PLAN_TIER_ALIASES }
    default:
      return null
  }
}

/** Options for the plan-tier Select; empty when platform has no catalog. */
export function planTierOptionsForPlatform(platform: string): PlanTierOption[] {
  return catalogForPlatform(platform)?.options.slice() ?? []
}

/** Normalized stored aliases for a filter ID, or [] when platform/tier is empty/unknown. */
export function planTierAliases(platform: string, planTier: string): string[] {
  const catalog = catalogForPlatform(platform)
  if (!catalog) return []
  const key = (planTier || '').trim().toLowerCase()
  if (!key) return []
  const aliases = catalog.aliases[key]
  return aliases ? aliases.slice() : []
}

/** True when any stored credential value matches the platform-scoped filter aliases. */
export function matchesPlanTier(
  platform: string,
  planTier: string,
  ...storedValues: Array<string | null | undefined>
): boolean {
  const aliases = planTierAliases(platform, planTier)
  if (aliases.length === 0) return false
  const aliasSet = new Set(aliases)
  for (const value of storedValues) {
    const normalized = normalizePlanType(value)
    if (normalized && aliasSet.has(normalized)) return true
  }
  return false
}

/** Clear plan_tier when platform is empty or the selected ID is not in the new catalog. */
export function reconcilePlanTierForPlatform(
  platform: string,
  planTier: string | null | undefined
): string {
  if (!platform) return ''
  const current = typeof planTier === 'string' ? planTier : ''
  if (!current) return ''
  const options = planTierOptionsForPlatform(platform)
  return options.some((option) => option.value === current) ? current : ''
}
