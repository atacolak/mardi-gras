package components

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
)

func TestNewFooterParadeBindings(t *testing.T) {
	f := NewFooter(80, false, false, false)
	if len(f.Bindings) != len(ParadeBindings) {
		t.Fatalf("expected %d bindings, got %d", len(ParadeBindings), len(f.Bindings))
	}
	for i, b := range f.Bindings {
		if b.Key != ParadeBindings[i].Key || b.Desc != ParadeBindings[i].Desc {
			t.Fatalf("binding %d: got {%s,%s}, want {%s,%s}", i, b.Key, b.Desc, ParadeBindings[i].Key, ParadeBindings[i].Desc)
		}
	}
}

func TestNewFooterDetailBindings(t *testing.T) {
	f := NewFooter(80, true, false, false)
	if len(f.Bindings) != len(DetailBindings) {
		t.Fatalf("expected %d bindings, got %d", len(DetailBindings), len(f.Bindings))
	}
	for i, b := range f.Bindings {
		if b.Key != DetailBindings[i].Key || b.Desc != DetailBindings[i].Desc {
			t.Fatalf("binding %d: got {%s,%s}, want {%s,%s}", i, b.Key, b.Desc, DetailBindings[i].Key, DetailBindings[i].Desc)
		}
	}
}

func TestNewFooterGasTownAddsBindings(t *testing.T) {
	f := NewFooter(80, false, true, false)

	// Should have ParadeBindings + 3 Gas Town bindings (gas town, problems, nudge)
	expected := len(ParadeBindings) + 3
	if len(f.Bindings) != expected {
		t.Fatalf("expected %d bindings with Gas Town, got %d", expected, len(f.Bindings))
	}

	foundNudge := false
	foundSling := false
	quitIdx := -1
	for i, b := range f.Bindings {
		switch b.Key {
		case "s":
			if b.Desc == "sling" {
				foundSling = true
			}
		case "n":
			foundNudge = true
			if quitIdx >= 0 {
				t.Fatal("nudge binding appears after quit")
			}
		case "q":
			quitIdx = i
		}
	}
	if foundSling {
		t.Fatal("s is bead sort, not sling")
	}
	if !foundNudge {
		t.Fatal("missing nudge binding")
	}
}

// The actors hint is progressive-hide: it is in the footer only when the
// actor CLI was found at startup, the way the Gas Town hints track `gt`.
func TestNewFooterActorsAddsBindings(t *testing.T) {
	without := NewFooter(80, false, false, false)
	for _, b := range without.Bindings {
		if b.Key == "o" {
			t.Fatal("actors hint shown without the actor CLI on PATH")
		}
	}

	with := NewFooter(80, false, false, true)
	expected := len(ParadeBindings) + 1
	if len(with.Bindings) != expected {
		t.Fatalf("expected %d bindings with actors, got %d", expected, len(with.Bindings))
	}
	found, navIdx, quitIdx := false, -1, -1
	for i, b := range with.Bindings {
		switch b.Key {
		case "o":
			found = true
			if b.Desc != "actors" {
				t.Errorf("actors binding desc = %q, want %q", b.Desc, "actors")
			}
		case "j/k":
			navIdx = i
		case "q":
			quitIdx = i
		}
	}
	if !found {
		t.Fatal("missing actors binding")
	}
	oIdx := -1
	for i, b := range with.Bindings {
		if b.Key == "o" {
			oIdx = i
			break
		}
	}
	if navIdx >= 0 && oIdx > navIdx {
		t.Errorf("actors hint at %d is after j/k at %d; it must sit with the early operator hints", oIdx, navIdx)
	}
	if quitIdx >= 0 && oIdx > quitIdx {
		t.Error("actors hint appears after quit")
	}
}

// fitBindings drops whole hint chips from the tail once the row is too narrow,
// so a binding ordered last is invisible at 80 columns — the common terminal.
// The actors hint is the only entry point to the pane, so it has to be ordered
// among the hints that survive there (VerifyT3 finding 4).
func TestFooterActorsHintSurvivesAt80Columns(t *testing.T) {
	f := NewFooter(80, false, true, true)
	f.SourceMode = data.SourceCLI
	f.SourceLabel = "br list"
	f.LastRefresh = time.Now()

	output := f.View()
	if !strings.Contains(output, "actors") {
		t.Fatalf("actors hint dropped from an 80-column footer: %q", output)
	}
}

func TestInsertBefore(t *testing.T) {
	bindings := []FooterBinding{
		{Key: "a", Desc: "alpha"},
		{Key: "b", Desc: "beta"},
		{Key: "c", Desc: "gamma"},
	}
	extra := FooterBinding{Key: "x", Desc: "extra"}

	result := insertBefore(bindings, "b", extra)
	if len(result) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(result))
	}
	if result[1].Key != "x" {
		t.Fatalf("expected extra at index 1, got %s", result[1].Key)
	}
	if result[2].Key != "b" {
		t.Fatalf("expected b at index 2, got %s", result[2].Key)
	}
}

func TestInsertBeforeMissingKey(t *testing.T) {
	bindings := []FooterBinding{
		{Key: "a", Desc: "alpha"},
		{Key: "b", Desc: "beta"},
	}
	extra := FooterBinding{Key: "x", Desc: "extra"}

	result := insertBefore(bindings, "z", extra)
	if len(result) != 3 {
		t.Fatalf("expected 3 bindings, got %d", len(result))
	}
	if result[2].Key != "x" {
		t.Fatalf("expected extra appended at end, got %s at index 2", result[2].Key)
	}
}

