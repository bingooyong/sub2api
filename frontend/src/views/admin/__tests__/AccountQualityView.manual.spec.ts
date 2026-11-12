import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AccountQualityView from '../AccountQualityView.vue'

vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key, te: () => false }) }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<main><slot /></main>' } }))
vi.mock('@/api/admin/accountQuality', () => ({ listQualityPlans: vi.fn(async () => []), listQualityOperations: vi.fn(async () => ({ items: [], next_cursor: 0 })) }))
vi.mock('@/api/admin/groups', () => ({ getAllIncludingInactive: vi.fn(async () => []) }))
vi.mock('@/api/admin/accounts', () => ({ list: vi.fn(async () => ({ items: [], total: 0 })) }))
vi.mock('@/api/admin/scheduledTests', () => ({ default: {} }))

afterEach(() => { document.body.innerHTML = '' })

describe('quality operations manual disclosure', () => {
  it('links expanded state, focuses the opened panel and restores the opener while retaining its mounted state', async () => {
    const wrapper = mount(AccountQualityView, { attachTo: document.body, global: { stubs: { AppLayout: { template: '<main><slot /></main>' } } } })
    try {
      await flushPromises()
      const opener = wrapper.get('button[aria-controls="manual-quality-panel"]')
      expect(opener.attributes('aria-expanded')).toBe('false')
      ;(opener.element as HTMLButtonElement).focus()
      await opener.trigger('click')
      await flushPromises()
      const panel = wrapper.get('#manual-quality-panel')
      expect(opener.attributes('aria-expanded')).toBe('true')
      expect(panel.element.contains(document.activeElement)).toBe(true)
      const close = panel.get('[data-testid="manual-close"]')
      ;(close.element as HTMLButtonElement).focus()
      await close.trigger('click')
      await flushPromises()
      expect(opener.attributes('aria-expanded')).toBe('false')
      expect(document.activeElement).toBe(opener.element)
      expect(panel.isVisible()).toBe(false)
      await opener.trigger('click')
      await flushPromises()
      expect(wrapper.get('#manual-quality-panel').element).toBe(panel.element)
      expect(panel.element.contains(document.activeElement)).toBe(true)
    } finally { wrapper.unmount() }
  })
})
