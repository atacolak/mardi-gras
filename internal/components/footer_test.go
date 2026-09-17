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

	// Should have ParadeBindings + 4 Gas Town bindings (gas town, problems, sling, nudge)
	expected := len(ParadeBindings) + 4
	if len(f.Bindings) != expected {
		t.Fatalf("expected %d bindings with Gas Town, got %d", expected, len(f.Bindings))
	}

	// Find sling and nudge before quit
	foundSling := false
	foundNudge := false
	quitIdx := -1
	for i, b := range f.Bindings {
		switch b.Key {
		case "s":
			foundSling = true
			if quitIdx >= 0 {
				t.Fatal("sling binding appears after quit")
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
	if !foundSling {
		t.Fatal("missing sling binding")
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

func TestFooterViewWithBeadsContext(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		LastRefresh: time.Now(),
		BeadsContext: &data.BeadsContext{
			Database: "mardi_gras",
			Backend:  "dolt",
		},
	}
	output := f.View()
	if !strings.Contains(output, "mardi_gras/dolt") {
		t.Fatalf("footer should contain database/backend, got: %s", output)
	}
}

func TestFooterViewWithBeadsContextVersion(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		LastRefresh: time.Now(),
		BeadsContext: &data.BeadsContext{
			Database:  "mardi_gras",
			Backend:   "dolt",
			BdVersion: "0.60.0",
		},
	}
	output := f.View()
	if !strings.Contains(output, "v0.60.0") {
		t.Fatalf("footer should contain bd version, got: %s", output)
	}
}

func TestFooterViewWithBeadsContextNoBackend(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		LastRefresh: time.Now(),
		BeadsContext: &data.BeadsContext{
			Database: "mardi_gras",
		},
	}
	output := f.View()
	if !strings.Contains(output, "mardi_gras") {
		t.Fatalf("footer should contain database name, got: %s", output)
	}
	if strings.Contains(output, "mardi_gras/") {
		t.Fatalf("footer should not have trailing slash without backend, got: %s", output)
	}
}

func TestFooterViewWithoutBeadsContext(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		LastRefresh: time.Now(),
	}
	output := f.View()
	if strings.Contains(output, "mardi_gras") {
		t.Fatalf("footer should not contain context info when nil, got: %s", output)
	}
}

// Source.Label() already returns "br list" for CLIBr, but the footer used to
// hardcode "bd list" for every SourceCLI source. Prefer the resolved label.
func TestFooterViewUsesSourceLabel(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		SourceLabel: "br list",
		LastRefresh: time.Now(),
	}
	output := f.View()
	if !strings.Contains(output, "br list") {
		t.Fatalf("footer should report SourceLabel, got: %s", output)
	}
	if strings.Contains(output, "bd list") {
		t.Fatalf("footer should not fall back to bd list when SourceLabel is set, got: %s", output)
	}
}

func TestFooterViewCLIFallsBackToBdList(t *testing.T) {
	f := Footer{
		Width:       120,
		Bindings:    ParadeBindings,
		SourceMode:  data.SourceCLI,
		LastRefresh: time.Now(),
	}
	output := f.View()
	if !strings.Contains(output, "bd list") {
		t.Fatalf("empty SourceLabel should keep the bd list fallback, got: %s", output)
	}
}

func TestFooterViewJSONLIgnoresSourceLabel(t *testing.T) {
	f := Footer{
		Width:        120,
		Bindings:     ParadeBindings,
		SourceMode:   data.SourceJSONL,
		SourcePath:   "/tmp/.beads/issues.jsonl",
		SourceLabel:  "br list",
		LastRefresh:  time.Now(),
		PathExplicit: false,
	}
	output := f.View()
	if !strings.Contains(output, "issues.jsonl") {
		t.Fatalf("JSONL footer should keep the file basename, got: %s", output)
	}
	if strings.Contains(output, "br list") {
		t.Fatalf("JSONL footer should ignore SourceLabel, got: %s", output)
	}
	if !strings.Contains(output, "(legacy)") {
		t.Fatalf("JSONL footer should keep the (legacy) mode, got: %s", output)
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
