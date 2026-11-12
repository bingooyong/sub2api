import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ManualQualityError, runManualQualityProbe } from '../admin/manualQuality'
import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'
import axios from 'axios'

vi.mock('axios', () => ({ default: { post: vi.fn() } }))

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
  beforeEach(() => { localStorage.setItem('auth_token', 'admin-token'); vi.mocked(axios.post).mockReset() })
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
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toMatchObject(new ManualQualityError('http', 'HTTP 400: Unsupported test model'))
  })

  it('handles a final event without a blank-line terminator', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(responseFor(event({ type: 'content', text: '21' }) + 'data: {"type":"test_complete","success":true}')))
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).resolves.toBeUndefined()
  })

  it('rejects an already cancelled request before dispatch', async () => {
    const cancel = vi.fn()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream({ cancel }))))
    const controller = new AbortController()
    controller.abort()
    await expect(runManualQualityProbe(1, request, controller.signal, () => undefined)).rejects.toMatchObject({ name: 'AbortError' })
    expect(fetch).not.toHaveBeenCalled()
  })
})

function seedRefreshSession() {
  localStorage.setItem('refresh_token', 'refresh-old')
  localStorage.setItem('auth_user', JSON.stringify({ id: 7 }))
}
const tokens = { access_token: 'admin-new', refresh_token: 'refresh-new', expires_in: 3600, token_type: 'Bearer' }
const refreshResponse = { data: { code: 0, data: tokens } }
const success = () => responseFor(event({ type: 'content', text: '21' }) + event({ type: 'test_complete', success: true }))

