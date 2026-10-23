package app

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

const (
	commentsDockMinExpanded  = 8
	commentsDockMinBody      = 4
	defaultCommentsDockRatio = 0.42
	replyPrefix              = "re:"

	dockHitHeader  = -1
	dockHitCompose = -2
	dockHitBlank   = -3
)

type commentThread struct {
	root    gastown.Comment
	replies []gastown.Comment
}

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
	return strings.Join(m.paintCommentsDock(width, height).lines, "\n")
}

type dockPaint struct {
	lines []string
	hits  []int
}

func (m Model) paintCommentsDock(width, height int) dockPaint {
	if width < 8 {
		width = 8
	}
	if height < 1 {
		height = 1
	}
	threads := m.commentThreads()
	n := 0
	for _, th := range threads {
		n += 1 + len(th.replies)
	}
	unread := m.unreadThreadCount(threads)
	cursor := m.clampedCommentCursor(len(threads))

	header := m.dockHeaderLine(n, unread, width)
	if !m.commentsDockOpen || height == 1 {
		return dockPaint{lines: []string{header}, hits: []int{dockHitHeader}}
	}

	innerW := width - 1
	if innerW < 8 {
		innerW = 8
	}

	var blocks []dockPaint
	for i, th := range threads {
		blocks = append(blocks, m.paintThread(th, i, i == cursor, innerW))
	}

	compose := goldBar(padDockLine(" "+ui.Cursor+" add a comment…", innerW), innerW)
	compose = lipgloss.NewStyle().Foreground(ui.Dim).Render(compose)

	topBar := goldBar(lipgloss.NewStyle().Foreground(ui.Gold).Render(strings.Repeat(ui.BoxHorizontal, innerW)), innerW)
	bodyH := height - 3 // top + header + compose
	if bodyH < 1 {
		bodyH = 1
	}
	body := windowCommentBlocks(blocks, cursor, m.commentsDockScroll, bodyH, innerW)

	lines := make([]string, 0, height)
	hits := make([]int, 0, height)
	lines = append(lines, topBar)
	hits = append(hits, dockHitHeader)
	lines = append(lines, header)
	hits = append(hits, dockHitHeader)
	lines = append(lines, body.lines...)
	hits = append(hits, body.hits...)
	for len(lines) < height-1 {
		lines = append(lines, goldBar(strings.Repeat(" ", innerW), innerW))
		hits = append(hits, dockHitBlank)
	}
	if len(lines) > height-1 {
		lines = lines[:height-1]
		hits = hits[:height-1]
	}
	lines = append(lines, compose)
	hits = append(hits, dockHitCompose)
	return dockPaint{lines: lines, hits: hits}
}

func (m Model) dockHeaderLine(n, unread, width int) string {
	title := fmt.Sprintf(" %s  %d comments", ui.SymCommentNerd, n)
	if unread > 0 {
		title = fmt.Sprintf(" %s  %d comments  ·  %d unread", ui.SymCommentNerd, n, unread)
	}
	fold := ui.SymDockFold
	if m.commentsDockOpen {
		fold = ui.SymDockUnfold
	}
	foldLabel := " " + fold + " "
	fill := width - 1 - lipgloss.Width(title) - lipgloss.Width(foldLabel)
	if fill < 1 {
		fill = 1
	}
	inner := title + strings.Repeat(" ", fill) + foldLabel
	if lipgloss.Width(inner) > width-1 {
		inner = ansi.Truncate(inner, width-1, "")
	}
	styled := lipgloss.NewStyle().
		Foreground(ui.Gold).
		Background(lipgloss.Color("#1a1a22")).
		Render(inner)
	if pad := (width - 1) - lipgloss.Width(styled); pad > 0 {
		styled += lipgloss.NewStyle().Background(lipgloss.Color("#1a1a22")).Render(strings.Repeat(" ", pad))
	}
	return goldBar(styled, width-1)
}

