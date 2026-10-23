package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

const (
	commentsDockMinExpanded  = 6
	commentsDockMinBody      = 4
	defaultCommentsDockRatio = 0.38
)

func (m Model) commentsDockHeight() int {
	h := m.detail.Height
	if h <= 0 || m.detail.Issue == nil {
		return 0
	}
	if !m.commentsDockOpen {
		return 1
	}
	if m.commentsDockOverride > 0 {
		return clampDockHeight(m.commentsDockOverride, h)
	}
	ratio := m.commentsDockRatio
	if ratio <= 0 {
		ratio = defaultCommentsDockRatio
	}
	return clampDockHeight(int(float64(h)*ratio), h)
}

func clampDockHeight(want, detailH int) int {
	maxH := detailH - commentsDockMinBody
	if maxH < 1 {
		maxH = 1
	}
	if want < commentsDockMinExpanded {
		want = commentsDockMinExpanded
	}
	if want > maxH {
		want = maxH
	}
	if want < 1 {
		return 1
	}
	return want
}

func (m Model) stampCommentsDock(base string) string {
	if m.detail.Issue == nil || m.layoutPreset == LayoutWide {
		return base
	}
	h := m.commentsDockHeight()
	if h <= 0 {
		return base
	}
	w := m.detail.Width
	if w < 8 {
		return base
	}
	block := m.renderCommentsDock(w, h)
	top := m.detail.Height - h
	if top < 0 {
		top = 0
	}
	return stampRect(base, 0, top, block)
}

func (m Model) renderCommentsDock(width, height int) string {
	issue := m.detail.Issue
	n := 0
	if m.detail.CommentsIssueID == issue.ID {
		n = len(m.detail.Comments)
	}
	title := fmt.Sprintf(" %s  COMMENTS", ui.SymCommentNerd)
	if n > 0 {
		title = fmt.Sprintf(" %s  COMMENTS  %d comments", ui.SymCommentNerd, n)
	}
	fold := ui.SymDockFold
	if !m.commentsDockOpen {
		fold = ui.SymDockUnfold
	}
	foldLabel := " " + fold + " fold "
	titleW := lipgloss.Width(title)
	foldW := lipgloss.Width(foldLabel)
	fill := width - titleW - foldW
	if fill < 1 {
		fill = 1
	}
	headerPlain := title + strings.Repeat(" ", fill) + foldLabel
	if lipgloss.Width(headerPlain) > width {
		headerPlain = ansi.Truncate(headerPlain, width, "")
	}
	header := lipgloss.NewStyle().
		Foreground(ui.Gold).
		Background(lipgloss.Color("#1a1a22")).
		Width(width).
		Render(headerPlain)

	if !m.commentsDockOpen || height <= 1 {
		return header
	}

	var body strings.Builder
	inner := height - 1
	if m.commentsErr != nil {
		body.WriteString(lipgloss.NewStyle().Foreground(ui.StatusStalled).Width(width).Render("  comments failed: " + m.commentsErr.Error()))
		inner--
	} else if n == 0 {
		empty := lipgloss.NewStyle().Foreground(ui.Muted).Align(lipgloss.Center).Width(width).Render(ui.SymCommentNerd + "  No comments on this bead yet.")
		hint := lipgloss.NewStyle().Foreground(ui.Dim).Align(lipgloss.Center).Width(width).Render("Add notes, findings, questions, or updates about this task.")
		body.WriteString(empty + "\n" + hint)
		inner -= 2
	} else {
		for _, c := range m.detail.Comments {
			seen := m.commentIsSeen(issue.ID, c.ID)
			mark := " "
			if !seen {
				mark = ui.SymUnseen
			}
			author := c.Author
			if author == "" {
				author = "?"
			}
			line := fmt.Sprintf("%s %s  %s", mark, author, strings.ReplaceAll(c.Body, "\n", " "))
			if lipgloss.Width(line) > width-1 {
				line = ansi.Truncate(line, width-1, "…")
			}
			style := lipgloss.NewStyle().Foreground(ui.Light).Width(width)
			if !seen {
				style = style.Foreground(ui.Gold)
			}
			body.WriteString(style.Render(" "+line) + "\n")
			inner--
			if inner <= 1 {
				break
			}
		}
	}
	compose := lipgloss.NewStyle().Foreground(ui.Dim).Width(width).Render("  > add a comment…          press c to comment")
	if inner > 0 {
		if body.Len() > 0 && !strings.HasSuffix(body.String(), "\n") {
			body.WriteByte('\n')
		}
		body.WriteString(compose)
		inner--
	}
	out := header + "\n" + body.String()
	lines := strings.Split(out, "\n")
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, ln := range lines {
		if lipgloss.Width(ln) < width {
			lines[i] = ln + strings.Repeat(" ", width-lipgloss.Width(ln))
		}
	}
	box := lipgloss.NewStyle().
		Foreground(ui.Gold).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ui.Gold).
		Width(width).
		MaxHeight(height)
	// Border would grow the box past the overlay slot. Paint without an extra
	// outer border; the gold header is the bar.
	_ = box
	return strings.Join(lines, "\n")
}

func (m Model) commentIsSeen(issueID, commentID string) bool {
	if issueID == "" || commentID == "" || m.seenCommentIDs == nil {
		return false
	}
	return m.seenCommentIDs[issueID][commentID]
}

func (m *Model) markCommentSeen(issueID, commentID string) {
	if issueID == "" || commentID == "" {
		return
	}
	if m.seenCommentIDs == nil {
		m.seenCommentIDs = map[string]map[string]bool{}
	}
	if m.seenCommentIDs[issueID] == nil {
		m.seenCommentIDs[issueID] = map[string]bool{}
	}
	m.seenCommentIDs[issueID][commentID] = true
}

// commentsDockTopRow is the detail-local row of the dock header.
func (m Model) commentsDockTopRow() int {
	h := m.commentsDockHeight()
	if h <= 0 {
		return -1
	}
	return m.detail.Height - h
}

func (m Model) commentsDockFoldHit(detailX, detailRow int) bool {
	if m.detail.Issue == nil {
		return false
	}
	top := m.commentsDockTopRow()
	if detailRow != top {
		return false
	}
	w := m.detail.Width
	return detailX >= w-8
}

func (m Model) commentsDockHeaderHit(detailRow int) bool {
	return m.detail.Issue != nil && detailRow == m.commentsDockTopRow()
}

func (m *Model) hoverCommentsDock(detailX, detailRow int) {
	if m.detail.Issue == nil || !m.commentsDockOpen {
		return
	}
	top := m.commentsDockTopRow()
	bodyRow := detailRow - top - 1
	if bodyRow < 0 {
		return
	}
	if m.detail.CommentsIssueID != m.detail.Issue.ID {
		return
	}
	_ = detailX
	if bodyRow >= 0 && bodyRow < len(m.detail.Comments) {
		m.markCommentSeen(m.detail.Issue.ID, m.detail.Comments[bodyRow].ID)
	}
}

func commentPreviewLine(c gastown.Comment, width int) string {
	line := strings.ReplaceAll(c.Body, "\n", " ")
	if lipgloss.Width(line) > width {
		return ansi.Truncate(line, width, "…")
	}
	return line
}