describe('manual stream administrator session boundary', () => {
  beforeEach(() => {
    localStorage.setItem('auth_token', 'admin-token')
    seedRefreshSession()
    vi.mocked(axios.post).mockReset()
  })
  afterEach(() => { vi.unstubAllGlobals(); localStorage.clear(); vi.useRealTimers() })

  it('shares one real token refresh between concurrent 401s and preserves frozen requests', async () => {
    let resolve!: (value: typeof refreshResponse) => void
    vi.mocked(axios.post).mockImplementation(() => new Promise(done => { resolve = done }))
    const fetch = vi.fn().mockImplementation((_url, init) => Promise.resolve(init.headers.Authorization === 'Bearer admin-new' ? success() : new Response('', { status: 401 })))
    vi.stubGlobal('fetch', fetch)
    const first = runManualQualityProbe(1, request, new AbortController().signal, () => undefined)
    const second = runManualQualityProbe(2, request, new AbortController().signal, () => undefined)
    await vi.waitFor(() => expect(axios.post).toHaveBeenCalledOnce())
    resolve(refreshResponse)
    await Promise.all([first, second])
    expect(fetch).toHaveBeenCalledTimes(4)
    for (const [, init] of fetch.mock.calls) {
      expect(init.body).toBe(JSON.stringify(request))
      expect(init.headers[ADMIN_UI_REQUEST_HEADER]).toBe('1')
    }
  })

  it('retries a 401 at most once and never replays upstream SSE errors', async () => {
    vi.mocked(axios.post).mockResolvedValue(refreshResponse)
    const fetch = vi.fn().mockImplementation(() => Promise.resolve(new Response('', { status: 401 })))
    vi.stubGlobal('fetch', fetch)
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toMatchObject({ code: 'http', status: 401 })
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(axios.post).toHaveBeenCalledOnce()
    fetch.mockClear().mockResolvedValue(responseFor(event({ type: 'error', error: 'HTTP 401 upstream' })))
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toMatchObject({ code: 'failed' })
    expect(fetch).toHaveBeenCalledOnce()
    expect(axios.post).toHaveBeenCalledOnce()
  })

  it('aborts immediately while shared refresh continues for the other probe', async () => {
    let resolve!: (value: typeof refreshResponse) => void
    vi.mocked(axios.post).mockImplementation(() => new Promise(done => { resolve = done }))
    const fetch = vi.fn().mockImplementation((_url, init) => Promise.resolve(init.headers.Authorization === 'Bearer admin-new' ? success() : new Response('', { status: 401 })))
    vi.stubGlobal('fetch', fetch)
    const controller = new AbortController()
    const cancelled = runManualQualityProbe(1, request, controller.signal, () => undefined)
    const active = runManualQualityProbe(2, request, new AbortController().signal, () => undefined)
    await vi.waitFor(() => expect(axios.post).toHaveBeenCalledOnce())
    const rejection = expect(cancelled).rejects.toMatchObject({ name: 'AbortError' })
    controller.abort()
    await rejection
    resolve(refreshResponse)
    await active
    expect(fetch).toHaveBeenCalledTimes(3)
    expect(localStorage.getItem('auth_token')).toBe('admin-new')
  })

  it('does not retry when refresh fails', async () => {
    vi.useFakeTimers()
    vi.mocked(axios.post).mockRejectedValue(new Error('refresh denied'))
    const fetch = vi.fn().mockResolvedValue(new Response('', { status: 401 }))
    vi.stubGlobal('fetch', fetch)
    const rejection = expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toThrow('refresh denied')
    await vi.advanceTimersByTimeAsync(1100)
    await rejection
    expect(fetch).toHaveBeenCalledOnce()
  })

  it.each(['logout', 'replacement'])('does not replay after session %s while receiving the 401', async kind => {
    const fetch = vi.fn().mockImplementation(() => {
      if (kind === 'logout') localStorage.clear()
      else localStorage.setItem('auth_user', JSON.stringify({ id: 8 }))
      return Promise.resolve(new Response('', { status: 401 }))
    })
    vi.stubGlobal('fetch', fetch)
    await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toMatchObject({ code: 'http' })
    expect(fetch).toHaveBeenCalledOnce()
    expect(axios.post).not.toHaveBeenCalled()
  })

  it.each(['logout', 'replacement'])('does not replay after session %s during refresh', async kind => {
    vi.useFakeTimers()
    let resolve!: (value: typeof refreshResponse) => void
    vi.mocked(axios.post).mockImplementation(() => new Promise(done => { resolve = done }))
    const fetch = vi.fn().mockResolvedValue(new Response('', { status: 401 }))
    vi.stubGlobal('fetch', fetch)
    const pending = runManualQualityProbe(1, request, new AbortController().signal, () => undefined)
    await vi.advanceTimersByTimeAsync(0)
    expect(axios.post).toHaveBeenCalledOnce()
    if (kind === 'logout') localStorage.clear()
    else {
      localStorage.setItem('auth_user', JSON.stringify({ id: 8 }))
      localStorage.setItem('auth_token', 'other-access')
      localStorage.setItem('refresh_token', 'other-refresh')
    }
    const rejection = expect(pending).rejects.toThrow('Session changed during token refresh')
    resolve(refreshResponse)
    await vi.advanceTimersByTimeAsync(1100)
    await rejection
    expect(fetch).toHaveBeenCalledOnce()
    expect(localStorage.getItem('auth_token')).toBe(kind === 'logout' ? null : 'other-access')
  })

  it('opens compliance confirmation with metadata while preserving the HTTP error and avoiding retry', async () => {
    const metadata = { revision: 'v2', required: true }
    const listener = vi.fn()
    window.addEventListener('admin-compliance-required', listener)
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: 'ADMIN_COMPLIANCE_ACK_REQUIRED', message: 'Confirm terms', metadata }), { status: 423 }))
    vi.stubGlobal('fetch', fetch)
    try {
      await expect(runManualQualityProbe(1, request, new AbortController().signal, () => undefined)).rejects.toMatchObject({ code: 'http', status: 423, apiCode: 'ADMIN_COMPLIANCE_ACK_REQUIRED', metadata, message: 'HTTP 423: Confirm terms' })
      expect(listener).toHaveBeenCalledOnce()
      expect(listener.mock.calls[0]?.[0].detail).toEqual(metadata)
      expect(fetch).toHaveBeenCalledOnce()
      expect(axios.post).not.toHaveBeenCalled()
    } finally { window.removeEventListener('admin-compliance-required', listener) }
  })
})
