package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// pageUpdateURL builds the URL for a page-update view, e.g.
// /admin/cms?path=/pages/page-update&page_id=X&view=media.
func pageUpdateURL(view string) string {
	return adminURL("/pages/page-update") + "&page_id=" + seedPageID + "&view=" + view
}

// TestMediaUploadFlow verifies the media tab's Add Media modal uploads a
// file (multipart, field "files[]") that then appears in the media list.
func TestMediaUploadFlow(t *testing.T) {
	ctx := newTab(t)

	fileName := "e2e-upload-" + time.Now().Format("150405.000000") + ".txt"
	tmpFile := filepath.Join(t.TempDir(), fileName)
	if err := os.WriteFile(tmpFile, []byte("e2e media upload content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Wait for the Vue media app to mount: the "Add" .btn-primary button
	// only exists after the template is rendered.
	err := chromedp.Run(ctx,
		chromedp.Navigate(pageUpdateURL("media")),
		chromedp.WaitReady(`#page-media-app button.btn-primary`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("media view did not load: %v", err)
	}

	// Open the Add Media modal, then drive the hidden file input via
	// CDP's DOM.setFileInputFiles — no real file picker needed.
	err = chromedp.Run(ctx,
		chromedp.Click(`#page-media-app button.btn-primary`, chromedp.ByQuery),
		chromedp.WaitReady(`.modal input[type="file"]`, chromedp.ByQuery),
		chromedp.SetUploadFiles(`.modal input[type="file"]`, []string{tmpFile}),
		// Upload fires on the input's change event; poll for the card.
		chromedp.WaitReady(`.media-file-card`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("media upload flow failed: %v", err)
	}

	var body string
	if err := chromedp.Run(ctx, chromedp.Text("body", &body)); err != nil {
		t.Fatalf("reading media view failed: %v", err)
	}
	if !strings.Contains(body, fileName) {
		t.Fatalf("expected uploaded file %q in media list", fileName)
	}
}
