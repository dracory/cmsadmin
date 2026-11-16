package e2e

import (
	"strings"
	"testing"
)

// TestPageManagerLoadsSeeded verifies the Vue page manager loads and
// lists the seeded pages.
func TestPageManagerLoadsSeeded(t *testing.T) {
	body := waitTable(t, newTab(t), adminURL("/pages/page-manager"))

	if !strings.Contains(body, seedPage) {
		t.Fatalf("expected seeded page %q in page manager", seedPage)
	}
}
