// Package views provides the main TUI panels: the parade list, issue detail
// panel, Gas Town dashboard, and problems overlay.
package views

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// paradeSection defines how each parade group renders.
type paradeSection struct {
	Title          string
	Symbol         string
	Style          lipgloss.Style
	Color          color.Color
	State          data.SemanticState
	BorderVertical string
}

// sections is built per call (not a package var) so a theme switch at startup
// is reflected in the captured styles and pre-rendered borders. It iterates
// data.StateOrder() so the parade, the header, and the tmux widget can never
// disagree about which states exist or in what order.
func sections() []*paradeSection {
	var out []*paradeSection
	for _, state := range data.StateOrder() {
		c := ui.ExecColor(int(state))
		out = append(out, &paradeSection{
			Title:          state.Label(),
			Symbol:         ui.ExecSymbol(int(state)),
			Style:          ui.ExecSectionStyle(int(state)),
			Color:          c,
			State:          state,
			BorderVertical: lipgloss.NewStyle().Foreground(c).Render(ui.BoxVertical),
		})
	}
	return out
}

// ParadeItem is a renderable entry — a section header, footer, or issue.
type ParadeItem struct {
	IsHeader    bool
	IsFooter    bool
	Section     *paradeSection
	Issue       *data.Issue
	Eval        *data.DepEval
	State       data.SemanticState
	RenderedID  string
	Depth       int
	HasChildren bool
}

func (item ParadeItem) isSelectable() bool {
	return !item.IsHeader && !item.IsFooter
}

// Parade is the priority-sorted work tree view.
type Parade struct {
	Items           []ParadeItem
	Cursor          int
	Width           int
	Height          int
	ScrollOffset    int
	AllIssues       []data.Issue
	Groups          map[data.SemanticState][]data.Issue
	Unmapped        []data.Issue // issues DeriveState cannot classify; not rendered this wave
	issueMap        map[string]*data.Issue
	blockingTypes   map[string]bool
	SelectedIssue   *data.Issue
	ActiveAgents    map[string]string // issueID -> tmux window name
	TownStatus      *gastown.TownStatus
	ChangedIDs      map[string]bool  // recently changed issues (change indicator dot)
	OrphanedIDs     map[string]bool  // orphaned issues from dead rigs
	ZombieIDs       map[string]bool  // issues with dead agent sessions (zombie polecats)
	Selected        map[string]bool  // multi-selected issue IDs
	MatchHighlights map[string][]int // issueID -> matched char indices in title (fuzzy search)
	Collapsed       map[string]bool
}

func NewParade(issues []data.Issue, width, height int, blockingTypes map[string]bool) Parade {
	groups, unmapped := data.GroupBySemanticState(issues, blockingTypes)
	issueMap := data.BuildIssueMap(issues)
	return NewParadeWithData(issues, groups, unmapped, issueMap, width, height, blockingTypes)
}

// NewParadeWithData creates a parade view using precomputed grouping data.
func NewParadeWithData(
	issues []data.Issue,
	groups map[data.SemanticState][]data.Issue,
	unmapped []data.Issue,
	issueMap map[string]*data.Issue,
	width, height int,
	blockingTypes map[string]bool,
) Parade {
	if groups == nil {
		groups, unmapped = data.GroupBySemanticState(issues, blockingTypes)
	}
	if issueMap == nil {
		issueMap = data.BuildIssueMap(issues)
	}
	p := Parade{
		Width:         width,
		Height:        height,
		AllIssues:     issues,
		Groups:        groups,
		Unmapped:      unmapped,
		issueMap:      issueMap,
		blockingTypes: blockingTypes,
		Collapsed:     make(map[string]bool),
	}
	p.rebuildItems()
	if len(p.Items) > 0 {
		// Move cursor to first selectable item
		for i, item := range p.Items {
			if item.isSelectable() {
				p.Cursor = i
				p.SelectedIssue = item.Issue
				break
			}
		}
	}
	return p
}

// rebuildItems flattens groups into the renderable item list.
func (p *Parade) rebuildItems() {
	p.Items = nil
	var main []data.Issue
	for _, issue := range p.AllIssues {
		state, ok := data.DeriveState(&issue, p.issueMap, p.blockingTypes)
		if ok && isMainTreeState(state) {
			main = append(main, issue)
		}
	}
	p.appendForest(main)
}

