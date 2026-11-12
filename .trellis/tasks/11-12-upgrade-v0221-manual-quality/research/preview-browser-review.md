# Generated preview browser review

Date: 2026-11-12

## Question and method

The built-app mobile screenshot showed a blank completed preview after stop and resize. Verify whether the policy-first `extractPelicanHtml` change prevents ordinary generated HTML/SVG or inline styles from rendering.

Transpiled the actual `frontend/src/utils/pelicanHtml.ts` with the project's installed TypeScript compiler. Rendered its returned strings in an isolated Chromium page, using the production iframe attributes `sandbox="allow-scripts"` and `referrerpolicy="no-referrer"`. No application server, real account or Go build was used.

The three fixtures were:

1. The exact full HTML/SVG bicycle markup from `output/playwright/manual-quality-fixture.js`.
2. A standalone SVG with a background rectangle and circle.
3. A full document with an inline script before its explicit head, a styled SVG and a test external fetch.

Waited for each frame's SVG to be visible and two animation frames before screenshots. The mobile viewport was 440×1500; a desktop screenshot used 1600×1200.

## Observed evidence

- The exact bicycle fixture rendered visibly. Its SVG measured **416×300 pixels**, at x=0/y=0 inside the frame.
- Its original inline style remained effective: body background was **`rgb(188, 232, 244)`** and the SVG filled the frame height.
- The standalone SVG rendered visibly and measured 400×288 pixels.
- The early-script fixture rendered visibly, retained its `drawing` body class and `rgb(213, 242, 222)` background, and its inline script executed.
- All three parsed documents had the trusted CSP as the first head element.
- The deliberately attempted `https://csp-probe.invalid/blocked` fetch generated **zero observed network requests**.
- The page emitted **zero uncaught page errors**. The intentional fetch generated expected CSP-denial console messages; those are distinct from page errors.

Artifacts retained under the ignored output directory:

- `output/playwright/pelican-helper-mobile-render.png`
- `output/playwright/pelican-helper-desktop-render.png`
- `output/playwright/pelican-browser-check.log`
- `output/playwright/pelican-browser-check.js`

## Conclusion and limits

The policy-first helper correctly renders the tested ordinary full HTML, SVG, inline styling and script content in real Chromium. No additional helper change was made during this verification.

This isolated check does not establish why the earlier application screenshot was blank. Loading, resize, cancellation, component lifetime and screenshot timing must be distinguished with a final application check; no cause is inferred here.

The isolated browser was closed and its temporary `frontend/.playwright-cli` traces were removed after verification.

## Production application follow-up

Checked the production preview at `http://127.0.0.1:3107` with the fake admin/account/SSE fixture in `output/playwright/manual-quality-fixture.js`; no live account requests were made. Started the manual comparison, stopped it after completed results appeared, scrolled the iframe into view, then closed/reopened the panel and resized to a 390×844 mobile viewport. Waited for animation frames before capture.

- Card states were `success`, `success`, `cancelled`.
- Desktop SVG measured **270×202**; mobile SVG measured **288×216** after stop/reopen/resize.
- Both retained body background `rgb(188, 232, 244)`, visible visibility, opacity 1 and original SVG styling.
- Element screenshots visibly contained the bicycle drawing. Pillow counted **52,665 nonwhite pixels / 515 distinct colors** on desktop and **60,055 nonwhite pixels / 540 distinct colors** on mobile.
- Artifacts: `output/playwright/pelican-prod-desktop-iframe.png`, `output/playwright/pelican-prod-mobile-iframe.png`, `output/playwright/pelican-prod-mobile-viewport.png`, `output/playwright/pelican-prod-before-scroll.png`, and `output/playwright/pelican-prod-viewport-check.{js,log}`.

No preview regression was reproduced in the actual production application, and no further product code change was needed. Earlier full-page white screenshots do not establish a rendering failure; their precise capture/painting cause remains unisolated.
