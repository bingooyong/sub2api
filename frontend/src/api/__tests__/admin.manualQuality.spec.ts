import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ManualQualityError, runManualQualityProbe } from '../admin/manualQuality'
import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'

const request = { model_id: 'gpt-test', prompt: 'Draw a bicycle', reasoning_effort: 'high' }
function responseFor(text: string, split = 13) {
  const bytes = new TextEncoder().encode(text)
  let offset = 0
  return new Response(new ReadableStream<Uint8Array>({
    pull(controller) {
      if (offset >= bytes.length) controller.close()
      else { controller.enqueue(bytes.slice(offset, offset + split)); offset += split }
    }
  }), { headers: { 'Content-Type': 'text/event-stream' } })
}
const event = (value: object) => `data: ${JSON.stringify(value)}\r\n\r\n`

describe('manual quality probe client', () => {
  beforeEach(() => { localStorage.setItem('auth_token', 'admin-token') })
  afterEach(() => { vi.unstubAllGlobals(); localStorage.clear() })

  it('sends only shared probe options with admin auth and decodes split Unicode SSE', async () => {
    const fetch = vi.fn().mockResolvedValue(responseFor(': heartbeat\r\n\r\n' + event({ type: 'content', text: '鹈鹕' }) + event({ type: 'content', text: '<svg></svg>' }) + event({ type: 'test_complete', success: true }), 1))
    vi.stubGlobal('fetch', fetch)
    const signal = new AbortController().signal
    const text: string[] = []
    await runManualQualityProbe(42, request, signal, value => text.push(value))
    expect(text.join('')).toBe('鹈鹕<svg></svg>')
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/admin/accounts/42/pelican-test'), expect.objectContaining({
      method: 'POST', signal, body: JSON.stringify(request), credentials: 'include',
      headers: expect.objectContaining({ Authorization: 'Bearer admin-token', [ADMIN_UI_REQUEST_HEADER]: '1' })
    }))
    expect(JSON.parse(fetch.mock.calls[0]![1].body)).toEqual(request)
  })

  it('retains streamed partial output when the terminal event fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(responseFor(event({ type: 'content', text: 'partial drawing' }) + event({ type: 'test_complete', success: false, error: 'quota exceeded' }))))
    const text: string[] = []
    await expect(runManualQualityProbe(1, request, new AbortController().signal, value => text.push(value))).rejects.toMatchObject({ code: 'failed', message: 'quota exceeded' })
    expect(text).toEqual(['partial drawing'])
  })

  it.each([
    ['truncated', event({ type: 'content', text: '21' }), 'incomplete'],
    ['empty', event({ type: 'test_complete', success: true }), 'empty'],
    ['malformed', 'data: {broken}\n\n', 'invalidStream'],
    ['upstream error', event({ type: 'error', error: 'upstream unavailable' }), 'failed']
  ])('rejects %s streams', async (_name, text, code) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(responseFor(text)))
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toMatchObject({ code })
  })

  it('reports HTTP validation failures before attempting to parse SSE', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ message: 'Unsupported test model' }), { status: 400 })))
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toEqual(new ManualQualityError('http', 'HTTP 400: Unsupported test model'))
  })

  it('handles a final event without a blank-line terminator', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(responseFor(event({ type: 'content', text: '21' }) + 'data: {"type":"test_complete","success":true}')))
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).resolves.toBeUndefined()
  })

  it('rejects an already cancelled request and closes its response reader', async () => {
    const cancel = vi.fn()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream({ cancel }))))
    const controller = new AbortController()
    controller.abort()
    await expect(runManualQualityProbe(1, request, controller.signal, () => undefined)).rejects.toMatchObject({ name: 'AbortError' })
    expect(cancel).toHaveBeenCalledOnce()
  })
})
