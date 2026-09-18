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

// Header renders the top title bar with bead string and counts.
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

// View renders the header.
func (h Header) View() string {
	// One count per semantic state, in StateOrder — the same order the parade
	// sections and the tmux widget use, so a four-bucket header cannot be
	// written and the three surfaces cannot drift apart.
	var countsB strings.Builder
	total := 0
	for _, state := range data.StateOrder() {
		count := len(h.Groups[state])
		total += count
		countsB.WriteString(" ")
		countsB.WriteString(lipgloss.NewStyle().
			Foreground(ui.ExecColor(int(state))).
			Render(fmt.Sprintf("%d%s", count, ui.ExecSymbol(int(state)))))
	}

	titleStr := fmt.Sprintf("%s MARDI GRAS %s", ui.FleurDeLis, ui.FleurDeLis)
	title := ui.HeaderStyle.Render(ui.ApplyMardiGrasGradient(titleStr))

	counts := countsB.String()

	agentInfo := ""
	if h.AgentCount > 0 {
		agentStyle := lipgloss.NewStyle().Foreground(ui.StatusAgent).Bold(true)
		agentInfo = agentStyle.Render(fmt.Sprintf(" %s%d", ui.SymAgent, h.AgentCount))
	}

	gasTownInfo := ""
	if h.GasTownAvailable && h.TownStatus != nil {
		working := h.TownStatus.WorkingCount()
		totalAgents := len(h.TownStatus.Agents)
		gtStyle := lipgloss.NewStyle().Foreground(ui.BrightPurple).Italic(true)

		parts := []string{fmt.Sprintf("gt:%d/%d", working, totalAgents)}

		if rigCount := len(h.TownStatus.Rigs); rigCount > 1 {
			parts = append(parts, fmt.Sprintf("%d rigs", rigCount))
		}

		if mail := h.TownStatus.UnreadMail(); mail > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", ui.SymMail, mail))
		}

		activeConvoys := 0
		for _, c := range h.TownStatus.Convoys {
			if c.Status == "open" {
				activeConvoys++
			}
		}
		if activeConvoys > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", ui.SymConvoy, activeConvoys))
		}

		if mq := h.TownStatus.MQStatus(); mq != nil && (mq.Pending > 0 || mq.InFlight > 0) {
			mqLabel := fmt.Sprintf("MQ:%d", mq.Pending+mq.InFlight)
			if mq.Health == "stale" || mq.State == "blocked" {
				mqLabel = lipgloss.NewStyle().Foreground(ui.StatusStalled).Bold(true).Render(mqLabel)
			} else {
				mqLabel = gtStyle.Render(mqLabel)
			}
			parts = append(parts, mqLabel)
		}

		gasTownInfo = gtStyle.Render(" " + strings.Join(parts, " "))
	}

	currentInfo := ""
	if h.CurrentIssueID != "" {
		currentStyle := lipgloss.NewStyle().Foreground(ui.BrightGold).Italic(true)
		currentInfo = currentStyle.Render(fmt.Sprintf(" %s %s", ui.SymWorking, h.CurrentIssueID))
	}

	problemInfo := ""
	if h.ProblemCount > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(ui.StatusStalled).Bold(true)
		problemInfo = warnStyle.Render(fmt.Sprintf(" %s%d", ui.SymWarning, h.ProblemCount))
	}

	bar := h.renderProgressBar(total, len(h.Groups[data.StateDone]), 20)

	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		counts,
		currentInfo,
		agentInfo,
		gasTownInfo,
		problemInfo,
		"  ",
		bar,
	)

	// Pad to full width
	titleLine = lipgloss.NewStyle().Width(h.Width).Render(titleLine)

	beadStr := h.renderBeadString()

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, beadStr)
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
