import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import ManualQualityPanel from '../ManualQualityPanel.vue'
import { list } from '@/api/admin/accounts'
import { runManualQualityProbe } from '@/api/admin/manualQuality'
import { CANDY_PROMPT } from '@/utils/intelligenceTest'
import type { AccountListItem } from '@/types'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/admin/accounts', () => ({ list: vi.fn() }))
vi.mock('@/api/admin/manualQuality', async () => ({ ...await vi.importActual<typeof import('@/api/admin/manualQuality')>('@/api/admin/manualQuality'), runManualQualityProbe: vi.fn() }))

const account = (id: number) => ({ id, name: `Account ${id}`, platform: 'openai', type: 'oauth', status: 'active' }) as AccountListItem
const wrappers: VueWrapper[] = []
interface Pending {
  accountID: number
  signal: AbortSignal
  onText: (text: string) => void
  resolve: () => void
  reject: (error: Error) => void
}
let pending: Pending[]
async function open() {
  const wrapper = mount(ManualQualityPanel)
  wrappers.push(wrapper)
  await flushPromises()
  return wrapper
}
async function selectAll(wrapper: VueWrapper) { for (const box of wrapper.findAll('[data-testid="manual-account"]')) await box.setValue(true) }
const results = (wrapper: VueWrapper) => wrapper.findAll('[data-testid="manual-result"]')

