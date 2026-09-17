package tmux

import (
	"regexp"
	"strings"
	"testing"

	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

var tmuxMarkup = regexp.MustCompile(`#\[[^\]]*\]`)

// stripMarkup drops tmux #[...] directives so the rendered counts can be
// compared as one ordered sequence.
func stripMarkup(s string) string {
	return tmuxMarkup.ReplaceAllString(s, "")
}

// sampleGroups loads the sample fixture and buckets it by derived state.
func sampleGroups(t *testing.T) map[data.SemanticState][]data.Issue {
	t.Helper()
	issues, _, err := data.LoadIssues("../../testdata/sample.jsonl")
	if err != nil {
		t.Fatalf("LoadIssues: %v", err)
	}
	groups, unmapped := data.GroupBySemanticState(issues, data.DefaultBlockingTypes)
	if len(unmapped) != 0 {
		t.Fatalf("sample fixture has unmapped issues: %v", unmapped)
	}
	return groups
}

func TestStatusLineFormat(t *testing.T) {
	got := StatusLine(sampleGroups(t))

	if !strings.Contains(got, "#[fg=") {
		t.Errorf("expected tmux fg markup, got: %s", got)
	}

	// Fleur plus six states, each with its own color segment.
	if n := strings.Count(got, "#[fg="); n != len(data.StateOrder())+1 {
		t.Errorf("expected %d fg segments, got %d: %s", len(data.StateOrder())+1, n, got)
	}
	for i, colour := range []string{"colour220", "colour42", "colour196", "colour240", "colour208", "colour244"} {
		if !strings.Contains(got, colour) {
			t.Errorf("missing color %s for %s: %s", colour, data.StateOrder()[i].Label(), got)
		}
	}

	// The six counts, in StateOrder. Four buckets cannot produce this line.
	if plain := stripMarkup(got); !strings.Contains(plain, "12○ 3● 3⊘ 0⏸ 0◐ 3✓") {
		t.Errorf("expected six ordered counts, got: %q", plain)
	}
	if !strings.Contains(got, ui.FleurDeLis) {
		t.Errorf("missing fleur in: %s", got)
	}
}

// TestStatusLineMardNobAwaitingReview pins the frozen hard-example fixture to
// the tmux surface without spawning a process. The counts come from loading the
// fixture and deriving states, so an expectation edit cannot satisfy it: this
// fails on the same derivation or render regression the built-binary test
// catches, even when that test is skipped.
//
// The fixture's whole point is the ordered pair 0◐ with 7✓: the settled epic
// remains operator review until stored review status is adopted on the board.
func TestStatusLineMardNobAwaitingReview(t *testing.T) {
	issues, _, err := data.LoadIssues("../../testdata/mard-nob-awaiting-review.jsonl")
	if err != nil {
		t.Fatalf("LoadIssues: %v", err)
	}

	groups, unmapped := data.GroupBySemanticState(issues, data.DefaultBlockingTypes)
	if len(unmapped) != 0 {
		t.Fatalf("hard-example fixture has unmapped issues: %v", unmapped)
	}
	if work := groups[data.StateWorking]; len(work) != 0 {
		t.Fatalf("hard-example epic rendered Working: %+v", work)
	}

	// Six ordered pairs, nothing else on the line.
	want := ui.FleurDeLis + " " + strings.Join([]string{"0○", "0●", "0⊘", "0⏸", "1◐", "7✓"}, " ")
	if plain := stripMarkup(StatusLine(groups)); plain != want {
		t.Errorf("status line = %q, want %q", plain, want)
	}
}

func TestStatusLineEmptyGroups(t *testing.T) {
	groups := map[data.SemanticState][]data.Issue{}
	for _, state := range data.StateOrder() {
		groups[state] = []data.Issue{}
	}

	got := StatusLine(groups)

	if plain := stripMarkup(got); !strings.Contains(plain, "0○ 0● 0⊘ 0⏸ 0◐ 0✓") {
	}
}