func (m Model) paintThread(th commentThread, idx int, selected bool, innerW int) dockPaint {
	rootBody := formatCommentBody(displayBody(th.root))
	unread := m.detail.Issue != nil && !m.commentIsSeen(m.detail.Issue.ID, th.root.ID)
	var raw []string
	var hits []int

	if selected {
		raw = append(raw, m.threadMetaLine(th.root, true, unread, innerW))
		hits = append(hits, idx)
		for _, line := range wrapCommentLines(rootBody, innerW-2) {
			raw = append(raw, " "+line)
			hits = append(hits, idx)
		}
		for _, rep := range th.replies {
			_, body := splitReply(rep)
			repUnread := m.detail.Issue != nil && !m.commentIsSeen(m.detail.Issue.ID, rep.ID)
			raw = append(raw, m.replyMetaLine(rep, selected, repUnread, innerW))
			hits = append(hits, idx)
			for _, line := range wrapCommentLines(formatCommentBody(body), innerW-6) {
				raw = append(raw, "    "+line)
				hits = append(hits, idx)
			}
		}
	} else {
		preview := strings.ReplaceAll(rootBody, "\n", " ")
		head := m.nameAndWhen(th.root, false, unread, " ")
		suffix := ""
		if n := len(th.replies); n > 0 {
			suffix = fmt.Sprintf("  %d", n)
		}
		line := head + "  " + lipgloss.NewStyle().Foreground(ui.Muted).Render(preview)
		if lipgloss.Width(line)+lipgloss.Width(suffix) > innerW {
			line = ansi.Truncate(line, innerW-lipgloss.Width(suffix), "…")
		}
		raw = append(raw, line+lipgloss.NewStyle().Foreground(ui.Dim).Render(suffix))
		hits = append(hits, idx)
	}

	out := dockPaint{lines: make([]string, 0, len(raw)), hits: hits}
	for i, ln := range raw {
		padded := padDockLine(ln, innerW)
		if selected && hits[i] == idx {
			padded = ui.DockSelectedRow(padded, innerW)
		}
		out.lines = append(out.lines, goldBar(padded, innerW))
	}
	return out
}

const nameTimeGap = "    " // one tab less

func (m Model) threadMetaLine(c gastown.Comment, selected, unread bool, innerW int) string {
	_ = innerW
	return m.nameAndWhen(c, selected, unread, " ")
}

func (m Model) replyMetaLine(c gastown.Comment, selected, unread bool, innerW int) string {
	_ = innerW
	return m.nameAndWhen(c, selected, unread, "    ")
}

func (m Model) nameAndWhen(c gastown.Comment, selected, unread bool, indent string) string {
	name := m.authorLabel(c.Author, selected, unread)
	clock := lipgloss.NewStyle().Foreground(ui.Muted).Render(commentClock(c.Time))
	out := indent + name + nameTimeGap + clock
	if selected {
		if date := commentDate(c.Time); date != "" {
			out += " " + lipgloss.NewStyle().Foreground(ui.Dim).Render(date)
		}
	}
	return out
}

func (m Model) authorLabel(name string, selected, unread bool) string {
	if name == "" {
		name = "?"
	}
	ink := authorColor(name, selected)
	s := lipgloss.NewStyle().Foreground(ink).Render(name)
	if unread {
		s = wavyUnderline(s)
	}
	return s
}

func (m Model) commentThreads() []commentThread {
	comments := m.dockComments()
	byID := map[string]gastown.Comment{}
	for _, c := range comments {
		byID[c.ID] = c
	}
	children := map[string][]gastown.Comment{}
	var roots []gastown.Comment
	for _, c := range comments {
		parent, _ := splitReply(c)
		if parent != "" && parent != c.ID {
			if _, ok := byID[parent]; ok {
				children[parent] = append(children[parent], c)
				continue
			}
		}
		roots = append(roots, c)
	}
	out := make([]commentThread, 0, len(roots))
	for _, root := range roots {
		out = append(out, commentThread{root: root, replies: children[root.ID]})
	}
	return out
}

func (m Model) unreadThreadCount(threads []commentThread) int {
	if m.detail.Issue == nil {
		return 0
	}
	n := 0
	for _, th := range threads {
		if !m.commentIsSeen(m.detail.Issue.ID, th.root.ID) {
			n++
			continue
		}
		for _, r := range th.replies {
			if !m.commentIsSeen(m.detail.Issue.ID, r.ID) {
				n++
				break
			}
		}
	}
	return n
}

func windowCommentBlocks(blocks []dockPaint, cursor, scroll, bodyH, innerW int) dockPaint {
	if bodyH < 1 {
		return dockPaint{}
	}
	if len(blocks) == 0 {
		empty := lipgloss.NewStyle().Foreground(ui.Muted).Render("  no comments yet")
		return dockPaint{lines: []string{goldBar(padDockLine(empty, innerW), innerW)}, hits: []int{dockHitBlank}}
	}
	var allLines []string
	var allHits []int
	starts := make([]int, len(blocks))
	for i, b := range blocks {
		starts[i] = len(allLines)
		allLines = append(allLines, b.lines...)
		allHits = append(allHits, b.hits...)
	}
	total := len(allLines)
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(blocks) {
		cursor = len(blocks) - 1
	}
	curStart := starts[cursor]
	curEnd := curStart + len(blocks[cursor].lines)
	off := 0
	if scroll > 0 {
		off = starts[min(scroll, len(blocks)-1)]
	}
	if curEnd-curStart >= bodyH {
		off = curStart
	} else {
		if curStart < off {
			off = curStart
		}
		if curEnd > off+bodyH {
			off = curEnd - bodyH
		}
	}
	if off < 0 {
		off = 0
	}
	if total > bodyH && off > total-bodyH {
		off = total - bodyH
	}
	end := off + bodyH
	if end > total {
		end = total
	}
	return dockPaint{lines: allLines[off:end], hits: allHits[off:end]}
}

