import { describe, expect, it } from 'vitest'
import { extractPelicanHtml } from '../pelicanHtml'

const parse = (raw: string) => new DOMParser().parseFromString(extractPelicanHtml(raw), 'text/html')

describe('pelican HTML preview policy', () => {
  it('installs the trusted no-script CSP before model markup and removes active content', () => {
    const document = parse(`<html><script id="early">location.href='/leak'</script><head>
      <meta http-equiv="refresh" content="0;url=/leak"><base href="https://example.com/">
      <link rel="prefetch" href="/leak"></head><body onload="alert(1)">
      <iframe src="/leak"></iframe><object data="/leak"></object><form action="/leak"><input></form>
      <svg onload="alert(1)"><a href="/leak" target="_self"><text>drawing</text></a></svg></body></html>`)
    expect(document.head.firstElementChild?.getAttribute('http-equiv')).toBe('Content-Security-Policy')
    expect(document.head.firstElementChild?.getAttribute('content')).toContain("script-src 'none'")
    expect(document.querySelector('script,base,link,iframe,object,form,input,[onload],[target]')).toBeNull()
    expect(document.querySelectorAll('meta')).toHaveLength(2)
    expect(document.querySelector('svg a')?.hasAttribute('href')).toBe(false)
  })

  it('preserves document styling, visual animation and body attributes', () => {
    const document = parse('<!doctype html><html lang="zh"><head><style>svg { width: 100%; animation: spin 2s infinite } @keyframes spin { to { opacity: .5 } }</style></head><body class="drawing"><svg><circle id="wheel" r="5"/><animate attributeName="x" from="0" to="5" dur="1s" repeatCount="indefinite" /></svg></body></html>')
    expect(document.compatMode).toBe('CSS1Compat')
    expect(document.documentElement.lang).toBe('zh')
    expect(document.body.className).toBe('drawing')
    expect(document.querySelector('style')?.textContent).toContain('@keyframes spin')
    expect(document.querySelector('svg animate')?.getAttribute('attributeName')).toBe('x')
    expect(document.head.firstElementChild?.getAttribute('content')).toContain("connect-src 'none'")
  })

  it('removes HTML/SVG links and animated URL changes while retaining internal artwork references', () => {
    const document = parse(`<html><body><a href="/leak" ping="/ping">click</a><svg>
      <a xlink:href="/leak"><text>click</text><animate attributeName="href" values="/leak"/></a>
      <set attributeName="xlink:href" to="/leak"/><animate attributeName="onload" to="alert(1)"/>
      <use href="#wheel"/><use xlink:href="https://example.com/art.svg#wheel"/>
      <circle id="wheel" fill="url(#paint)"/><defs><linearGradient id="paint"/></defs>
      </svg></body></html>`)
    expect(document.querySelector('a[href],a[ping],svg a[xlink\:href],animate,set')).toBeNull()
    expect(document.querySelector('use')?.getAttribute('href')).toBe('#wheel')
    expect(document.querySelectorAll('use')[1]?.hasAttribute('xlink:href')).toBe(false)
    expect(document.querySelector('circle')?.getAttribute('fill')).toBe('url(#paint)')
  })

  it.each([
    '@import "/leak"; body { color: red }',
    'body { background: url("/leak") }',
    'body { background: URL( /leak ) }',
    'body { background: image-set("/leak" 1x) }',
    'body { background: -webkit-image-set("/leak" 1x) }',
    String.raw`body { background: u\72l("/leak") }`,
    'body { background: u/**/rl("/leak") }'
  ])('drops resource-bearing or obfuscated CSS before iframe parsing: %s', css => {
    const document = parse(`<html style='${css}'><head><style>${css}</style></head><body style='${css}'><svg><circle fill='url(/leak)'/></svg><img srcset='/leak 1x' /><table background='/leak'></table></body></html>`)
    expect(document.querySelector('style')).toBeNull()
    expect(document.body.hasAttribute('style')).toBe(false)
    expect(document.documentElement.hasAttribute('style')).toBe(false)
    expect(document.querySelector('circle')?.hasAttribute('fill')).toBe(false)
    expect(document.querySelector('img')?.hasAttribute('srcset')).toBe(false)
    expect(document.querySelector('table')?.hasAttribute('background')).toBe(false)
  })

  it('wraps fenced standalone SVG in the same policy-first document', () => {
    const document = parse('```xml\n<svg xmlns="http://www.w3.org/2000/svg"><circle r="5" /></svg>\n```')
    expect(document.head.firstElementChild?.getAttribute('http-equiv')).toBe('Content-Security-Policy')
    expect(document.querySelector('svg circle')).not.toBeNull()
    expect(extractPelicanHtml('plain answer')).toBe('')
  })
})
