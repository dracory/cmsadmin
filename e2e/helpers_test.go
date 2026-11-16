package e2e

import (
	"context"
	"net/url"
	"testing"

	"github.com/chromedp/chromedp"
)

// adminURL builds a path-routed URL, e.g. /admin/cms?path=/pages/page-manager.
func adminURL(path string) string {
	return baseURL + adminPath + "?path=" + url.QueryEscape(path)
}

// waitTable navigates to url and waits until a data table has rendered
// at least two body rows, then returns the page's text content.
//
// Waits for the SECOND table row: the raw v-for template row may exist
// in the DOM before Vue mounts, so `table tr` alone can resolve while
// the table is still a template. A second row only exists after v-for
// has expanded with real data.
func waitTable(t *testing.T, ctx context.Context, url string) string {
	t.Helper()
	return waitSelector(t, ctx, url, "table tbody tr:nth-of-type(2)")
}

// waitSelector navigates to url, waits for sel to be ready, and returns
// the body's text content.
func waitSelector(t *testing.T, ctx context.Context, url, sel string) string {
	t.Helper()
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady(sel, chromedp.ByQuery),
		chromedp.Text("body", &body),
	)
	if err != nil {
		t.Fatalf("page %s failed: %v", url, err)
	}
	return body
}

// bodyText returns the page's current body text.
func bodyText(t *testing.T, ctx context.Context) string {
	t.Helper()
	var body string
	if err := chromedp.Run(ctx, chromedp.Text("body", &body)); err != nil {
		t.Fatalf("reading body failed: %v", err)
	}
	return body
}

// rowFindJS returns JS that finds the table row whose text contains name
// and evaluates inner against it. Row positions shift as tests mutate the
// shared store, so rows are located by text, not index.
func rowFindJS(name, inner string) string {
	return `[...document.querySelectorAll('table tbody tr')]` +
		`.find(tr => tr.textContent.includes('` + name + `'))` +
		`?.` + inner
}

// setInputJS returns JS that sets an input/select's value. Plain (non-Vue)
// server-rendered forms read .value at submit time, so no event dispatch
// is needed.
func setInputJS(sel, value string) string {
	return `(() => { const el = document.querySelector('` + sel + `');` +
		` if (!el) return false; el.value = '` + value + `'; return true })()`
}

// clickButtonByTextJS returns JS that clicks the first element matching
// sel whose text contains label.
func clickButtonByTextJS(sel, label string) string {
	return `(() => { const el = [...document.querySelectorAll('` + sel + `')]` +
		`.find(b => b.textContent.includes('` + label + `'));` +
		` if (!el) return false; el.click(); return true })()`
}
