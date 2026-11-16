// Package e2e contains end-to-end tests that drive the cmsadmin panel
// with a real headless Chrome via chromedp (CDP). The admin is served
// in-process by httptest on a random port — no binary build needed.
package e2e

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/dracory/cmsadmin"
	"github.com/dracory/cmsstore"
	"github.com/dracory/cmsstore/testutils"
	_ "modernc.org/sqlite"
)

var (
	baseURL    string
	allocCtx   context.Context
	testStore  cmsstore.StoreInterface
	seedSiteID string
	seedPage   string
	seedPageID string
	seedBlock  = "e2e-seed-block"
)

const adminPath = "/admin/cms"

func TestMain(m *testing.M) {
	// In-process admin server backed by a seeded in-memory SQLite store.
	// Shared-cache DSN is required: plain ":memory:" gives each pooled
	// connection its own private DB, so a second connection opened under
	// concurrent e2e load would see an empty schema and fail randomly.
	// Same options as cmsstore/testutils.InitStore.
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		panic("e2e: failed to open db: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		panic("e2e: failed to ping db: " + err.Error())
	}

	store, err := cmsstore.NewStore(cmsstore.NewStoreOptions{
		DB:                         db,
		BlockTableName:             "block_table",
		PageTableName:              "page_table",
		SiteTableName:              "site_table",
		TemplateTableName:          "template_table",
		MenusEnabled:               true,
		MenuTableName:              "menu_table",
		MenuItemTableName:          "menu_item_table",
		TranslationsEnabled:        true,
		TranslationTableName:       "translation_table",
		TranslationLanguageDefault: "en",
		VersioningEnabled:          true,
		VersioningTableName:        "versioning_table",
		MediaEnabled:               true,
		MediaTableName:             "media_table",
		AutomigrateEnabled:         true,
	})
	if err != nil {
		panic("e2e: failed to init store: " + err.Error())
	}
	testStore = store

	seedStore(store)

	panel, err := cmsadmin.New(cmsadmin.AdminOptions{
		Store:        store,
		Logger:       slog.New(slog.NewTextHandler(os.Stdout, nil)),
		AdminHomeURL: "/admin",
	})
	if err != nil {
		panic("e2e: failed to create cmsadmin: " + err.Error())
	}

	mux := http.NewServeMux()
	mux.Handle(adminPath, http.HandlerFunc(panel.Handle))
	srv := httptest.NewServer(mux)
	baseURL = srv.URL

	// One shared browser process for the entire test run.
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Headless)
	if bin := chromeBin(); bin != "" {
		opts = append(opts, chromedp.ExecPath(bin))
	}
	var cancel context.CancelFunc
	allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)

	code := m.Run()

	cancel()
	srv.Close()
	os.Exit(code)
}

// seedStore populates the store with a site, pages and a template so the
// manager pages have rows to render.
func seedStore(store cmsstore.StoreInterface) {
	site, err := testutils.SeedSite(store, "SITE_01")
	if err != nil {
		panic("e2e: failed to seed site: " + err.Error())
	}
	seedSiteID = site.ID()

	seedPage = "e2e-seed-page"
	page, err := testutils.SeedPage(store, seedSiteID, seedPage)
	if err != nil {
		panic("e2e: failed to seed page: " + err.Error())
	}
	seedPageID = page.ID()
	if _, err := testutils.SeedPage(store, seedSiteID, "e2e-seed-page-2"); err != nil {
		panic("e2e: failed to seed page 2: " + err.Error())
	}
	if _, err := testutils.SeedTemplate(store, seedSiteID, "_01"); err != nil {
		panic("e2e: failed to seed template: " + err.Error())
	}
	if _, err := testutils.SeedTranslation(store, seedSiteID, "_01"); err != nil {
		panic("e2e: failed to seed translation: " + err.Error())
	}

	block := cmsstore.NewBlock()
	block.SetName(seedBlock)
	block.SetType(cmsstore.BLOCK_TYPE_HTML)
	block.SetSiteID(seedSiteID)
	block.SetStatus(cmsstore.BLOCK_STATUS_ACTIVE)
	if err := store.BlockCreate(context.Background(), block); err != nil {
		panic("e2e: failed to seed block: " + err.Error())
	}
}

// newTab opens a fresh tab in the shared browser and returns its
// context. Each tab gets a 30s deadline; cleanup closes the tab.
func newTab(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := chromedp.NewContext(allocCtx)
	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() { cancelTimeout(); cancel() })
	return ctx
}

// chromeBin resolves a headless Chromium binary. Order: CHROME_BIN env,
// Playwright's downloaded headless shell, empty (chromedp finds Chrome
// on PATH / standard install locations).
func chromeBin() string {
	if b := os.Getenv("CHROME_BIN"); b != "" {
		return b
	}
	patterns := []string{
		// Windows
		filepath.Join(os.Getenv("LOCALAPPDATA"),
			"ms-playwright/chromium_headless_shell-*/chrome-headless-shell-win64/chrome-headless-shell.exe"),
		// WSL: Windows-side Playwright shell via interop
		"/mnt/*/Users/*/AppData/Local/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-win64/chrome-headless-shell.exe",
		// Linux
		filepath.Join(os.Getenv("HOME"),
			".cache/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-linux64/chrome-headless-shell"),
	}
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	return ""
}
