import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'
import { buildApiUrl } from '../url'
import { refreshAuthTokens } from '../tokenRefresh'

export interface ManualQualityRequest {
  model_id: string
  prompt: string
  reasoning_effort: string
}

export type ManualQualityErrorCode = 'http' | 'noBody' | 'invalidStream' | 'incomplete' | 'empty' | 'failed'

export class ManualQualityError extends Error {
  constructor(
    public readonly code: ManualQualityErrorCode,
    message = '',
    public readonly status?: number,
    public readonly apiCode?: string,
    public readonly metadata?: unknown
  ) {
    super(message)
  }
}

/** Cancelling a probe must not cancel a refresh shared with other requests. */
function waitForRefresh<T>(pending: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const abort = () => reject(new DOMException('Aborted', 'AbortError'))
    signal.addEventListener('abort', abort, { once: true })
    if (signal.aborted) abort()
    void pending.then(resolve, reject).finally(() => signal.removeEventListener('abort', abort))
  })
}

/** Owns the account-test SSE contract; callers receive text and a terminal result. */
export async function runManualQualityProbe(
  accountID: number,
  request: Readonly<ManualQualityRequest>,
  signal: AbortSignal,
  onText: (text: string) => void
): Promise<void> {
  signal.throwIfAborted()
  const token = localStorage.getItem('auth_token')
  const sessionUser = localStorage.getItem('auth_user')
  const body = JSON.stringify(request)
  const send = (accessToken: string | null) => fetch(buildApiUrl(`/admin/accounts/${accountID}/pelican-test`), {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      [ADMIN_UI_REQUEST_HEADER]: '1',
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {})
    },
    body,
    signal
  })
  let response = await send(token)
  signal.throwIfAborted()
  if (response.status === 401 && localStorage.getItem('refresh_token') &&
      localStorage.getItem('auth_token') && localStorage.getItem('auth_user') === sessionUser) {
    await response.body?.cancel()
    signal.throwIfAborted()
    const tokens = await waitForRefresh(refreshAuthTokens({ failedAccessToken: token }), signal)
    signal.throwIfAborted()
    // A request from an older session must not be replayed as another user.
    if (localStorage.getItem('auth_user') !== sessionUser ||
        localStorage.getItem('auth_token') !== tokens.access_token ||
        localStorage.getItem('refresh_token') !== tokens.refresh_token) {
      throw new ManualQualityError('http', 'Session changed during token refresh', 401)
    }
    response = await send(tokens.access_token)
    signal.throwIfAborted()
  }
  if (!response.ok) {
    let message = `HTTP ${response.status}`
    let apiCode: string | undefined
    let metadata: unknown
    try {
      const errorBody: unknown = await response.json()
      if (errorBody && typeof errorBody === 'object') {
        if ('message' in errorBody && typeof errorBody.message === 'string') message += `: ${errorBody.message}`
        if ('code' in errorBody && typeof errorBody.code === 'string') apiCode = errorBody.code
        if ('metadata' in errorBody) metadata = errorBody.metadata
      }
    } catch { /* Keep the HTTP status for non-JSON proxy errors. */ }
    if (response.status === 423 && apiCode === 'ADMIN_COMPLIANCE_ACK_REQUIRED') {
      window.dispatchEvent(new CustomEvent('admin-compliance-required', { detail: metadata || {} }))
    }
    throw new ManualQualityError('http', message, response.status, apiCode, metadata)
  }
  const reader = response.body?.getReader()
  if (!reader) throw new ManualQualityError('noBody')
  const decoder = new TextDecoder()
  let buffer = '', data: string[] = [], completed = false, output = ''
  const dispatch = () => {
    if (!data.length) return
    const raw = data.join('\n')
    data = []
    if (raw === '[DONE]') return
    let event: unknown
    try { event = JSON.parse(raw) } catch { throw new ManualQualityError('invalidStream') }
    if (!event || typeof event !== 'object' || !('type' in event)) return
    if (event.type === 'content') {
      if (!('text' in event) || typeof event.text !== 'string') throw new ManualQualityError('invalidStream')
      output += event.text
      onText(event.text)
    } else if (event.type === 'test_complete') {
      if (!('success' in event) || event.success !== true) {
        throw new ManualQualityError('failed', 'error' in event && typeof event.error === 'string' ? event.error : '')
      }
      completed = true
    } else if (event.type === 'error') {
      throw new ManualQualityError('failed', 'error' in event && typeof event.error === 'string' ? event.error : '')
    }
  }
  const line = (value: string) => {
    const normalized = value.replace(/\r$/, '')
    if (!normalized) dispatch()
    else if (normalized.startsWith('data:')) data.push(normalized.slice(5).replace(/^ /, ''))
  }
  try {
    while (!completed) {
      if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
      const { done, value } = await reader.read()
      if (done) {
        buffer += decoder.decode()
        if (buffer) line(buffer)
        dispatch()
        break
      }
      buffer += decoder.decode(value, { stream: true })
      let newline: number
      while ((newline = buffer.indexOf('\n')) >= 0) {
        const value = buffer.slice(0, newline)
        buffer = buffer.slice(newline + 1)
        line(value)
        if (completed) break
      }
    }
    if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
    if (!completed) throw new ManualQualityError('incomplete')
    if (!output.trim()) throw new ManualQualityError('empty')
  } finally {
    await reader.cancel().catch(() => undefined)
    reader.releaseLock()
  }
}
