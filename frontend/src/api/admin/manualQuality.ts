import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'
import { buildApiUrl } from '../url'

export interface ManualQualityRequest {
  model_id: string
  prompt: string
  reasoning_effort: string
}

export type ManualQualityErrorCode = 'http' | 'noBody' | 'invalidStream' | 'incomplete' | 'empty' | 'failed'

export class ManualQualityError extends Error {
  constructor(public readonly code: ManualQualityErrorCode, message = '') {
    super(message)
  }
}

/** Owns the account-test SSE contract; callers receive text and a terminal result. */
export async function runManualQualityProbe(
  accountID: number,
  request: Readonly<ManualQualityRequest>,
  signal: AbortSignal,
  onText: (text: string) => void
): Promise<void> {
  const token = localStorage.getItem('auth_token')
  const response = await fetch(buildApiUrl(`/admin/accounts/${accountID}/pelican-test`), {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      [ADMIN_UI_REQUEST_HEADER]: '1',
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify(request),
    signal
  })
  if (!response.ok) {
    let message = `HTTP ${response.status}`
    try {
      const body: unknown = await response.json()
      if (body && typeof body === 'object' && 'message' in body && typeof body.message === 'string') {
        message += `: ${body.message}`
      }
    } catch { /* Keep the HTTP status for non-JSON proxy errors. */ }
    throw new ManualQualityError('http', message)
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
