package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/dracory/cmsstore"
)

// createSite seeds a site directly in the shared store and returns
// its ID.
func createSite(t *testing.T, name, status string) string {
	t.Helper()
	site := cmsstore.NewSite()
	site.SetName(name)
	site.SetStatus(status)
	if err := testStore.SiteCreate(context.Background(), site); err != nil {
		t.Fatalf("failed to seed site %q: %v", name, err)
	}
	return site.ID()
}

// siteInStore reports whether a site matching name exists (excluding
// soft-deleted).
func siteInStore(name string) bool {
	sites, err := testStore.SiteList(context.Background(),
		cmsstore.SiteQuery().SetNameLike(name))
	return err == nil && len(sites) > 0
}

// siteSoftDeleted reports whether the site is soft-deleted.
// Delete is a SiteSoftDelete — the row is excluded from default
// SiteList queries, so SoftDeleteIncluded is required to see it.
func siteSoftDeleted(siteID string) bool {
	sites, err := testStore.SiteList(context.Background(),
		cmsstore.SiteQuery().SetID(siteID).SetSoftDeletedIncluded(true).SetLimit(1))
	return err == nil && len(sites) > 0 && sites[0].IsSoftDeleted()
}

// TestSiteCreateFlow verifies the "New Site" htmx modal creates a site
// which then appears in the manager.
func TestSiteCreateFlow(t *testing.T) {
	ctx := newTab(t)
	name := "e2e-site-" + time.Now().Format("150405.000000")

	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/sites/site-manager")),
		chromedp.WaitReady(`table tbody tr`, chromedp.ByQuery),
		chromedp.Evaluate(clickButtonByTextJS(`button.btn-primary`, "New Site"), nil),
		chromedp.WaitReady(`#ModalPageCreate input[name="site_name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("create modal did not open: %v", err)
	}

	var ok bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(setInputJS(`#ModalPageCreate input[name="site_name"]`, name), &ok),
	)
	if err != nil || !ok {
		t.Fatalf("create form fill failed (ok=%v): %v", ok, err)
	}

	// Same headless caveat as blocks: the appended swal/reload scripts
	// don't execute — retry the click until the POST lands, then verify
	// server-side and re-navigate.
	if !clickUntil(ctx, `#ModalPageCreate .modal-footer .btn-primary`,
		func() bool { return siteInStore(name) }) {
		t.Fatalf("expected created site %q in store", name)
	}

	body := waitSelector(t, ctx, adminURL("/sites/site-manager"), `table tbody tr`)
	if !strings.Contains(body, name) {
		t.Fatalf("expected created site %q in manager", name)
	}
}

// TestSiteUpdateFlow verifies renaming a site via the settings view
// persists and shows in the manager.
func TestSiteUpdateFlow(t *testing.T) {
	ctx := newTab(t)
	oldName := "e2e-site-old-" + time.Now().Format("150405.000000")
	newName := "e2e-site-new-" + time.Now().Format("150405.000000")
	siteID := createSite(t, oldName, cmsstore.SITE_STATUS_ACTIVE)

	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/sites/site-update")+"&site_id="+siteID+"&view=settings"),
		chromedp.WaitReady(`#FormpageUpdate input[name="site_name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("site update page did not load: %v", err)
	}

	var ok bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(setInputJS(`#FormpageUpdate input[name="site_name"]`, newName), &ok),
		chromedp.Click(`button[hx-include="#FormpageUpdate"]`, chromedp.ByQuery),
		chromedp.Sleep(1500*time.Millisecond),
	)
	if err != nil || !ok {
		t.Fatalf("update save failed (ok=%v): %v", ok, err)
	}

	body := waitSelector(t, ctx, adminURL("/sites/site-manager"), `table tbody tr`)
	if !strings.Contains(body, newName) {
		t.Fatalf("expected renamed site %q in manager", newName)
	}
	if strings.Contains(body, oldName) {
		t.Fatalf("expected old name %q to be gone", oldName)
	}
}

