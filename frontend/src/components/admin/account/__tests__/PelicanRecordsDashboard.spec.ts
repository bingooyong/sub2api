import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PelicanRecordsDashboard from '../PelicanRecordsDashboard.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/admin/scheduledTests', () => ({ scheduledTestsAPI: { listPelicanHistory: vi.fn(async () => ({ items: [] })) } }))

describe('Pelican record previews', () => {
  it('uses the passive preview policy and retains raw output for script-driven drawings', async () => {
    const output = '<html><body><svg><circle r="5"/></svg><script>location.href="/leak"</script></body></html>'
    const wrapper = mount(PelicanRecordsDashboard, { props: { accounts: [], account: null, manualRecord: { runs: [{ output, status: 'success' }] } } })
    try {
      await flushPromises()
      const frame = wrapper.get('iframe')
      expect(frame.attributes('sandbox')).toBe('')
      expect(frame.attributes('srcdoc')).toContain("script-src 'none'")
      expect(frame.attributes('srcdoc')).not.toContain('<script')
      expect(wrapper.get('details pre').text()).toBe(output)
      expect(wrapper.text()).toContain('qualityOps.manual.previewHelp')
    } finally { wrapper.unmount() }
  })
})