func goldBar(inner string, innerW int) string {
	if lipgloss.Width(inner) > innerW {
		inner = ansi.Truncate(inner, innerW, "")
	}
	if pad := innerW - lipgloss.Width(inner); pad > 0 {
		inner += strings.Repeat(" ", pad)
	}
	bar := lipgloss.NewStyle().Foreground(ui.Gold).Render(ui.BoxVertical)
	return bar + inner
}

func padDockLine(s string, width int) string {
	if lipgloss.Width(s) > width {
		s = ansi.Truncate(s, width, "…")
	}
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func (m Model) dockComments() []gastown.Comment {
	if m.detail.Issue == nil {
		return nil
	}
	if m.detail.CommentsIssueID == m.detail.Issue.ID {
		return m.detail.Comments
	}
	// Keep the last paint if a refetch wiped the id; showing 0 is the vanish bug.
	if m.detail.CommentsIssueID == "" && len(m.detail.Comments) > 0 {
		return m.detail.Comments
	}
	return nil
}

func (m Model) clampedCommentCursor(n int) int {
	if n <= 0 {
		return 0
	}
	c := m.commentsDockCursor
	if c < 0 {
		return 0
	}
	if c >= n {
		return n - 1
	}
	return c
}

func (m *Model) moveCommentCursor(delta int) {
	n := len(m.commentThreads())
	if n == 0 {
		return
	}
	m.commentsDockCursor = m.clampedCommentCursor(n) + delta
	if m.commentsDockCursor < 0 {
		m.commentsDockCursor = 0
	}
	if m.commentsDockCursor >= n {
		m.commentsDockCursor = n - 1
	}
	if m.commentsDockScroll > m.commentsDockCursor {
		m.commentsDockScroll = m.commentsDockCursor
	}
	m.markSeenAtCursor()
}

func (m *Model) markSeenAtCursor() {
	threads := m.commentThreads()
	if m.detail.Issue == nil {
		return
	}
	i := m.clampedCommentCursor(len(threads))
	if i < 0 || i >= len(threads) {
		return
	}
	m.markCommentSeen(m.detail.Issue.ID, threads[i].root.ID)
	for _, r := range threads[i].replies {
		m.markCommentSeen(m.detail.Issue.ID, r.ID)
	}
}

func threadIndexFor(threads []commentThread, id string) (int, bool) {
	if id == "" {
		return 0, false
	}
	for i, th := range threads {
		if th.root.ID == id {
			return i, true
		}
		for _, r := range th.replies {
			if r.ID == id {
				return i, true
			}
		}
	}
	return 0, false
}

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
	headerRow := top
	if m.commentsDockOpen && m.commentsDockHeight() > 1 {
		headerRow = top + 1
	}
	if detailRow != headerRow && detailRow != top {
		return false
	}
	return detailX >= m.detail.Width-4
}

func (m Model) commentsDockHeaderHit(detailRow int) bool {
	if m.detail.Issue == nil {
		return false
	}
	top := m.commentsDockTopRow()
	if detailRow == top {
		return true
	}
	return m.commentsDockOpen && detailRow == top+1
}

func (m Model) commentsDockContains(detailRow int) bool {
	top := m.commentsDockTopRow()
	if top < 0 {
		return false
	}
	return detailRow >= top && detailRow < m.detail.Height
}

func (m Model) commentsDockHitAt(detailRow int) int {
	top := m.commentsDockTopRow()
	if top < 0 {
		return dockHitBlank
	}
	idx := detailRow - top
	paint := m.paintCommentsDock(m.detail.Width, m.commentsDockHeight())
	if idx < 0 || idx >= len(paint.hits) {
		return dockHitBlank
	}
	return paint.hits[idx]
}

func (m *Model) hoverCommentsDock(detailX, detailRow int) {
	// Unread is click-to-read, not hover. Motion is ignored on purpose.
	_, _ = detailX, detailRow
}

func (m *Model) selectCommentAt(detailRow int) bool {
	hit := m.commentsDockHitAt(detailRow)
	if hit == dockHitCompose {
		return true
	}
	if hit < 0 {
		return false
	}
	m.commentsFocus = true
	m.commentsDockOpen = true
	m.commentsDockCursor = hit
	m.markSeenAtCursor()
	return true
}

