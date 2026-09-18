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

const previewVertGrow = 2

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

func previewYInset(xInset int) int {
	y := xInset - previewVertGrow
	if y < 2 {
		return 2
	}
	return y
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
	d.SetPreviewSize(w, h)
	d.SetIssue(issue)
	return d
}

func (m Model) previewGeom(stackIndex int) (x, y, innerW, innerH int) {
	w, h := m.detail.Width, m.detail.Height
	x = previewPad(w, h) * (stackIndex + 1)
	y = previewYInset(x)
	innerW = w - 2*x - 2
	innerH = h - 2*y - 2
	return x, y, innerW, innerH
}

func (m Model) previewInnerSize(stackIndex int) (w, h int) {
	_, _, w, h = m.previewGeom(stackIndex)
	return w, h
}

func (m *Model) scrollTopPreview(n int) bool {
	if len(m.previews) == 0 || n == 0 {
		return false
	}
	d := &m.previews[len(m.previews)-1].detail
	if n > 0 {
		d.Viewport.ScrollDown(n)
	} else {
		d.Viewport.ScrollUp(-n)
	}
	return true
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
	for i := len(m.previews) - 1; i >= 0; i-- {
		x, y, innerW, innerH := m.previewGeom(i)
		boxW, boxH := innerW+2, innerH+2
		inBox := dx >= x && dx < x+boxW && dy >= y && dy < y+boxH
		if inBox {
			return previewHit{
				kind:     previewHitContent,
				layer:    i,
				detail:   &m.previews[i].detail,
				bodyRow:  dy - y - 1,
				contentX: dx - x - 1,
			}
		}
		if i == 0 {
			if dx >= 0 && dx < w && dy >= 0 && dy < h {
				return previewHit{kind: previewHitGap, layer: i}
			}
			continue
		}
		px, py, pw, ph := m.previewGeom(i - 1)
		if dx >= px && dx < px+pw+2 && dy >= py && dy < py+ph+2 {
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
	out := base
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ui.Gold)
	for i, layer := range m.previews {
		x, y, innerW, innerH := m.previewGeom(i)
		if innerW < 20 || innerH < 6 {
			continue
		}
		layer.detail.SetPreviewSize(innerW, innerH)
		framed := boxStyle.Width(innerW).Height(innerH).Render(layer.detail.ContentView())
		out = stampRect(out, x, y, framed)
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
