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

  const csp = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: blob:; media-src data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'">`
  // A model can put executable markup before its declared head. Install the
  // policy in a trusted head before parsing any of that markup; the HTML parser
  // merges the later html/body attributes and retains inline animation content.
  html = html.replace(/^<!doctype\s+html\b[^>]*>/i, '')
  return `<!doctype html><html><head>${csp}<meta charset="utf-8"></head>${html}</html>`
}