// RebuildItems refreshes the flattened rows after restoring parade state.
func (p *Parade) RebuildItems() {
	p.rebuildItems()
}

func isMainTreeState(state data.SemanticState) bool {
	return state == data.StateReady || state == data.StateWorking ||
		state == data.StateWaitingBlocked || state == data.StateDeferred ||
		state == data.StateOperatorReview || state == data.StateDone
}

func orderForest(issues []data.Issue) (ordered []*data.Issue, depth map[string]int, hasChildren map[string]bool) {
	depth = make(map[string]int, len(issues))
	hasChildren = make(map[string]bool, len(issues))
	byID := make(map[string]*data.Issue, len(issues))
	for i := range issues {
		byID[issues[i].ID] = &issues[i]
	}
	children := make(map[string][]*data.Issue)
	for _, issue := range byID {
		parentID := issue.ParentRelationshipID()
		if parentID == "" || parentID == issue.ID {
			continue
		}
		if _, exists := byID[parentID]; exists {
			children[parentID] = append(children[parentID], issue)
		}
	}
	less := func(a, b *data.Issue) bool {
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.ID < b.ID
	}
	for parentID := range children {
		sort.Slice(children[parentID], func(i, j int) bool { return less(children[parentID][i], children[parentID][j]) })
		hasChildren[parentID] = true
	}
	var roots []*data.Issue
	for _, issue := range byID {
		parentID := issue.ParentRelationshipID()
		if parentID == "" || parentID == issue.ID {
			roots = append(roots, issue)
			continue
		}
		if _, exists := byID[parentID]; !exists {
			roots = append(roots, issue)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return less(roots[i], roots[j]) })
	visited := make(map[string]bool, len(issues))
	var walk func(*data.Issue, int)
	walk = func(issue *data.Issue, d int) {
		if issue == nil || visited[issue.ID] {
			return
		}
		visited[issue.ID] = true
		ordered = append(ordered, issue)
		depth[issue.ID] = d
		for _, child := range children[issue.ID] {
			walk(child, d+1)
		}
	}
	for _, root := range roots {
		walk(root, 0)
	}
	var remaining []*data.Issue
	for _, issue := range byID {
		if !visited[issue.ID] {
			remaining = append(remaining, issue)
		}
	}
	sort.Slice(remaining, func(i, j int) bool { return less(remaining[i], remaining[j]) })
	for _, issue := range remaining {
		walk(issue, 0)
	}
	return ordered, depth, hasChildren
}

func (p *Parade) appendForest(issues []data.Issue) {
	p.appendForestRows(issues, nil, data.StateReady)
}

func (p *Parade) appendForestRows(issues []data.Issue, sec *paradeSection, fallback data.SemanticState) {
	ordered, depths, children := orderForest(issues)
	for _, issue := range ordered {
		if p.hasCollapsedAncestor(issue.ID) {
			continue
		}
		actual := p.issueMap[issue.ID]
		if actual == nil {
			actual = issue
		}
		state, ok := data.DeriveState(actual, p.issueMap, p.blockingTypes)
		if !ok {
			state = fallback
		}
		eval := actual.EvaluateDependencies(p.issueMap, p.blockingTypes)
		p.Items = append(p.Items, ParadeItem{
			Section:     sec,
			Issue:       actual,
			Eval:        &eval,
			State:       state,
			RenderedID:  lipgloss.NewStyle().Foreground(statusColor(state)).Render(relativeDisplayID(actual, depths[issue.ID])),
			Depth:       depths[issue.ID],
			HasChildren: children[issue.ID],
		})
	}
}

func (p *Parade) hasCollapsedAncestor(issueID string) bool {
	seen := map[string]bool{issueID: true}
	for current := p.issueMap[issueID]; current != nil; {
		parentID := current.ParentRelationshipID()
		if parentID == "" || seen[parentID] {
			return false
		}
		if p.Collapsed[parentID] {
			return true
		}
		seen[parentID] = true
		current = p.issueMap[parentID]
	}
	return false
}