func (m *Model) startCommentReply() tea.Cmd {
	issue := m.parade.SelectedIssue
	if issue == nil {
		return nil
	}
	threads := m.commentThreads()
	if len(threads) == 0 {
		return nil
	}
	th := threads[m.clampedCommentCursor(len(threads))]
	m.replyToID = th.root.ID
	m.replyToAuthor = th.root.Author
	m.commentsDockOpen = true
	m.commentsFocus = true
	who := th.root.Author
	if who == "" {
		who = "comment"
	}
	return m.startQuickAction("reply", issue.ID, "reply> ", "reply to "+who+"…")
}

func (m Model) selectedThread() (commentThread, bool) {
	threads := m.commentThreads()
	if len(threads) == 0 {
		return commentThread{}, false
	}
	return threads[m.clampedCommentCursor(len(threads))], true
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
	if m.seenCommentIDs[issueID][commentID] {
		return
	}
	m.seenCommentIDs[issueID][commentID] = true
	saveSeenComments(m.projectDir, m.seenCommentIDs)
}

func encodeReply(parentID, body string) string {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return body
	}
	return replyPrefix + parentID + "\n" + body
}

func splitReply(c gastown.Comment) (parentID, body string) {
	body = c.Body
	if !strings.HasPrefix(body, replyPrefix) {
		return "", body
	}
	rest := strings.TrimPrefix(body, replyPrefix)
	id, rest, ok := strings.Cut(rest, "\n")
	if !ok {
		return "", body
	}
	return strings.TrimSpace(id), rest
}

func displayBody(c gastown.Comment) string {
	_, body := splitReply(c)
	return body
}

func formatCommentBody(s string) string {
	s = strings.ReplaceAll(s, "</hcom>", "\n</hcom>")
	s = strings.ReplaceAll(s, "\n\n</hcom>", "\n</hcom>")
	return strings.TrimSpace(s)
}

func parseCommentTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z"} {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t.Local(), true
		}
	}
	return time.Time{}, false
}

func commentClock(raw string) string {
	if t, ok := parseCommentTime(raw); ok {
		return t.Format("15:04")
	}
	if len(raw) >= 16 {
		return raw[11:16]
	}
	return raw
}

func commentDate(raw string) string {
	if t, ok := parseCommentTime(raw); ok {
		return t.Format("2 Jan")
	}
	return ""
}

func compactCommentTime(raw string) string {
	c, d := commentClock(raw), commentDate(raw)
	if d == "" {
		return c
	}
	return c + " " + d
}

func wrapCommentLines(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	if strings.TrimSpace(s) == "" {
		return []string{lipgloss.NewStyle().Foreground(ui.Dim).Render("(empty)")}
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}
		var line string
		for _, word := range strings.Fields(para) {
			if line == "" {
				if lipgloss.Width(word) <= width {
					line = word
					continue
				}
				for lipgloss.Width(word) > width {
					cut := cutWidth(word, width)
					out = append(out, word[:cut])
					word = word[cut:]
				}
				line = word
				continue
			}
			cand := line + " " + word
			if lipgloss.Width(cand) <= width {
				line = cand
				continue
			}
			out = append(out, line)
			line = word
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func cutWidth(s string, width int) int {
	w := 0
	for i, r := range s {
		rw := 1
		if r == '\t' {
			rw = 4
		}
		if w+rw > width {
			if i == 0 {
				return len(string(r))
			}
			return i
		}
		w += rw
	}
	return len(s)
}

func authorColor(name string, selected bool) color.Color {
	palette := []color.Color{
		lipgloss.Color("#5E8A7A"),
		lipgloss.Color("#7A6B9A"),
		lipgloss.Color("#8A7355"),
		lipgloss.Color("#5A7A8A"),
		lipgloss.Color("#8A5E6E"),
		lipgloss.Color("#6A8A5E"),
		lipgloss.Color("#8A7A4E"),
		lipgloss.Color("#6E5A8A"),
	}
	h := 0
	for _, r := range strings.ToLower(name) {
		h = 31*h + int(r)
	}
	if h < 0 {
		h = -h
	}
	base := palette[h%len(palette)]
	if selected {
		return brightenColor(base)
	}
	return base
}

func brightenColor(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	mix := func(v uint32) uint8 {
		x := int(v >> 8)
		return uint8(x + (255-x)/6)
	}
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", mix(r), mix(g), mix(b)))
}

func wavyUnderline(s string) string {
	// CSI 4:3 curly underline (kitty/vte). Unsupported terminals ignore it.
	return "\x1b[4:3m" + s + "\x1b[24m"
}
