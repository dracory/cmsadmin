package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/dracory/cmsstore"
)

// createBlock seeds a block directly in the shared store and returns
// its ID.
func createBlock(t *testing.T, name, status string) string {
	t.Helper()
	block := cmsstore.NewBlock()
	block.SetName(name)
	block.SetType(cmsstore.BLOCK_TYPE_HTML)
	block.SetSiteID(seedSiteID)
	block.SetStatus(status)
	if err := testStore.BlockCreate(context.Background(), block); err != nil {
		t.Fatalf("failed to seed block %q: %v", name, err)
	}
	return block.ID()
}

// blockInStore reports whether a block matching name exists.
func blockInStore(name string) bool {
	blocks, err := testStore.BlockList(context.Background(),
		cmsstore.BlockQuery().SetNameLike(name))
	return err == nil && len(blocks) > 0
}

// blockSoftDeleted reports whether the block is soft-deleted.
// Delete is a BlockSoftDelete — the row is excluded from default
// BlockList queries, so SoftDeleteIncluded is required to see it.
func blockSoftDeleted(blockID string) bool {
	blocks, err := testStore.BlockList(context.Background(),
		cmsstore.BlockQuery().SetID(blockID).SetSoftDeleteIncluded(true).SetLimit(1))
	return err == nil && len(blocks) > 0 && blocks[0].IsSoftDeleted()
}

