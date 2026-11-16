package e2e

import (
	"strings"
	"testing"
)

// TestSiteManagerLoadsSeeded verifies the site manager lists the
// seeded site.
func TestSiteManagerLoadsSeeded(t *testing.T) {
	body := waitSelector(t, newTab(t), adminURL("/sites/site-manager"), "table")

	if !strings.Contains(body, "SITE_01") {
		t.Fatalf("expected seeded site %q in site manager", "SITE_01")
	}
}
