package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/ui"
	"github.com/matt-wright86/mardi-gras/internal/views"
)

const maxPreviews = 2

type previewLayer struct {
	issue  *data.Issue
	detail views.Detail
}

type previewHitKind int

const (
	previewHitNone previewHitKind = iota
	previewHitContent
	previewHitGap
)

type previewHit struct {
	kind     previewHitKind
	layer    int
	detail   *views.Detail
	bodyRow  int
	contentX int
}

func isBareCtrl(msg tea.KeyPressMsg) bool {
	if msg.Text != "" {
		return false
	}
	switch msg.String() {
	case "leftctrl", "rightctrl":
		return true
	}
	return msg.Code == tea.KeyLeftCtrl || msg.Code == tea.KeyRightCtrl
}

func previewPad(w, h int) int {
	p := 7
	if max := w / 8; max < p {
		p = max
	}
	if max := h / 8; max < p {
		p = max
	}
	if p < 2 {
		return 2
	}
	return p
}

func (m Model) handlePreviewTrigger() (tea.Model, tea.Cmd) {
	issue := m.issueAtPointer()
	if issue == nil {
		return m, nil
	}
	m.pushPreview(issue)
	return m, nil
}

func (m *Model) pushPreview(issue *data.Issue) {
	if issue == nil {
		return
	}
	if m.detail.Issue != nil && m.detail.Issue.ID == issue.ID && len(m.previews) == 0 {
		return
	}
	if n := len(m.previews); n > 0 && m.previews[n-1].issue != nil && m.previews[n-1].issue.ID == issue.ID {
		return
	}
	if len(m.previews) >= maxPreviews {
		return
	}
	w, h := m.previewInnerSize(len(m.previews))
	if w < 20 || h < 6 {
		return
	}
	d := m.newPreviewDetail(issue, w, h)
	m.previews = append(m.previews, previewLayer{issue: issue, detail: d})
}

func (m *Model) popPreview() {
	if len(m.previews) == 0 {
		return
	}
	m.previews = m.previews[:len(m.previews)-1]
}

func (m *Model) popPreviewTo(layer int) {
	if layer < 0 {
		m.previews = nil
		return
	}
	if layer >= len(m.previews) {
		return
	}
	m.previews = m.previews[:layer]
}

func (m *Model) clearPreviews() {
	m.previews = nil
}

func (m Model) newPreviewDetail(issue *data.Issue, w, h int) views.Detail {
	d := views.NewDetail(w, h, m.issues)
	d.AllIssues = m.issues
	d.IssueMap = m.detail.IssueMap
	d.BlockingTypes = m.detail.BlockingTypes
	d.MetadataSchema = m.detail.MetadataSchema
	d.SetSize(w, h)
	d.SetIssue(issue)
	return d
}

func (m Model) previewInnerSize(stackIndex int) (w, h int) {
	pad := previewPad(m.detail.Width, m.detail.Height)
	inset := pad * (stackIndex + 1)
	w = m.detail.Width - 2*inset - 2
	h = m.detail.Height - 2*inset - 2
	return w, h
}

func (m *Model) resizePreviews() {
	for i := range m.previews {
		w, h := m.previewInnerSize(i)
		if w < 20 || h < 6 {
			continue
		}
		issue := m.previews[i].issue
		m.previews[i].detail = m.newPreviewDetail(issue, w, h)
	}
}

func (m Model) issueAtPointer() *data.Issue {
	_, _, paradeW := m.bodyBounds()
	dx := m.lastMouseX - paradeW
	dy := m.lastMouseY - headerHeight
	if hit := m.hitPreview(dx, dy); hit.kind == previewHitContent && hit.detail != nil {
		return hit.detail.ReferenceAtXY(hit.bodyRow, hit.contentX)
	}
	if dx < 0 {
		return nil
	}
	bodyRow := dy
	contentX := dx - m.detail.ContentInsetX()
	return m.detail.ReferenceAtXY(bodyRow, contentX)
}

func (m Model) hitPreview(dx, dy int) previewHit {
	if len(m.previews) == 0 {
		return previewHit{}
	}
	w, h := m.detail.Width, m.detail.Height
	pad := previewPad(w, h)
	for i := len(m.previews) - 1; i >= 0; i-- {
		inset := pad * (i + 1)
		boxW := w - 2*inset
		boxH := h - 2*inset
		inBox := dx >= inset && dx < inset+boxW && dy >= inset && dy < inset+boxH
		if inBox {
			return previewHit{
				kind:     previewHitContent,
				layer:    i,
				detail:   &m.previews[i].detail,
				bodyRow:  dy - inset - 1,
				contentX: dx - inset - 1,
			}
		}
		prevInset := pad * i
		inOuter := dx >= prevInset && dx < w-prevInset && dy >= prevInset && dy < h-prevInset
		if inOuter {
			return previewHit{kind: previewHitGap, layer: i}
		}
	}
	return previewHit{}
}

func (m Model) viewDetailWithPreviews() string {
	base := m.detail.View()
	if len(m.previews) == 0 {
		return base
	}
	w, h := m.detail.Width, m.detail.Height
	pad := previewPad(w, h)
	out := base
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ui.Gold)
	for i, layer := range m.previews {
		inset := pad * (i + 1)
		innerW, innerH := m.previewInnerSize(i)
		if innerW < 20 || innerH < 6 {
			continue
		}
		layer.detail.SetSize(innerW, innerH)
		framed := boxStyle.Width(innerW).Height(innerH).Render(layer.detail.ContentView())
		out = stampRect(out, inset, inset, framed)
	}
	return out
}

func stampRect(base string, col, row int, block string) string {
	baseLines := strings.Split(base, "\n")
	blockLines := strings.Split(block, "\n")
	for i, bl := range blockLines {
		r := row + i
		if r < 0 {
			continue
		}
		for len(baseLines) <= r {
			baseLines = append(baseLines, "")
		}
		baseLines[r] = spliceLine(baseLines[r], col, bl)
	}
	return strings.Join(baseLines, "\n")
}

func spliceLine(base string, col int, insert string) string {
	if col < 0 {
		col = 0
	}
	insertW := lipgloss.Width(insert)
	left := ansi.Cut(base, 0, col)
	if pad := col - lipgloss.Width(left); pad > 0 {
		left += strings.Repeat(" ", pad)
	}
	right := ansi.TruncateLeft(base, col+insertW, "")
	return left + insert + right
}

func (m *Model) openIssueInDetail(issue *data.Issue) {
	if issue == nil {
		return
	}
	m.clearPreviews()
	m.activPane = PaneDetail
	m.detail.Focused = true
	m.detail.SetIssue(issue)
	if m.restoreParadeSelection(issue.ID) {
		m.syncSelection()
	}
}
