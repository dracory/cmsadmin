# e2e

End-to-end tests that drive the cmsadmin panel in a real headless Chrome
via [chromedp](https://github.com/chromedp/chromedp) (CDP). Unlike go-rod,
chromedp has no `leakless.exe` helper, so it avoids the Windows Defender
false positive entirely.

## How it works

- `TestMain` mounts the admin on `httptest.NewServer` (in-process, random
  port) backed by a seeded in-memory SQLite store via
  `cmsstore/testutils.InitStore` — no binary build needed. One site, two
  pages, and one template are seeded.
- One shared browser is launched via `chromedp.NewExecAllocator` for the
  whole run; each test opens a fresh tab with a 30s deadline (`newTab`).
- Chrome binary resolution order: `CHROME_BIN` env → Windows Playwright
  `chromium_headless_shell` → WSL-side Windows shell → Linux Playwright
  shell → chromedp's default lookup (Chrome on PATH / install dirs).

## Run

```bash
go test -v ./e2e/     # or: task e2e
```

Note: on WSL, run tests with the Windows Go toolchain (`go.exe`) —
Windows-side Chrome binds 127.0.0.1, which WSL cannot reach (no
WSL→Windows localhost forwarding). The Linux Playwright shell also
requires system libs (libnspr4 etc.) that may be missing.

## Coverage

- home — dashboard tiles render
- `/pages/page-manager` — Vue-driven page list loads seeded pages
- `/sites/site-manager` — server-rendered site table lists seeded site
- `/blocks/block-manager` — block table lists seeded block
- `/translations/translation-manager` — translation table lists seeded translation

## Gotchas learned

- Prefer `chromedp.WaitReady` over `chromedp.WaitVisible`: on some pages
  `WaitVisible` polls forever even though the DOM is fully loaded.
- Routing is query-param based: `?path=/pages/page-manager`
  (see `shared/consts.go`). Path values must be URL-escaped — use
  `adminURL(path)` in `helpers_test.go`.
- The page manager renders via Vue after an AJAX fetch — wait for the
  second `tbody tr` before asserting (the v-for template row exists in
  the DOM before Vue mounts).
- Tests share one store/browser across the whole run — don't assert on
  fixed row indices and don't delete seeded fixtures.
- chromedp v0.23 rewrote the API (generic `Run[T]`); this suite pins the
  stable v0.14 line.