// relativeDisplayID is the one place a row's label is chosen: a root row
// always shows its full ID, and a nested row defers to data.RelativeDisplayID's
// edge-checked judgment.
func relativeDisplayID(iss *data.Issue, depth int) string {
	if depth <= 0 {
		return iss.ID
	}
	return data.RelativeDisplayID(iss)
}

// MoveUp moves the cursor up, skipping headers and footers.
func (p *Parade) MoveUp() {
	for i := p.Cursor - 1; i >= 0; i-- {
		if p.Items[i].isSelectable() {
			p.Cursor = i
			p.SelectedIssue = p.Items[i].Issue
			p.ensureVisible()
			return
		}
	}
}

// MoveDown moves the cursor down, skipping headers and footers.
func (p *Parade) MoveDown() {
	for i := p.Cursor + 1; i < len(p.Items); i++ {
		if p.Items[i].isSelectable() {
			p.Cursor = i
			p.SelectedIssue = p.Items[i].Issue
			p.ensureVisible()
			return
		}
	}
}

// ToggleNode toggles a non-leaf issue and preserves selection by full ID.
func (p *Parade) ToggleNode(issueID string) {
	hasChildren := false
	for _, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == issueID {
			hasChildren = item.HasChildren
			break
		}
	}
	if !hasChildren {
		return
	}
	selectedID := ""
	if p.SelectedIssue != nil {
		selectedID = p.SelectedIssue.ID
	}
	if p.Collapsed == nil {
		p.Collapsed = make(map[string]bool)
	}
	p.Collapsed[issueID] = !p.Collapsed[issueID]
	p.rebuildItems()
	p.restoreSelection(selectedID)
}

func (p *Parade) restoreSelection(selectedID string) {
	for selectedID != "" {
		for i, item := range p.Items {
			if item.isSelectable() && item.Issue.ID == selectedID {
				p.Cursor = i
				p.SelectedIssue = item.Issue
				p.ensureVisible()
				return
			}
		}
		selected := p.issueMap[selectedID]
		if selected == nil {
			break
		}
		selectedID = selected.ParentRelationshipID()
	}
	for i, item := range p.Items {
		if item.isSelectable() {
			p.Cursor = i
			p.SelectedIssue = item.Issue
			p.ensureVisible()
			return
		}
	}
	p.Cursor = 0
	p.ScrollOffset = 0
	p.SelectedIssue = nil
}

// IssueAtViewportRow resolves a visible issue row using the current scroll offset.
func (p *Parade) IssueAtViewportRow(row int) *data.Issue {
	if row < 0 || row >= p.Height {
		return nil
	}
	index := p.ScrollOffset + row
	if index < 0 || index >= len(p.Items) || !p.Items[index].isSelectable() {
		return nil
	}
	return p.Items[index].Issue
}

// clampScroll ensures ScrollOffset is within valid bounds for the current Items slice.
func (p *Parade) clampScroll() {
	maxOffset := len(p.Items) - p.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.ScrollOffset > maxOffset {
		p.ScrollOffset = maxOffset
	}
	if p.ScrollOffset < 0 {
		p.ScrollOffset = 0
	}
}

// ensureVisible adjusts scroll offset so cursor is visible.
func (p *Parade) ensureVisible() {
	if p.Cursor < p.ScrollOffset {
		p.ScrollOffset = p.Cursor
	}
	if p.Cursor >= p.ScrollOffset+p.Height {
		p.ScrollOffset = p.Cursor - p.Height + 1
	}
	p.clampScroll()
}

// ToggleSelect toggles multi-select on the issue at the cursor.
func (p *Parade) ToggleSelect() {
	if p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return
	}
	item := p.Items[p.Cursor]
	if !item.isSelectable() || item.Issue == nil {
		return
	}
	if p.Selected == nil {
		p.Selected = make(map[string]bool)
	}
	id := item.Issue.ID
	if p.Selected[id] {
		delete(p.Selected, id)
	} else {
		p.Selected[id] = true
	}
}

// ClearSelection removes all multi-selections.
func (p *Parade) ClearSelection() {
	p.Selected = nil
}

