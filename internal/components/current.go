package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/matt-wright86/mardi-gras/internal/digest"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// Current is the compact project-summary strip under the necklace.
// It only renders a published shared digest. It does not summarize the board.
type Current struct {
	Width  int
	Digest *digest.Digest
	Err    error
	Now    time.Time
}

// Height is the number of terminal rows the strip occupies. Zero when there
// is no published digest — the cockpit layout does not change.
func (c Current) Height() int {
	if c.Digest == nil {
		return 0
	}
	n := 2 // label+freshness, then facts
	if c.narrative() != "" {
		n++
	}
	return n
}

// View renders the strip, or "" when there is nothing to consume.
func (c Current) View() string {
	if c.Width <= 0 || c.Digest == nil {
		return ""
	}
	now := c.Now
	if now.IsZero() {
		now = time.Now()
	}
	fresh := c.Digest.Freshness(now)
	freshStyle := ui.CurrentFresh
	if c.Digest.Stale(now) {
		freshStyle = ui.CurrentStale
	}
	head := ui.CurrentLabel.Render("CURRENT") + "  " + freshStyle.Render(fresh)
	if c.Err != nil {
		head += "  " + ui.CurrentStale.Render("read error")
	}
	lines := []string{pad(head, c.Width)}
	lines = append(lines, pad(ui.CurrentFacts.Render(c.factsLine()), c.Width))
	if n := c.narrative(); n != "" {
		// Prefix marks compression so it cannot be mistaken for work state.
		line := ui.CurrentNarrative.Render("note  " + n)
		lines = append(lines, pad(line, c.Width))
	}
	return strings.Join(lines, "\n")
}

func (c Current) factsLine() string {
	d := c.Digest
	var parts []string
	if d.Sprint != nil && d.Sprint.Epic != "" {
		s := d.Sprint.Epic
		if d.Sprint.Phase != "" {
			s += " " + d.Sprint.Phase
		}
		parts = append(parts, s)
	}
	if d.Now != nil {
		item := fmt.Sprintf("now %d", d.Now.Count)
		if len(d.Now.Frontier) > 0 && d.Now.Frontier[0].ID != "" {
			item += " " + d.Now.Frontier[0].ID
		}
		parts = append(parts, item)
	}
	if n := len(d.Working); n > 0 {
		w := fmt.Sprintf("working %d", n)
		if d.Working[0].Actor != "" {
			w += " " + d.Working[0].Actor
		}
		parts = append(parts, w)
	}
	if len(d.Waiting) > 0 {
		w := d.Waiting[0]
		label := "waiting"
		if w.Reason != "" {
			label += " " + w.Reason
		}
		if w.Count > 0 {
			label += fmt.Sprintf(" %d", w.Count)
		}
		parts = append(parts, label)
	}
	if d.AwaitingReview {
		parts = append(parts, "awaiting review")
	}
	if d.ForUser > 0 {
		parts = append(parts, fmt.Sprintf("for user %d", d.ForUser))
	}
	if len(d.Recent) > 0 && d.Recent[0].ID != "" {
		parts = append(parts, "landed "+d.Recent[0].ID)
	}
	if len(parts) == 0 {
		return "shared digest"
	}
	return strings.Join(parts, "  ·  ")
}

func (c Current) narrative() string {
	if c.Digest.Narrative == nil {
		return ""
	}
	return strings.Join(strings.Fields(c.Digest.Narrative.Text), " ")
}

func pad(s string, width int) string {
	return lipgloss.NewStyle().Width(width).MaxHeight(1).Render(s)
}
