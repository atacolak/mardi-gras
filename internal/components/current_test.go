package components

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/digest"
)

func TestCurrentHiddenWithoutDigest(t *testing.T) {
	c := Current{Width: 80}
	if c.Height() != 0 || c.View() != "" {
		t.Fatalf("empty current must occupy no chrome: h=%d view=%q", c.Height(), c.View())
	}
}

func TestCurrentShowsFactsFreshnessAndNarrative(t *testing.T) {
	now := time.Date(2026, 9, 19, 5, 5, 0, 0, time.UTC)
	c := Current{
		Width: 80,
		Now:   now,
		Digest: &digest.Digest{
			GeneratedAt: now.Add(-2 * time.Minute),
			Sprint:      &digest.Sprint{Epic: "mard-wte", Phase: "executing"},
			Now:         &digest.Now{Count: 2, Frontier: []digest.Frontier{{ID: "mard-wte.13"}}},
			Working:     []digest.Working{{Actor: "zime"}},
			Waiting:     []digest.Waiting{{Reason: "publish", Count: 1}},
			ForUser:     1,
			Recent:      []digest.Change{{ID: "mard-wte.1"}},
			Narrative:   &digest.Narrative{Text: "Dogfood is still landing."},
		},
	}
	if c.Height() != 3 {
		t.Fatalf("height = %d, want 3", c.Height())
	}
	plain := ansi.Strip(c.View())
	if !strings.Contains(plain, "CURRENT") {
		t.Fatalf("missing CURRENT label:\n%s", plain)
	}
	if !strings.Contains(plain, "2m ago") {
		t.Fatalf("missing freshness:\n%s", plain)
	}
	if !strings.Contains(plain, "mard-wte") || !strings.Contains(plain, "now 2") {
		t.Fatalf("missing deterministic facts:\n%s", plain)
	}
	if !strings.Contains(plain, "note  Dogfood is still landing.") {
		t.Fatalf("narrative must be marked as note, not work state:\n%s", plain)
	}
	// Must not look like a parade row.
	for _, glyph := range []string{"●", "◐", "⊘", "⏸", "○", "✓"} {
		if strings.Contains(plain, glyph) {
			t.Fatalf("current must not use exec glyphs, found %q:\n%s", glyph, plain)
		}
	}
}

func TestCurrentStaleChip(t *testing.T) {
	now := time.Date(2026, 9, 19, 5, 20, 0, 0, time.UTC)
	c := Current{
		Width:  60,
		Now:    now,
		Digest: &digest.Digest{GeneratedAt: now.Add(-20 * time.Minute), Sprint: &digest.Sprint{Epic: "mard-wte"}},
	}
	plain := ansi.Strip(c.View())
	if !strings.Contains(plain, "stale 20m") {
		t.Fatalf("stale freshness:\n%s", plain)
	}
}
