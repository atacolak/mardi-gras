// Package tmux provides a status bar widget that renders parade issue counts
// in tmux-compatible color format.
package tmux

import (
	"fmt"
	"strings"

	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// tmux 256-color equivalents for the semantic execution state palette, keyed by
// state. ui.ExecColor cannot be used here: tmux markup carries no lipgloss.
func execColour(state data.SemanticState) string {
	switch state {
	case data.StateReady:
		return "colour220" // BrightGold
	case data.StateWorking:
		return "colour42" // BrightGreen
	case data.StateWaitingBlocked:
		return "colour196" // Red
	case data.StateDeferred:
		return "colour240" // Dim
	case data.StateOperatorReview:
		return "colour208" // Orange
	case data.StateDone:
		return "colour244" // Muted
	}
	return "colour244"
}

const colourFleur = "colour134" // Purple #7B2D8E

// StatusLine returns a tmux-formatted status string showing one count per
// semantic execution state, in StateOrder.
// Output uses tmux #[fg=colourN] markup — no lipgloss dependency.
func StatusLine(groups map[data.SemanticState][]data.Issue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#[fg=%s]%s", colourFleur, ui.FleurDeLis)
	for _, state := range data.StateOrder() {
		fmt.Fprintf(&b, " #[fg=%s]%d%s",
			execColour(state), len(groups[state]), ui.ExecSymbol(int(state)))
	}
	return b.String()
}
