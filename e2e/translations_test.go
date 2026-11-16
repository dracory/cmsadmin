package e2e

import (
	"strings"
	"testing"
)

// TestTranslationManagerLoadsSeeded verifies the translation manager
// lists the seeded translation.
func TestTranslationManagerLoadsSeeded(t *testing.T) {
	body := waitSelector(t, newTab(t), adminURL("/translations/translation-manager"), "table tbody tr")

	if !strings.Contains(body, "Translation_01") {
		t.Fatalf("expected seeded translation %q in translation manager", "Translation_01")
	}
}
