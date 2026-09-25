import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('AccountTableFilters plan tier', () => {
  it('disables plan-tier select when platform is empty', () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: { platform: '', type: '', plan_tier: '', status: '', privacy_mode: '', group: '' }
      }
    })

    const planTier = wrapper.findComponent('[data-testid="account-plan-tier-filter"]')
    expect(planTier.exists()).toBe(true)
    expect(planTier.props('disabled')).toBe(true)
  })

  it('clears incompatible plan_tier when platform changes', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: 'openai',
          type: '',
          plan_tier: 'pro_5x',
          status: '',
          privacy_mode: '',
          group: ''
        }
      }
    })

    const selects = wrapper.findAllComponents({ name: 'Select' })
    // platform is the second interactive control after SearchInput's siblings: platform Select is first Select
    await selects[0].vm.$emit('update:modelValue', 'grok')

    const emitted = wrapper.emitted('update:filters')
    expect(emitted).toBeTruthy()
    const latest = emitted![emitted!.length - 1][0] as Record<string, unknown>
    expect(latest.platform).toBe('grok')
    expect(latest.plan_tier).toBe('')
  })

  it('exposes Grok options without ChatGPT Pro 20x labels', () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: 'grok',
          type: '',
          plan_tier: '',
          status: '',
          privacy_mode: '',
          group: ''
        }
      }
    })

    const planTier = wrapper.findComponent('[data-testid="account-plan-tier-filter"]')
    const options = (planTier.props('options') as Array<{ value: string; label: string }>) || []
    const labels = options.map((option) => option.label)
    expect(labels).toContain('Grok Free')
    expect(labels).toContain('SuperGrok')
    expect(labels).not.toContain('Pro 20x')
    expect(labels).not.toContain('Business Premium')
  })
})