describe('ManualQualityPanel', () => {
  beforeEach(() => {
    pending = []
    vi.mocked(list).mockResolvedValue({ items: [account(1), account(2), account(3)], total: 3, page: 1, page_size: 50, pages: 1 })
    vi.mocked(runManualQualityProbe).mockImplementation((accountID, _request, signal, onText) => new Promise<void>((resolve, reject) => { pending.push({ accountID, signal, onText, resolve, reject }) }))
  })
  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
    vi.clearAllMocks()
  })

  it('limits concurrency, isolates account failures and snapshots settings for queued requests', async () => {
    const wrapper = await open()
    await selectAll(wrapper)
    await wrapper.get('[data-testid="manual-concurrency"]').setValue(2)
    await wrapper.get('form').trigger('submit')
    expect(runManualQualityProbe).toHaveBeenCalledTimes(2)
    expect(results(wrapper).map(card => card.attributes('data-status'))).toEqual(['running', 'running', 'pending'])
    await wrapper.get('[data-testid="manual-model"]').setValue('changed-after-start')
    pending[0]!.onText('<svg><text>pelican</text></svg>')
    pending[0]!.resolve()
    await flushPromises()
    expect(runManualQualityProbe).toHaveBeenCalledTimes(3)
    for (const call of vi.mocked(runManualQualityProbe).mock.calls) {
      expect(call[1]).toMatchObject({ model_id: 'gpt-6-astra', reasoning_effort: 'high' })
      expect(call[1].prompt).toContain('鹈鹕骑自行车')
      expect(call[1].prompt).toContain('直接返回独立 HTML')
    }
    pending[1]!.reject(new Error('HTTP 429'))
    pending[2]!.onText('No drawing this time')
    pending[2]!.resolve()
    await flushPromises()
    expect(results(wrapper).map(card => card.attributes('data-status'))).toEqual(['success', 'error', 'success'])
    expect(wrapper.text()).toContain('HTTP 429')
    const frame = wrapper.get('iframe')
    expect(frame.attributes('sandbox')).toBe('')
    expect(frame.attributes('referrerpolicy')).toBe('no-referrer')
    expect(frame.attributes('srcdoc')).toContain("connect-src 'none'")
    expect(frame.attributes('srcdoc')).toContain('Content-Security-Policy')
    expect(wrapper.text()).toContain('qualityOps.manual.noPreview')
    expect(wrapper.text()).not.toContain('qualityOps.passed')
  })

  it('stops active and queued runs while keeping finished cards and rejects stale callbacks', async () => {
    const wrapper = await open()
    await selectAll(wrapper)
    await wrapper.get('[data-testid="manual-concurrency"]').setValue(1)
    await wrapper.get('form').trigger('submit')
    pending[0]!.onText('<svg></svg>')
    pending[0]!.resolve()
    await flushPromises()
    expect(pending).toHaveLength(2)
    await wrapper.get('[data-testid="manual-stop"]').trigger('click')
    expect(pending[1]!.signal.aborted).toBe(true)
    expect(results(wrapper).map(card => card.attributes('data-status'))).toEqual(['success', 'cancelled', 'cancelled'])
    expect(wrapper.find('iframe').exists()).toBe(true)
    await wrapper.get('form').trigger('submit')
    expect(pending).toHaveLength(3)
    pending[1]!.onText('stale response')
    pending[1]!.resolve()
    await flushPromises()
    expect(results(wrapper).map(card => card.attributes('data-status'))).toEqual(['running', 'pending', 'pending'])
    expect(wrapper.text()).not.toContain('stale response')
    expect(pending).toHaveLength(3)
  })

  it('retains selected accounts across paginated searches and does not exclude scheduled accounts', async () => {
    vi.mocked(list).mockResolvedValueOnce({ items: [account(1)], total: 51, page: 1, page_size: 50, pages: 2 })
      .mockResolvedValueOnce({ items: [account(51)], total: 51, page: 2, page_size: 50, pages: 2 })
    const wrapper = await open()
    await wrapper.get('[data-testid="manual-account"]').setValue(true)
    expect(wrapper.get('[data-testid="manual-account"]').attributes('disabled')).toBeUndefined()
    await wrapper.get('[data-testid="manual-next"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="manual-account"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    expect(vi.mocked(runManualQualityProbe).mock.calls.map(call => call[0])).toEqual([1, 51])
    expect(list).toHaveBeenLastCalledWith(2, 50, expect.objectContaining({ platform: 'openai', lite: 'true' }), expect.objectContaining({ signal: expect.any(AbortSignal) }))
  })

  it('uses the candy contract and reference without adding an HTML requirement', async () => {
    const wrapper = await open()
    await wrapper.get('[data-testid="manual-account"]').setValue(true)
    await wrapper.get('[data-testid="manual-preset"]').setValue('candy')
    await wrapper.get('form').trigger('submit')
    const body = vi.mocked(runManualQualityProbe).mock.calls[0]![1]
    expect(body.prompt).toContain(CANDY_PROMPT)
    expect(body.prompt).toContain('只输出最终整数')
    expect(body.prompt).not.toContain('独立 HTML')
    pending[0]!.onText('21')
    pending[0]!.resolve()
    await flushPromises()
    expect(results(wrapper)[0]!.attributes('data-status')).toBe('success')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('qualityOps.manual.candyReference')
  })

  it('sends custom questions unchanged and hides the candy reference after editing its question', async () => {
    const wrapper = await open()
    await wrapper.get('[data-testid="manual-account"]').setValue(true)
    await wrapper.get('[data-testid="manual-preset"]').setValue('candy')
    await wrapper.get('[data-testid="manual-prompt"]').setValue('A different candy question')
    expect(wrapper.text()).not.toContain('qualityOps.manual.candyReference')
    await wrapper.get('[data-testid="manual-preset"]').setValue('custom')
    await wrapper.get('[data-testid="manual-prompt"]').setValue('Draw a partridge riding a bicycle')
    await wrapper.get('form').trigger('submit')
    expect(vi.mocked(runManualQualityProbe).mock.calls[0]![1].prompt).toBe('Draw a partridge riding a bicycle')
  })

  it.each(['close', 'unmount'])('aborts requests on %s', async action => {
    const wrapper = await open()
    await selectAll(wrapper)
    await wrapper.get('form').trigger('submit')
    if (action === 'close') await wrapper.get('[data-testid="manual-close"]').trigger('click')
    else wrapper.unmount()
    expect(pending.every(call => call.signal.aborted)).toBe(true)
    if (action === 'close') {
      expect(wrapper.emitted('close')).toHaveLength(1)
      expect(results(wrapper).every(card => card.attributes('data-status') === 'cancelled')).toBe(true)
    }
  })
})
