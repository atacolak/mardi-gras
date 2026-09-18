// Package components provides reusable UI elements including the header,
// footer, help overlay, command palette, toast notifications, issue
// create/edit forms, and recovery dialog.
package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// Header renders the bead necklace at the top of the screen.
type Header struct {
	Width            int
	Groups           map[data.SemanticState][]data.Issue
	AgentCount       int
	TownStatus       *gastown.TownStatus
	GasTownAvailable bool
	ProblemCount     int
	BeadOffset       int    // 0 = rest necklace; 1..BeadRingFrames-1 = ring spin
	CurrentIssueID   string // active issue from bd show --current
}

// View renders the header: just the bead necklace. The old title / tally /
// progress line is gone — the parade already shows the work.
func (h Header) View() string {
	return h.renderBeadString()
}

// renderBeadString creates the decorative bead string separator with shimmer animation.
func (h Header) renderBeadString() string {
	beads := []string{ui.BeadRound, ui.BeadDiamond}

	var parts []string
	visibleWidth := 0
	ci := 0
	for visibleWidth < h.Width-2 {
		bead := beads[ci%2]
		parts = append(parts, bead)
		visibleWidth++
		if visibleWidth < h.Width-2 {
			parts = append(parts, ui.BeadDash)
			visibleWidth++
		}
		ci++
	}

	rawString := strings.Join(parts, "")

	// Rest is the static assortment. A non-zero offset is one ring spin:
	// colours shift left and wrap, then land back on this same gradient.
	var gradientString string
	if h.BeadOffset > 0 {
		phase := float64(h.BeadOffset) / float64(ui.BeadRingFrames)
		gradientString = ui.ApplyBeadRing(rawString, phase)
	} else {
		gradientString = ui.ApplyMardiGrasGradient(rawString)
	}

	if h.Width > visibleWidth {
		gradientString += strings.Repeat(" ", h.Width-visibleWidth)
	}
	return gradientString
}

func (h Header) renderProgressBar(total, done, length int) string {
	if total == 0 {
		return ""
	}
	filledLen := int((float64(done) / float64(total)) * float64(length))
	emptyLen := length - filledLen

	filled := strings.Repeat("█", filledLen)
	empty := strings.Repeat("█", emptyLen) // Or "━"

	percent := int((float64(done) / float64(total)) * 100)

	styledFilled := ui.ApplyPartialMardiGrasGradient(filled, length)
	styledEmpty := lipgloss.NewStyle().Foreground(ui.DimPurple).Render(empty)

	textRight := ui.HeaderCounts.Render(fmt.Sprintf(" %s%d%%", ui.SymPassed, percent))

	return styledFilled + styledEmpty + textRight
}
