import DOMPurify from 'dompurify'

// Keep ordinary drawing CSS and local SVG paint references. Resource-bearing
// or escaped CSS is omitted before iframe parsing, since a browser may preload
// CSS resources before enforcing the preview's meta CSP. Raw output is retained.
function isPassiveDrawingCss(value: string): boolean {
  if (/[\\]|\/\*|@import|(?:image|image-set)\s*\(/i.test(value)) return false
  const withoutLocalReferences = value.replace(/url\(\s*(['"]?)#[a-z_][\w:.-]*\1\s*\)/gi, '')
  return !/url\s*\(/i.test(withoutLocalReferences)
}

export function extractPelicanHtml(raw: string): string {
  let html = raw.trim()
  const fenced = html.match(/```(?:html|xml)?\s*([\s\S]*?)```/i)
  if (fenced?.[1]) html = fenced[1].trim()

  const lower = html.toLowerCase()
  const start = Math.min(...['<!doctype html', '<html', '<svg'].map((marker) => {
    const index = lower.indexOf(marker)
    return index < 0 ? Number.MAX_SAFE_INTEGER : index
  }))
  if (start !== Number.MAX_SAFE_INTEGER) html = html.slice(start).trim()
  if (!/<(?:!doctype\s+html|html|svg)[\s>]/i.test(html)) return ''

  const end = html.toLowerCase().lastIndexOf('</html>')
  if (end >= 0) html = html.slice(0, end + '</html>'.length)
  if (!/<html[\s>]/i.test(html) && /<svg[\s>]/i.test(html)) {
    html = `<!doctype html><html><head><meta charset="utf-8"><title>Pelican test</title></head><body>${html}</body></html>`
  }

  // DOMPurify builds an inert document. Never attach generated markup to the
  // application DOM, and retain only passive drawing content before srcdoc.
  const documentElement = DOMPurify.sanitize(html, {
    WHOLE_DOCUMENT: true,
    RETURN_DOM: true,
    USE_PROFILES: { html: true, svg: true, svgFilters: true },
    ADD_TAGS: ['animate', 'set', 'use'],
    FORBID_TAGS: ['script', 'meta', 'base', 'link', 'iframe', 'frame', 'frameset', 'object', 'embed', 'form', 'input', 'button', 'select', 'textarea', 'video', 'audio', 'source'],
    FORBID_ATTR: ['target', 'ping', 'action', 'formaction', 'srcdoc', 'download', 'autofocus', 'srcset', 'background', 'poster', 'xml:base']
  })
  if (!(documentElement instanceof HTMLHtmlElement)) return ''
  const visualAnimationAttributes = new Set([
    'x', 'y', 'x1', 'x2', 'y1', 'y2', 'cx', 'cy', 'r', 'rx', 'ry', 'dx', 'dy',
    'width', 'height', 'd', 'points', 'transform', 'opacity', 'fill', 'fill-opacity',
    'stroke', 'stroke-width', 'stroke-opacity', 'stroke-dasharray', 'stroke-dashoffset',
    'offset', 'stop-color', 'stop-opacity', 'viewbox'
  ])
  for (const element of [documentElement, ...documentElement.querySelectorAll('*')]) {
    const tag = element.localName.toLowerCase()
    if (tag === 'style' && !isPassiveDrawingCss(element.textContent || '')) {
      element.remove()
      continue
    }
    if ((tag === 'animate' || tag === 'set') && !visualAnimationAttributes.has((element.getAttribute('attributeName') || '').toLowerCase())) {
      element.remove()
      continue
    }
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase()
      if (!isPassiveDrawingCss(attribute.value)) {
        element.removeAttribute(attribute.name)
        continue
      }
      if (name !== 'href' && name !== 'xlink:href' && name !== 'src') continue
      const value = attribute.value.trim()
      const internalReference = tag !== 'a' && value.startsWith('#')
      const imageData = (tag === 'image' || tag === 'img') && /^data:image\//i.test(value)
      if (!internalReference && !imageData) element.removeAttribute(attribute.name)
    }
  }
  const document = documentElement.ownerDocument
  const head = documentElement.querySelector('head')
  if (!head) return ''
  const csp = document.createElement('meta')
  csp.httpEquiv = 'Content-Security-Policy'
  csp.content = "default-src 'none'; img-src data:; media-src 'none'; style-src 'unsafe-inline'; script-src 'none'; font-src data:; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"
  const charset = document.createElement('meta')
  charset.setAttribute('charset', 'utf-8')
  head.prepend(csp, charset)
  return `<!doctype html>${documentElement.outerHTML}`
}