func TestBulkFooterContainsCount(t *testing.T) {
	output := BulkFooter(80, 5, false)
	if !strings.Contains(output, "5") {
		t.Fatal("BulkFooter output should contain the selection count")
	}
}

func TestBulkFooterGasTownBindings(t *testing.T) {
	output := BulkFooter(80, 3, true)
	if !strings.Contains(output, "3") {
		t.Fatal("BulkFooter output should contain the selection count")
	}
	if !strings.Contains(output, "sling") {
		t.Fatal("BulkFooter with Gas Town should contain 'sling'")
	}
}

func TestBulkFooterNoGasTownNoSling(t *testing.T) {
	output := BulkFooter(80, 2, false)
	if strings.Contains(output, "sling") {
		t.Fatal("BulkFooter without Gas Town should not contain 'sling'")
	}
}

func TestFooterHidesIdleSourceChip(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		SourceLabel: "br list",
		LastRefresh: time.Now(),
		BeadsContext: &data.BeadsContext{
			Database:  "mardi_gras",
			Backend:   "dolt",
			BdVersion: "0.60.0",
		},
	}
	output := ansi.Strip(f.View())
	for _, needle := range []string{"br list", "bd list", "(cli)", "0s ago", "mardi_gras", "v0.60.0"} {
		if strings.Contains(output, needle) {
			t.Fatalf("idle source chip must stay hidden, found %q in: %s", needle, output)
		}
	}
}

func TestFooterHidesJSONLSourceChip(t *testing.T) {
	f := Footer{
		Width:        120,
		Bindings:     ParadeBindings,
		SourceMode:   data.SourceJSONL,
		SourcePath:   "/tmp/.beads/issues.jsonl",
		SourceLabel:  "br list",
		LastRefresh:  time.Now(),
		PathExplicit: false,
	}
	output := ansi.Strip(f.View())
	if strings.Contains(output, "issues.jsonl") || strings.Contains(output, "(legacy)") || strings.Contains(output, "br list") {
		t.Fatalf("JSONL idle source chip must stay hidden, got: %s", output)
	}
}

func TestFooterShowsDegradedSource(t *testing.T) {
	health := data.SourceHealth{State: data.HealthDegraded, LastSuccess: time.Now()}
	f := Footer{
		Width:        120,
		Bindings:     ParadeBindings,
		SourceMode:   data.SourceCLI,
		SourceLabel:  "br list",
		LastRefresh:  time.Now(),
		SourceHealth: &health,
	}
	output := ansi.Strip(f.View())
	if !strings.Contains(output, "degraded") {
		t.Fatalf("degraded source must still paint, got: %s", output)
	}
}

// The scope chip is the only affordance that explains an empty parade. The
// scope pass runs after the fuzzy/exclude/focus passes, so narrowing the scope
// root out of the loaded set leaves data.ScopeToSubtree returning nil, the
// parade goes blank, and every header count reads zero — with nothing on
// screen saying why. The chip has to be lit whenever a scope is active.
func TestFooterRendersEpicScopeChip(t *testing.T) {
	f := NewFooter(120, false, false, false)
	f.ScopeRootID = "mard-r43"
	if !strings.Contains(ansi.Strip(f.View()), "SCOPE mard-r43") {
		t.Fatal("missing scope chip")
	}
}

func TestFooterOmitsEmptyEpicScopeChip(t *testing.T) {
	f := NewFooter(120, false, false, false)
	if strings.Contains(ansi.Strip(f.View()), "SCOPE") {
		t.Fatal("footer shows a scope chip with no scope active")
	}
}

// An empty parade is the *common* shape of an active scope, not an edge case:
// scope an epic whose tree a narrowing pass already dropped and every row is
// gone. The chip must come from the scope field alone — never gated on there
// being rows, groups, or bindings left to show. This footer carries the source
// bar the app always sets and nothing else. Note the limit of the claim: it is
// ungated *within the footer*, and the footer is only one of several bottom
// bars — app.go swaps in the bulk bar, an input bar, or the filter bar, and no
// chip is drawn while one of those is up.
func TestFooterShowsScopeChipWithEmptyParade(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    nil,
		SourceMode:  data.SourceCLI,
		SourcePath:  "/home/sf/workspace/mardi-gras/.beads/issues.jsonl",
		LastRefresh: time.Now(),
		ScopeRootID: "mard-r43",
	}
	if !strings.Contains(ansi.Strip(f.View()), "SCOPE mard-r43") {
		t.Fatal("scope chip must render even when the parade has no rows")
	}
}

func TestFooterSetSortLabels(t *testing.T) {
	f := NewFooter(160, true, false, false)
	f.SetSortLabels("chronological", "priority")
	var epic, bead string
	for _, b := range f.Bindings {
		switch b.Key {
		case "S":
			epic = b.Desc
		case "s":
			bead = b.Desc
		}
	}
	if epic != "epics chronological" {
		t.Fatalf("S desc = %q", epic)
	}
	if bead != "beads priority" {
		t.Fatalf("s desc = %q", bead)
	}
	out := f.View()
	if !strings.Contains(out, "epics") || !strings.Contains(out, "beads") {
		t.Fatalf("footer missing sort chips: %s", out)
	}

	narrow := NewFooter(80, false, true, true)
	narrow.SetSortLabels("chronological", "priority")
	for _, b := range narrow.Bindings {
		if b.Key == "S" && b.Desc != "epics" {
			t.Fatalf("narrow S desc = %q, want short epics", b.Desc)
		}
		if b.Key == "s" && b.Desc != "beads" {
			t.Fatalf("narrow s desc = %q, want short beads", b.Desc)
		}
	}
}