// SelectedIssues returns the list of multi-selected issues.
func (p *Parade) SelectedIssues() []*data.Issue {
	if len(p.Selected) == 0 {
		return nil
	}
	var result []*data.Issue
	for _, item := range p.Items {
		if item.Issue != nil && p.Selected[item.Issue.ID] {
			result = append(result, item.Issue)
		}
	}
	return result
}

// SelectionCount returns the number of multi-selected issues.
// VisibleIssues returns the number of issue rows currently in the parade
// (section headers and footers excluded). Used for the filter match count.
func (p *Parade) VisibleIssues() int {
	n := 0
	for _, item := range p.Items {
		if item.isSelectable() {
			n++
		}
	}
	return n
}

func (p *Parade) SelectionCount() int {
	return len(p.Selected)
}

// SetSize updates the available dimensions.
func (p *Parade) SetSize(width, height int) {
	p.Width = width
	p.Height = height
}

// View renders the parade list.
func (p *Parade) View() string {
	if len(p.Items) == 0 {
		content := "No issues found"
		return lipgloss.NewStyle().Width(p.Width).Height(p.Height).Render(content)
	}

	p.clampScroll()
	var lines []string
	end := p.ScrollOffset + p.Height
	if end > len(p.Items) {
		end = len(p.Items)
	}
	for idx, item := range p.Items[p.ScrollOffset:end] {
		globalIdx := p.ScrollOffset + idx
		switch {
		case item.IsHeader:
			lines = append(lines, p.renderBorderTop(item.Section))
		case item.IsFooter:
			lines = append(lines, p.renderBorderBottom(item.Section))
		default:
			dist := globalIdx - p.Cursor
			if dist < 0 {
				dist = -dist
			}
			lines = append(lines, p.renderIssue(item, globalIdx == p.Cursor, dist))
		}
	}
	free := p.Height - len(lines)
	padLine := strings.Repeat(" ", p.Width)
	for i := 0; i < free; i++ {
		lines = append(lines, padLine)
	}
	return strings.Join(lines, "\n")
}

// renderBorderTop builds a top border line for an attention section.
func (p *Parade) renderBorderTop(sec *paradeSection) string {
	count := len(p.Groups[sec.State])
	borderStyle := lipgloss.NewStyle().Foreground(sec.Color)
	titleText := fmt.Sprintf("%s %s%s", sec.Symbol, sec.Title, ui.Superscript(count))
	coloredTitle := sec.Style.Render(titleText)
	titleWidth := lipgloss.Width(coloredTitle)
	prefix := borderStyle.Render(ui.BoxTopLeft + ui.BoxHorizontal + " ")
	suffix := borderStyle.Render(" " + ui.BoxTopRight)
	prefixW := lipgloss.Width(prefix)
	suffixW := lipgloss.Width(suffix)
	availableForTitle := p.Width - prefixW - suffixW - 1
	if titleWidth > availableForTitle && availableForTitle > 0 {
		titleText = truncate(titleText, availableForTitle)
		coloredTitle = sec.Style.Render(titleText)
		titleWidth = lipgloss.Width(coloredTitle)
	}
	fillLen := p.Width - prefixW - titleWidth - 1 - suffixW
	if fillLen < 1 {
		fillLen = 1
	}
	fill := borderStyle.Render(" " + strings.Repeat(ui.BoxHorizontal, fillLen))
	return prefix + coloredTitle + fill + suffix
}

// renderBorderBottom builds a bottom border line for an attention section.
func (p *Parade) renderBorderBottom(sec *paradeSection) string {
	borderStyle := lipgloss.NewStyle().Foreground(sec.Color)
	cornerL := borderStyle.Render(ui.BoxBottomLeft)
	cornerR := borderStyle.Render(ui.BoxBottomRight)
	cornersW := lipgloss.Width(cornerL) + lipgloss.Width(cornerR)
	fillLen := p.Width - cornersW
	if fillLen < 1 {
		fillLen = 1
	}
	fill := borderStyle.Render(strings.Repeat(ui.BoxHorizontal, fillLen))
	return cornerL + fill + cornerR
}

