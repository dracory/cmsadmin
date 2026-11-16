package e2e

import (
	"strings"
	"testing"
)

// TestBlockManagerLoadsSeeded verifies the block manager lists the
// seeded block.
func TestBlockManagerLoadsSeeded(t *testing.T) {
	body := waitSelector(t, newTab(t), adminURL("/blocks/block-manager"), "table tbody tr")

	if !strings.Contains(body, seedBlock) {
		t.Fatalf("expected seeded block %q in block manager", seedBlock)
	}
}
