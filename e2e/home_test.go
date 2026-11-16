package e2e

import (
	"strings"
	"testing"
)

// TestHomeDashboard verifies the dashboard renders the CMS tiles.
func TestHomeDashboard(t *testing.T) {
	body := waitSelector(t, newTab(t), adminURL("/"), "#page-manager")

	for _, want := range []string{
		"Content Management Dashboard",
		"Total Sites",
		"Total Pages",
		"Total Templates",
		"Total Blocks",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected dashboard to contain %q", want)
		}
	}
}