// renderIssue renders an issue row wrapped in │ section borders.
// distFromCursor controls positional fading (btop-style depth effect).
func (p *Parade) renderIssue(item ParadeItem, selected bool, distFromCursor int) string {
	issue := item.Issue

	// The row's glyph and ID color come from the derived semantic state.
	symStr := ui.ExecIndicator(int(item.State))
	var prioStr string
	switch issue.Priority {
	case 0:
		prioStr = ui.BadgeP0
	case 1:
		prioStr = ui.BadgeP1
	case 2:
		prioStr = ui.BadgeP2
	case 3:
		prioStr = ui.BadgeP3
	default:
		prioStr = ui.BadgeP4
	}
	// Closed issues render muted throughout — done work shouldn't compete
	// with live work for attention (audit #9).
	isClosed := issue.Status == data.StatusClosed
	if isClosed {
		prioStr = lipgloss.NewStyle().Foreground(ui.Muted).Render(fmt.Sprintf("P%d", issue.Priority))
	}
	pinBadge := ""
	pinWidth := 0
	if issue.Pinned {
		pinBadge = " " + ui.BadgePriority.Render("PIN")
		pinWidth = lipgloss.Width(pinBadge)
	}

	innerWidth := p.Width
	leftBorder, rightBorder := "", ""
	if item.Section != nil {
		innerWidth = p.Width - 4
		leftBorder = item.Section.BorderVertical
		rightBorder = item.Section.BorderVertical
	}
	compactBadges := innerWidth < 70

	// Multi-select checkbox
	selectPrefix := ""
	selectWidth := 0
	if len(p.Selected) > 0 {
		if p.Selected[issue.ID] {
			selectPrefix = lipgloss.NewStyle().Foreground(ui.BrightGold).Bold(true).Render(ui.SymSelected) + " "
		} else {
			selectPrefix = lipgloss.NewStyle().Foreground(ui.Dim).Render(ui.SymUnselected) + " "
		}
		selectWidth = 2
	}

	// Change indicator dot
	changePrefix := ""
	changeWidth := 0
	if p.ChangedIDs != nil && p.ChangedIDs[issue.ID] {
		changePrefix = lipgloss.NewStyle().Foreground(ui.BrightGold).Render(ui.SymChanged) + " "
		changeWidth = 2
	}

	// Orphan indicator (dead rig)
	orphanPrefix := ""
	orphanWidth := 0
	if p.OrphanedIDs != nil && p.OrphanedIDs[issue.ID] {
		orphanPrefix = lipgloss.NewStyle().Foreground(ui.StatusStalled).Render(ui.SymDeadRig) + " "
		orphanWidth = 2
	}

	// Zombie indicator (dead agent session, not a full dead rig)
	zombiePrefix := ""
	zombieWidth := 0
	if p.ZombieIDs != nil && p.ZombieIDs[issue.ID] && orphanWidth == 0 {
		zombiePrefix = lipgloss.NewStyle().Foreground(ui.StatusStalled).Render(ui.SymZombie) + " "
		zombieWidth = 2
	}

	// Agent badge prefix
	agentPrefix := ""
	agentWidth := 0
	if p.ActiveAgents != nil {
		if _, active := p.ActiveAgents[issue.ID]; active {
			if p.TownStatus != nil {
				// Gas Town: show named agent
				if a := p.TownStatus.AgentForIssue(issue.ID); a != nil {
					label := fmt.Sprintf("%s %s", ui.SymAgent, a.Name)
					agentPrefix = ui.AgentBadge.Render(label) + " "
					agentWidth = len(a.Name) + 2 // symbol + space + name + space
				} else {
					agentPrefix = ui.AgentBadge.Render(ui.SymAgent) + " "
					agentWidth = 2
				}
			} else {
				// Tmux-only: generic badge
				agentPrefix = ui.AgentBadge.Render(ui.SymAgent) + " "
				agentWidth = 2
			}
		}
	}

	// Hierarchical indent is computed per section by data.OrderHierarchically,
	// which places each child directly beneath its parent. It is deliberately
	// section-relative: a child whose parent sits in another section renders at
	// depth 0 rather than appearing to belong to whatever row precedes it.
	depth := item.Depth
	indent := strings.Repeat("  ", depth)
	indentWidth := depth * 2
	// Due date badge. Under width pressure the badge compresses ("▲151d")
	// so it never crowds out the title (audit #2).
	dueBadge := ""
	dueWidth := 0
	if issue.IsOverdue() {
		label := fmt.Sprintf("%s %s", ui.SymOverdue, issue.DueLabel())
		if compactBadges {
			label = ui.SymOverdue + strings.Fields(issue.DueLabel())[0]
		}
		dueBadge = " " + ui.OverdueBadge.Render(label)
		dueWidth = lipgloss.Width(dueBadge)
	} else if issue.DueAt != nil && issue.Status != data.StatusClosed {
		days := int(time.Until(*issue.DueAt).Hours() / 24)
		if days <= 3 {
			label := fmt.Sprintf("%s %s", ui.SymDueDate, issue.DueLabel())
			if compactBadges {
				label = ui.SymDueDate + strings.Fields(issue.DueLabel())[0]
			}
			dueBadge = " " + ui.DueSoonBadge.Render(label)
			dueWidth = lipgloss.Width(dueBadge)
		}
	}

	// Deferred badge
	deferBadge := ""
	deferWidth := 0
	if issue.IsDeferred() {
		deferBadge = " " + ui.DeferredStyle.Render(ui.SymDeferred)
		deferWidth = 2
	}

	// Comment badge — shows that an issue carries discussion without opening
	// it. Free from `bd list --json`, so it costs no extra CLI call. Width is
	// measured rather than assumed, since the glyph is wide.
	commentBadge := ""
	commentWidth := 0
	if issue.CommentCount > 0 {
		label := ui.SymComment + strconv.Itoa(issue.CommentCount)
		commentBadge = " " + ui.CommentBadge.Render(label)
		commentWidth = lipgloss.Width(commentBadge)
	}

	maxTitle := innerWidth - 16 - agentWidth - changeWidth - selectWidth - indentWidth - dueWidth - deferWidth - commentWidth - orphanWidth - zombieWidth - pinWidth
	if maxTitle < 0 {
		maxTitle = 0
	}
	title := truncate(issue.Title, maxTitle)

	// Apply dim styling to deferred issue titles, or highlight fuzzy matches.
	var renderedTitle string
	if indices, ok := p.MatchHighlights[issue.ID]; ok && len(indices) > 0 {
		renderedTitle = ui.HighlightMatches(title, indices, maxTitle)
	} else {
		titleStyle := lipgloss.NewStyle()
		if issue.IsDeferred() {
			titleStyle = ui.DeferredStyle
		}
		if isClosed {
			titleStyle = lipgloss.NewStyle().Foreground(ui.Muted)
		}
		renderedTitle = titleStyle.Render(title)
	}
	renderedID := item.RenderedID

	line := fmt.Sprintf("%s%s %s%s%s%s%s%s %s %s",
		indent,
		symStr,
		selectPrefix,
		changePrefix,
		orphanPrefix,
		zombiePrefix,
		agentPrefix,
		renderedID,
		renderedTitle,
		prioStr,
	)
	line += pinBadge + dueBadge + deferBadge + commentBadge

	if selected {
		cursor := ui.ItemCursor.Render(ui.Cursor + " ")
		content := ui.SelectedRow(cursor+line, innerWidth)
		if item.Section == nil {
			return content
		}
		return leftBorder + " " + content + " " + rightBorder
	}

	// Non-selected: pad with leading space for alignment (matching cursor indent)
	row := "  " + line
	if padLen := innerWidth - lipgloss.Width(row); padLen > 0 {
		row += strings.Repeat(" ", padLen)
	}
	content := ansi.Truncate(row, innerWidth, "")

	// Positional fade: items far from cursor get dimmed (btop-style depth)
	if distFromCursor > 6 {
		content = lipgloss.NewStyle().Faint(true).Render(content)
	}

	if item.Section == nil {
		return content
	}
	return leftBorder + " " + content + " " + rightBorder
}

// statusSymbol returns the glyph for an issue's derived execution state. The
// state comes from data.DeriveState — an issue whose state cannot be derived
// has no symbol here, and callers render its raw status instead.
func statusSymbol(state data.SemanticState) string {
	return ui.ExecSymbol(int(state))
}

// statusColor returns the palette color for an issue's derived execution state.
func statusColor(state data.SemanticState) color.Color {
	return ui.ExecColor(int(state))
}
