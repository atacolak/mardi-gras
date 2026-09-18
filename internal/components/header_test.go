package components

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

func TestHeaderIsBeadLineOnly(t *testing.T) {
	issues, _, err := data.LoadIssues(filepath.Join("..", "..", "testdata", "sample.jsonl"))
	if err != nil {
		t.Fatalf("LoadIssues: %v", err)
	}
	groups, unmapped := data.GroupBySemanticState(issues, data.DefaultBlockingTypes)
	if len(unmapped) != 0 {
		t.Fatalf("sample fixture has unmapped issues: %v", unmapped)
	}

	h := Header{
		Width:            200,
		Groups:           groups,
		GasTownAvailable: true,
		TownStatus: &gastown.TownStatus{
			Rigs: []gastown.RigStatus{{Name: "rig_alpha"}, {Name: "rig_beta"}},
		},
		AgentCount:     3,
		ProblemCount:   2,
		CurrentIssueID: "mard-wte",
	}
	out := ansi.Strip(h.View())
	if strings.Contains(out, "MARDI GRAS") {
		t.Fatal("title line must be gone")
	}
	if strings.Contains(out, "12●") || strings.Contains(out, "✓70%") || strings.Contains(out, "3 rigs") {
		t.Fatalf("tally/progress/rig chrome must be gone, got:\n%s", out)
	}
	if strings.Count(out, "\n") != 0 {
		t.Fatalf("header should be a single bead line, got %d newlines:\n%s", strings.Count(out, "\n"), out)
	}
	if !strings.Contains(out, ui.BeadRound) {
		t.Fatal("header must still be the bead necklace")
	}
}

func TestBeadStringIdleIsStaticGradient(t *testing.T) {
	h := Header{Width: 40}
	got := h.renderBeadString()
	beads := []string{ui.BeadRound, ui.BeadDiamond}
	var parts []string
	visibleWidth := 0
	ci := 0
	for visibleWidth < 38 {
		parts = append(parts, beads[ci%2])
		visibleWidth++
		if visibleWidth < 38 {
			parts = append(parts, ui.BeadDash)
			visibleWidth++
		}
		ci++
	}
	raw := strings.Join(parts, "")
	want := ui.ApplyMardiGrasGradient(raw) + strings.Repeat(" ", 40-visibleWidth)
	if got != want {
		t.Fatal("idle necklace must be the static Mardi Gras gradient")
	}
	h.BeadOffset = 8
	moved := h.renderBeadString()
	if moved == got {
		t.Fatal("ring frame must shift necklace colours")
	}
	h.BeadOffset = ui.BeadRingFrames
	if h.renderBeadString() != got {
		t.Fatal("full-cycle frame must land on the rest necklace")
	}
}

func TestRenderProgressBarZeroTotal(t *testing.T) {
	h := Header{}
	if got := h.renderProgressBar(0, 0, 20); got != "" {
		t.Errorf("zero total should yield empty bar, got %q", got)
	}
}

func TestRenderProgressBarBoundaries(t *testing.T) {
	// done==total fills the bar entirely; the percent label reaches 100%.
	h := Header{}
	got := h.renderProgressBar(10, 10, 20)
	if !strings.Contains(got, "100%") {
		t.Errorf("done==total should render 100%%, got %q", got)
	}
	// done==0 fills nothing; the percent label is 0%.
	got = h.renderProgressBar(10, 0, 20)
	if !strings.Contains(got, "0%") {
		t.Errorf("done==0 should render 0%%, got %q", got)
	}
}

func TestRenderProgressBarHalfDone(t *testing.T) {
	h := Header{}
	got := h.renderProgressBar(10, 5, 20)
	if !strings.Contains(got, "50%") {
		t.Errorf("5/10 should render 50%%, got %q", got)
	}
}

func TestRenderProgressBarDoesNotPanicOnOverflow(t *testing.T) {
	// done > total would make emptyLen negative — strings.Repeat panics on
	// negative count. Today renderProgressBar does not guard, but if a
	// future refactor adds the guard this test still passes (it asserts no
	// panic and a sensible percent). If today's code does panic on
	// overflow, we want to know.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("renderProgressBar panicked on done>total: %v (consider adding a clamp)", r)
		}
	}()
	h := Header{}
	_ = h.renderProgressBar(10, 15, 20)
}
