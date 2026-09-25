import { describe, expect, it } from 'vitest'

import {
  matchesPlanTier,
  planTierAliases,
  planTierOptionsForPlatform,
  reconcilePlanTierForPlatform
} from '@/utils/planTierFilter'

describe('planTierFilter catalogs', () => {
  it('exposes OpenAI ChatGPT tiers with badge labels', () => {
    const options = planTierOptionsForPlatform('openai')
    expect(options.map((o) => o.value)).toEqual([
      'free',
      'plus',
      'pro_5x',
      'pro_20x',
      'business_standard',
      'business_premium'
    ])
    expect(options.map((o) => o.label)).toEqual([
      'Free',
      'Plus',
      'Pro 5x',
      'Pro 20x',
      'Business Standard',
      'Business Premium'
    ])
  })

  it('matches OpenAI aliases the same way as PlatformTypeBadge', () => {
    expect(matchesPlanTier('openai', 'pro_5x', 'pro_lite')).toBe(true)
    expect(matchesPlanTier('openai', 'pro_20x', 'chatgptpro')).toBe(true)
    expect(matchesPlanTier('openai', 'business_standard', 'team')).toBe(true)
    expect(matchesPlanTier('openai', 'business_premium', 'self_serve_business_prolite')).toBe(true)
    expect(matchesPlanTier('openai', 'pro_5x', 'pro')).toBe(false)
  })

  it('uses Antigravity / Grok catalogs without ChatGPT labels', () => {
    expect(planTierOptionsForPlatform('antigravity').map((o) => o.label)).toEqual([
      'Free',
      'Pro',
      'Ultra'
    ])
    expect(planTierOptionsForPlatform('antigravity').some((o) => o.label.includes('Pro 20x'))).toBe(
      false
    )
    expect(planTierOptionsForPlatform('grok').map((o) => o.label)).toEqual([
      'Grok Free',
      'SuperGrok',
      'SuperGrok Lite',
      'SuperGrok Plus',
      'SuperGrok Heavy'
    ])
    expect(planTierAliases('grok', 'free')).toEqual(['free', 'basic', 'xbasic'])
    expect(planTierAliases('grok', 'supergrok_heavy')).toEqual(['supergrokheavy', 'heavy'])
  })

  it('ignores plan tier without a platform and clears incompatible selections', () => {
    expect(planTierOptionsForPlatform('')).toEqual([])
    expect(planTierAliases('', 'free')).toEqual([])
    expect(reconcilePlanTierForPlatform('', 'free')).toBe('')
    expect(reconcilePlanTierForPlatform('grok', 'pro_5x')).toBe('')
    expect(reconcilePlanTierForPlatform('openai', 'pro_5x')).toBe('pro_5x')
  })
})