// TestSiteDeleteFlow verifies deleting a site via its row's delete
// button and the htmx confirm modal (soft delete).
func TestSiteDeleteFlow(t *testing.T) {
	ctx := newTab(t)
	name := "e2e-sitedel-" + time.Now().Format("150405.000000")
	siteID := createSite(t, name, cmsstore.SITE_STATUS_ACTIVE)

	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/sites/site-manager")+"&filter_name="+name),
		chromedp.WaitReady(`table tbody tr`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("site manager did not load: %v", err)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(rowFindJS(name, `querySelector('.btn-danger').click(); true`), nil),
		chromedp.WaitReady(`#ModalSiteDelete`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("delete modal did not open: %v", err)
	}

	if !clickUntil(ctx, `#ModalSiteDelete .modal-footer .btn-primary`,
		func() bool { return siteSoftDeleted(siteID) }) {
		t.Fatalf("expected site %q to be soft-deleted in store", name)
	}

	// Soft-deleted rows are excluded from the default listing query.
	body := waitSelector(t, ctx, adminURL("/sites/site-manager"), `table`)
	if strings.Contains(body, name) {
		t.Fatalf("expected deleted site %q to be gone from manager", name)
	}
}

// TestSiteFilterByName verifies the filter_name param narrows the
// listing (the Filters modal submits the same params via GET).
func TestSiteFilterByName(t *testing.T) {
	ctx := newTab(t)
	suffix := time.Now().Format("150405.000000")
	matchName := "e2e-sflt-match-" + suffix
	otherName := "e2e-sflt-other-" + suffix
	createSite(t, matchName, cmsstore.SITE_STATUS_ACTIVE)
	createSite(t, otherName, cmsstore.SITE_STATUS_DRAFT)

	body := waitSelector(t, ctx,
		adminURL("/sites/site-manager")+"&filter_name="+matchName,
		`table tbody tr`)

	if !strings.Contains(body, matchName) {
		t.Fatalf("expected filtered site %q", matchName)
	}
	if strings.Contains(body, otherName) {
		t.Fatalf("expected %q to be filtered out", otherName)
	}
}

// TestSiteFilterByStatus verifies the filter_status param.
func TestSiteFilterByStatus(t *testing.T) {
	ctx := newTab(t)
	suffix := time.Now().Format("150405.000000")
	activeName := "e2e-sst-active-" + suffix
	draftName := "e2e-sst-draft-" + suffix
	createSite(t, activeName, cmsstore.SITE_STATUS_ACTIVE)
	createSite(t, draftName, cmsstore.SITE_STATUS_DRAFT)

	body := waitSelector(t, ctx,
		adminURL("/sites/site-manager")+"&filter_name=e2e-sst-&filter_status=draft",
		`table tbody tr`)

	if !strings.Contains(body, draftName) {
		t.Fatalf("expected draft site %q", draftName)
	}
	if strings.Contains(body, activeName) {
		t.Fatalf("expected active site %q to be filtered out", activeName)
	}
}

// TestSitePagination verifies real server-side pagination by clicking
// the rendered page-2 link: the site manager applies SetLimit/SetOffset
// and the pagination URL preserves filter_* and per_page params.
func TestSitePagination(t *testing.T) {
	ctx := newTab(t)
	createSite(t, "e2e-spg-a-"+time.Now().Format("150405.000000"), cmsstore.SITE_STATUS_ACTIVE)
	createSite(t, "e2e-spg-b-"+time.Now().Format("150405.000000"), cmsstore.SITE_STATUS_ACTIVE)

	url := adminURL("/sites/site-manager") + "&filter_name=e2e-spg-&per_page=1"

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady(`ul.pagination`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("pagination control did not render: %v", err)
	}

	// Pagination must be 1-based in both labels and hrefs — no 0 anywhere.
	var hasZero bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`!!document.querySelector('ul.pagination a[href$="page=0"], ul.pagination a[href*="page=0&"]')`,
		&hasZero)); err != nil {
		t.Fatalf("pagination zero check failed: %v", err)
	}
	if hasZero {
		t.Fatalf("pagination contains a page=0 link — must be 1-based")
	}

	firstRowJS := `(document.querySelector('table tbody tr')||{}).textContent || ''`
	var page1Row string
	if err := chromedp.Run(ctx, chromedp.Evaluate(firstRowJS, &page1Row)); err != nil {
		t.Fatalf("reading page 1 row failed: %v", err)
	}
	if !strings.Contains(page1Row, "e2e-spg-") {
		t.Fatalf("expected an e2e-spg- site on page 1, got %.80q", page1Row)
	}

	// Click the page 2 link (1-based externally, 0-based offset internally).
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(clickLinkByPageParamJS(`ul.pagination a`, 2), nil)); err != nil {
		t.Fatalf("pagination link click failed: %v", err)
	}

	// Poll until the navigated page renders a different first row.
	var page2Row string
	for range 20 {
		if err := chromedp.Run(ctx, chromedp.Evaluate(firstRowJS, &page2Row)); err == nil &&
			page2Row != "" && page2Row != page1Row {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if page2Row == "" || page2Row == page1Row {
		t.Fatalf("expected different row on page 2 (p1=%.60q p2=%.60q)", page1Row, page2Row)
	}
	if !strings.Contains(page2Row, "e2e-spg-") {
		t.Fatalf("expected an e2e-spg- site on page 2, got %.80q", page2Row)
	}
}
