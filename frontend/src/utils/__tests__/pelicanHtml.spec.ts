import { describe, expect, it } from 'vitest'
import { extractPelicanHtml } from '../pelicanHtml'

describe('pelican HTML preview policy', () => {
  it('places the trusted CSP before model scripts even when markup precedes an explicit head', () => {
    const html = extractPelicanHtml('<html><script id="early">window.preview = true</script><head><title>Drawing</title></head><body><svg></svg></body></html>')
    const document = new DOMParser().parseFromString(html, 'text/html')
    expect(document.head.firstElementChild?.getAttribute('http-equiv')).toBe('Content-Security-Policy')
    expect(html.indexOf('Content-Security-Policy')).toBeLessThan(html.indexOf('<script'))
    expect(document.querySelector('#early')).not.toBeNull()
  })

  it('preserves document styling, inline animation and body attributes after installing the policy', () => {
    const html = extractPelicanHtml('<!doctype html><html lang="zh"><head><style>svg { width: 100% }</style></head><body class="drawing"><svg><animate attributeName="x" /></svg><script>window.animateDrawing = true</script></body></html>')
    const document = new DOMParser().parseFromString(html, 'text/html')
    expect(document.compatMode).toBe('CSS1Compat')
    expect(document.documentElement.lang).toBe('zh')
    expect(document.body.className).toBe('drawing')
    expect(document.querySelector('style')?.textContent).toContain('width: 100%')
    expect(document.querySelector('svg animate')).not.toBeNull()
    expect(document.querySelector('script')?.textContent).toContain('animateDrawing')
    expect(document.head.firstElementChild?.getAttribute('content')).toContain("connect-src 'none'")
  })

  it('wraps fenced standalone SVG in the same policy-first document', () => {
    const html = extractPelicanHtml('```xml\n<svg xmlns="http://www.w3.org/2000/svg"><circle r="5" /></svg>\n```')
    const document = new DOMParser().parseFromString(html, 'text/html')
    expect(document.head.firstElementChild?.getAttribute('http-equiv')).toBe('Content-Security-Policy')
    expect(document.querySelector('svg circle')).not.toBeNull()
  })
})
