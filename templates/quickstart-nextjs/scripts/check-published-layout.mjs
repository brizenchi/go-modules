// Offline layout acceptance for CMS archives. Build against the local preview
// API first; Chrome renders only temporary HTML and embedded build assets.
import assert from "node:assert/strict";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { spawnSync } from "node:child_process";

const build = resolve(process.argv[2] || ".next");
const browser = process.argv[3] || [
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/usr/bin/google-chrome",
  "/usr/bin/chromium"
].find(existsSync);
assert.ok(browser && existsSync(browser), "Supply the Chrome/Chromium executable as the second argument.");
const output = process.argv[4] ? resolve(process.argv[4]) : mkdtempSync(join(tmpdir(), "saas-content-layout-"));
mkdirSync(output, { recursive: true });
const pages = ["resources", "blog", "docs", "features", "use-cases", "compare"].flatMap(page => [page, `zh/${page}`]);

function offlinePage(page) {
  let html = readFileSync(join(build, "server/app", `${page}.html`), "utf8");
  const css = [...html.matchAll(/<link\b[^>]*>/g)]
    .filter(([tag]) => tag.includes('rel="stylesheet"'))
    .map(([tag]) => {
      const url = tag.match(/href="([^"]+)"/)?.[1];
      assert.ok(url?.startsWith("/_next/"), "Styles must be local build assets.");
      const file = join(build, url.slice("/_next/".length).split("?")[0]);
      return readFileSync(file, "utf8").replace(/url\(([^)]+)\)/g, (match, raw) => {
        const asset = raw.replace(/^["']|["']$/g, "");
        if (asset.startsWith("data:")) return match;
        const target = asset.startsWith("/_next/") ? join(build, asset.slice("/_next/".length)) : resolve(dirname(file), asset);
        assert.ok(/\.woff2?$/.test(target), `Unexpected CSS asset: ${asset}`);
        return `url(data:font/woff2;base64,${readFileSync(target).toString("base64")})`;
      });
    }).join("\n");
  html = html.replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, "").replace(/<link\b[^>]*>/g, "");
  const measurement = `<script>document.fonts.ready.then(() => {
    const rows = [...document.querySelectorAll('main article')].filter(row => row.querySelector('h2')).map(row => ({
      title: row.querySelector('h2').innerText,
      width: row.getBoundingClientRect().width,
      heading: row.querySelector('h2').getBoundingClientRect().width,
      summary: row.querySelector('p')?.getBoundingClientRect().width || 0
    }));
    const input = document.querySelector('#published-search');
    const list = document.querySelector('main article')?.closest('li')?.parentElement;
    const pre = document.createElement('pre'); pre.id = 'layout-result';
    const result = { viewport: innerWidth, page: document.documentElement.scrollWidth, input: input?.getBoundingClientRect().width || 0, columns: list ? getComputedStyle(list).gridTemplateColumns.split(' ').length : 0, rows };
    pre.textContent = JSON.stringify(result);
    pre.hidden = true; document.body.append(pre);
    if (window.parent !== window) window.parent.postMessage({ type: 'layout-result', result }, '*');
  });</script>`;
  return html.replace("</head>", `<style>${css}</style></head>`).replace("</body>", `${measurement}</body>`);
}

for (const page of pages) {
  const source = offlinePage(page);
  for (const width of [1440, 768, 390]) {
    const profile = mkdtempSync(join(tmpdir(), "saas-layout-browser-"));
    try {
      const fixture = join(output, `${page.replaceAll("/", "-")}-${width}.html`);
      // A fixed iframe gives a real 390px CSS viewport even where Chrome's
      // native headless window has a 500px minimum. No device emulation needed.
      const escaped = source.replaceAll("&", "&amp;").replaceAll('"', "&quot;");
      writeFileSync(fixture, `<!doctype html><html><head><style>body{margin:0;overflow:hidden}iframe{display:block;border:0}</style><script>
        window.addEventListener('message', event => { if(event.data?.type !== 'layout-result') return;
          const pre=document.createElement('pre');pre.id='layout-result';pre.hidden=true;pre.textContent=JSON.stringify(event.data.result);document.body.append(pre);
        });</script></head><body><iframe title="Layout fixture" width="${width}" height="1200" srcdoc="${escaped}"></iframe></body></html>`);
      const result = spawnSync(browser, ["--headless", "--no-first-run", "--no-default-browser-check", "--disable-background-networking",
        `--user-data-dir=${profile}`, `--window-size=${width},1200`, "--virtual-time-budget=1200", "--dump-dom",
        `--screenshot=${join(output, `${page.replaceAll("/", "-")}-${width}.png`)}`, pathToFileURL(fixture).href],
      { encoding: "utf8", timeout: 30000, maxBuffer: 20 * 1024 * 1024 });
      assert.equal(result.status, 0, result.error?.message || result.stderr.slice(-500));
      const raw = result.stdout.match(/<pre id="layout-result"[^>]*>([^<]+)<\/pre>/)?.[1];
      assert.ok(raw, `${page}: browser returned layout measurements`);
      const layout = JSON.parse(raw);
      assert.equal(layout.viewport, width, `${page}: requested CSS viewport`);
      assert.ok(layout.rows.length > 0, `${page}: build must contain published CMS articles`);
      assert.equal(layout.columns, width > 1000 ? 3 : width > 640 ? 2 : 1, `${page}/${width}: responsive card columns`);
      assert.ok(layout.page <= layout.viewport + 1, `${page}/${width}: no horizontal overflow`);
      assert.ok(layout.input >= Math.min(280, width - 48), `${page}/${width}: usable search width`);
      for (const row of layout.rows) {
        assert.ok(row.heading >= Math.min(480, row.width * 0.7), `${page}/${width}: title squeezed into ${row.heading}px: ${row.title}`);
        assert.ok(row.summary >= Math.min(420, row.width * 0.65), `${page}/${width}: summary squeezed into ${row.summary}px: ${row.title}`);
      }
      console.log(`${page} ${width}px: ${layout.rows.length} articles, first title ${Math.round(layout.rows[0].heading)}px, no overflow`);
    } finally {
      rmSync(profile, { recursive: true, force: true });
    }
  }
}
console.log(`Published layout checks passed: ${pages.length} pages × 3 viewports. Screenshots: ${output}`);