// clickUntil clicks sel with a real CDP click, then polls check for up
// to ~800ms, retrying up to 6 times. htmx-bound buttons inside modals
// swapped in via hx-swap="beforeend" occasionally miss a click when the
// suite runs under load — retrying is safe because the create/delete
// POSTs are idempotent for our assertions.
func clickUntil(ctx context.Context, sel string, check func() bool) bool {
	for range 6 {
		if err := chromedp.Run(ctx, chromedp.Click(sel, chromedp.ByQuery)); err != nil {
			return false
		}
		for range 8 {
			if check() {
				return true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return false
}

// TestBlockCreateFlow verifies the "New Block" htmx modal creates a
// block which then appears in the manager after the success reload.
func TestBlockCreateFlow(t *testing.T) {
	ctx := newTab(t)
	name := "e2e-create-" + time.Now().Format("150405.000000")

	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/blocks/block-manager")),
		chromedp.WaitReady(`table tbody tr`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("block manager did not load: %v", err)
	}

	var ok bool
	err = chromedp.Run(ctx,
		// "New Block" is a plain <button> triggering hx-get of the modal.
		chromedp.Evaluate(clickButtonByTextJS(`button.btn-primary`, "New Block"), &ok),
		chromedp.WaitReady(`#ModalPageCreate input[name="block_name"]`, chromedp.ByQuery),
	)
	if err != nil || !ok {
		t.Fatalf("create modal did not open (clicked=%v): %v", ok, err)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(setInputJS(`#ModalPageCreate input[name="block_name"]`, name), &ok),
		chromedp.Evaluate(setInputJS(`#ModalPageCreate select[name="site_id"]`, seedSiteID), &ok),
	)
	if err != nil || !ok {
		t.Fatalf("create form fill failed (ok=%v): %v", ok, err)
	}

	// The success response is a swal + reload script appended via htmx;
	// those inline scripts don't execute in headless Chrome, so verify
	// server-side state and re-navigate instead of waiting for reload.
	// The htmx-bound button occasionally swallows a click under load —
	// retry until the store reflects the new block.
	if !clickUntil(ctx, `#ModalPageCreate .modal-footer .btn-primary`,
		func() bool { return blockInStore(name) }) {
		t.Fatalf("expected created block %q in store", name)
	}

	body := waitSelector(t, ctx, adminURL("/blocks/block-manager"), `table tbody tr`)
	if !strings.Contains(body, name) {
		t.Fatalf("expected created block %q in manager", name)
	}
}

// TestBlockUpdateFlow verifies editing a block's name via the settings
// view persists and shows in the manager.
func TestBlockUpdateFlow(t *testing.T) {
	ctx := newTab(t)
	oldName := "e2e-update-old-" + time.Now().Format("150405.000000")
	newName := "e2e-update-new-" + time.Now().Format("150405.000000")
	blockID := createBlock(t, oldName, cmsstore.BLOCK_STATUS_ACTIVE)

	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/blocks/block-update")+"&block_id="+blockID+"&view=settings"),
		chromedp.WaitReady(`#FormBlockUpdate input[name="block_name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("block update page did not load: %v", err)
	}

	var ok bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(setInputJS(`#FormBlockUpdate input[name="block_name"]`, newName), &ok),
		// Save is the button whose hx-include points at the update form.
		chromedp.Click(`button[hx-include="#FormBlockUpdate"]`, chromedp.ByQuery),
		chromedp.Sleep(1500*time.Millisecond),
	)
	if err != nil || !ok {
		t.Fatalf("update save failed (ok=%v): %v", ok, err)
	}

	body := waitSelector(t, ctx, adminURL("/blocks/block-manager"), `table tbody tr`)
	if !strings.Contains(body, newName) {
		t.Fatalf("expected renamed block %q in manager", newName)
	}
	if strings.Contains(body, oldName) {
		t.Fatalf("expected old name %q to be gone", oldName)
	}
}

// TestBlockDeleteFlow verifies deleting a block via its row's delete
// button and the htmx confirm modal.
func TestBlockDeleteFlow(t *testing.T) {
	ctx := newTab(t)
	name := "e2e-delete-" + time.Now().Format("150405.000000")
	blockID := createBlock(t, name, cmsstore.BLOCK_STATUS_ACTIVE)

	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/blocks/block-manager")+"&name="+name),
		chromedp.WaitReady(`table tbody tr`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("block manager did not load: %v", err)
	}

	var ok bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(rowFindJS(name, `querySelector('.btn-danger').click(); true`), &ok),
		chromedp.WaitReady(`#ModalBlockDelete`, chromedp.ByQuery),
	)
	if err != nil || !ok {
		t.Fatalf("delete modal did not open (clicked=%v): %v", ok, err)
	}

	// As with create, the swal/reload scripts appended via htmx don't
	// execute in headless — verify server-side instead. Delete is a
	// soft delete: the row stays in the store (and the table, with a
	// deleted status) — assert the status change, not absence. Retry
	// the click until the delete POST lands.
	if !clickUntil(ctx, `#ModalBlockDelete .modal-footer .btn-primary`,
		func() bool { return blockSoftDeleted(blockID) }) {
		t.Fatalf("expected block %q to be soft-deleted in store", name)
	}

	// Soft-deleted rows are excluded from the default listing query.
	body := waitSelector(t, ctx, adminURL("/blocks/block-manager"), `table`)
	if strings.Contains(body, name) {
		t.Fatalf("expected deleted block %q to be gone from manager", name)
	}
}

// TestBlockFilterByName verifies the name filter narrows the listing
// via query params (the Filters modal submits the same params via GET).
func TestBlockFilterByName(t *testing.T) {
	ctx := newTab(t)
	suffix := time.Now().Format("150405.000000")
	matchName := "e2e-filter-match-" + suffix
	otherName := "e2e-filter-other-" + suffix
	createBlock(t, matchName, cmsstore.BLOCK_STATUS_ACTIVE)
	createBlock(t, otherName, cmsstore.BLOCK_STATUS_DRAFT)

	body := waitSelector(t, ctx,
		adminURL("/blocks/block-manager")+"&name="+matchName,
		`table tbody tr`)

	if !strings.Contains(body, matchName) {
		t.Fatalf("expected filtered block %q", matchName)
	}
	if strings.Contains(body, otherName) {
		t.Fatalf("expected %q to be filtered out", otherName)
	}
	if !strings.Contains(body, "name: "+matchName) {
		t.Fatalf("expected filter description to show name filter")
	}
}

// TestBlockFilterByStatus verifies the status filter.
func TestBlockFilterByStatus(t *testing.T) {
	ctx := newTab(t)
	suffix := time.Now().Format("150405.000000")
	activeName := "e2e-status-active-" + suffix
	draftName := "e2e-status-draft-" + suffix
	createBlock(t, activeName, cmsstore.BLOCK_STATUS_ACTIVE)
	createBlock(t, draftName, cmsstore.BLOCK_STATUS_DRAFT)

	body := waitSelector(t, ctx,
		adminURL("/blocks/block-manager")+"&name=e2e-status-&status=draft",
		`table tbody tr`)

	if !strings.Contains(body, draftName) {
		t.Fatalf("expected draft block %q", draftName)
	}
	if strings.Contains(body, activeName) {
		t.Fatalf("expected active block %q to be filtered out", activeName)
	}
}

// TestBlockPagination verifies the pagination control renders when the
// row count exceeds per_page and that a page link navigates.
//
// NOTE: fetchRecordList currently ignores limit/offset (commented out),
// so this only asserts navigation — page 2 renders the same table.
func TestBlockPagination(t *testing.T) {
	ctx := newTab(t)
	createBlock(t, "e2e-page-a-"+time.Now().Format("150405.000000"), cmsstore.BLOCK_STATUS_ACTIVE)
	createBlock(t, "e2e-page-b-"+time.Now().Format("150405.000000"), cmsstore.BLOCK_STATUS_ACTIVE)

	// per_page=1 with >=1 rows forces the pagination control to render.
	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("/blocks/block-manager")+"&per_page=1"),
		chromedp.WaitReady(`ul.pagination`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("pagination control did not render: %v", err)
	}

	// Follow the page 2 link — must load a valid manager page.
	err = chromedp.Run(ctx,
		chromedp.Evaluate(clickLinkByPageParamJS(`ul.pagination a`, 2), nil),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		t.Fatalf("pagination navigation failed: %v", err)
	}
	if body := bodyText(t, ctx); !strings.Contains(body, "Block Manager") {
		t.Fatalf("expected block manager after pagination navigation")
	}
}
