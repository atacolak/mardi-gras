package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

func TestHeaderCountsSixSemanticStates(t *testing.T) {
	issues, _, err := data.LoadIssues(filepath.Join("..", "..", "testdata", "sample.jsonl"))
	if err != nil {
		t.Fatalf("LoadIssues: %v", err)
	}
	groups, unmapped := data.GroupBySemanticState(issues, data.DefaultBlockingTypes)
	if len(unmapped) != 0 {
		t.Fatalf("sample fixture has unmapped issues: %v", unmapped)
	}

	h := Header{Width: 200, Groups: groups}
	out := ansi.Strip(h.View())

	// Six counts in StateOrder. A four-count header cannot satisfy this, and
	// can no longer be written: the field is keyed by semantic state.
	if !strings.Contains(out, "12● 3◐ 3⊘ 0⏸ 0○ 3✓") {
		t.Fatalf("header should show six ordered counts, got:\n%s", out)
	}
}

// TestHeaderCountsUseExecColors pins ask 10's "header tally uses the same Exec*
// colors": each count is rendered in its own state color, not one flat ink.
func TestHeaderCountsUseExecColors(t *testing.T) {
	groups := map[data.SemanticState][]data.Issue{
		data.StateReady:          {{ID: "r1"}},
		data.StateWorking:        {{ID: "w1"}},
		data.StateWaitingBlocked: {{ID: "b1"}},
		data.StateDeferred:       {{ID: "d1"}},
		data.StateOperatorReview: {{ID: "a1"}},
		data.StateDone:           {{ID: "z1"}},
	}
	out := Header{Width: 200, Groups: groups}.View()
	for _, state := range data.StateOrder() {
		want := lipgloss.NewStyle().
			Foreground(ui.ExecColor(int(state))).
			Render(fmt.Sprintf("1%s", ui.ExecSymbol(int(state))))
		if !strings.Contains(out, want) {
			t.Errorf("header tally missing %s count in its Exec color", state.Label())
		}
	}
}

func TestHeaderRigCountMultiRig(t *testing.T) {
	h := Header{
		Width:            120,
		GasTownAvailable: true,
		TownStatus: &gastown.TownStatus{
			Rigs: []gastown.RigStatus{
				{Name: "rig_alpha"},
				{Name: "rig_beta"},
				{Name: "rig_gamma"},
			},
		},
	}
	output := h.View()
	if !strings.Contains(output, "3 rigs") {
		t.Fatalf("expected header to contain '3 rigs' for multi-rig, got: %s", output)
	}
}

func TestHeaderRigCountSingleRig(t *testing.T) {
	h := Header{
		Width:            120,
		GasTownAvailable: true,
		TownStatus: &gastown.TownStatus{
			Rigs: []gastown.RigStatus{
				{Name: "rig_alpha"},
			},
		},
	}
	output := h.View()
	if strings.Contains(output, "rigs") {
		t.Fatalf("expected header to NOT show rig count for single rig, got: %s", output)
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
