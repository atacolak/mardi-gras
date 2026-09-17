package views

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// paradeIssues returns a mix of open, in_progress, and closed issues for testing.
func paradeIssues() []data.Issue {
	return []data.Issue{
		{ID: "mg-001", Title: "Rolling issue", Status: data.StatusInProgress, Priority: data.PriorityHigh, IssueType: data.TypeTask},
		{ID: "mg-002", Title: "Lined up issue", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeFeature},
		{ID: "mg-003", Title: "Another open issue", Status: data.StatusOpen, Priority: data.PriorityLow, IssueType: data.TypeBug},
		{ID: "mg-004", Title: "Closed issue", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask},
		{ID: "mg-005", Title: "Another closed", Status: data.StatusClosed, Priority: data.PriorityBacklog, IssueType: data.TypeChore},
	}
}

func newTestParade() Parade {
	return NewParade(paradeIssues(), 80, 20, data.DefaultBlockingTypes)
}

func TestNewParadeInitialCursor(t *testing.T) {
	p := newTestParade()

	if len(p.Items) == 0 {
		t.Fatal("expected items, got none")
	}

	item := p.Items[p.Cursor]
	if item.IsHeader || item.IsFooter {
		t.Fatalf("cursor at %d is on a non-selectable item (header=%v, footer=%v)", p.Cursor, item.IsHeader, item.IsFooter)
	}
	if item.Issue == nil {
		t.Fatal("cursor item has nil issue")
	}
}

func TestMoveDownUp(t *testing.T) {
	p := newTestParade()
	startCursor := p.Cursor
	startIssue := p.SelectedIssue

	p.MoveDown()
	if p.Cursor == startCursor {
		t.Fatal("MoveDown did not advance cursor")
	}
	if p.Items[p.Cursor].IsHeader || p.Items[p.Cursor].IsFooter {
		t.Fatal("MoveDown landed on non-selectable item")
	}
	if p.SelectedIssue == startIssue {
		t.Fatal("MoveDown did not update SelectedIssue")
	}

	downIssue := p.SelectedIssue
	p.MoveUp()
	if p.SelectedIssue == downIssue {
		t.Fatal("MoveUp did not change SelectedIssue")
	}
	if p.SelectedIssue.ID != startIssue.ID {
		t.Fatalf("MoveUp did not return to original issue: got %s, want %s", p.SelectedIssue.ID, startIssue.ID)
	}
}

func TestMoveDownClampsAtEnd(t *testing.T) {
	p := newTestParade()

	// Move to the very end
	for i := 0; i < len(p.Items)+10; i++ {
		p.MoveDown()
	}
	lastCursor := p.Cursor
	lastIssue := p.SelectedIssue

	p.MoveDown()
	if p.Cursor != lastCursor {
		t.Fatalf("MoveDown past end changed cursor: got %d, want %d", p.Cursor, lastCursor)
	}
	if p.SelectedIssue.ID != lastIssue.ID {
		t.Fatal("MoveDown past end changed SelectedIssue")
	}
}

func TestMoveUpClampsAtTop(t *testing.T) {
	p := newTestParade()
	firstCursor := p.Cursor
	firstIssue := p.SelectedIssue

	p.MoveUp()
	if p.Cursor != firstCursor {
		t.Fatalf("MoveUp past top changed cursor: got %d, want %d", p.Cursor, firstCursor)
	}
	if p.SelectedIssue.ID != firstIssue.ID {
		t.Fatal("MoveUp past top changed SelectedIssue")
	}
}

func TestToggleClosed(t *testing.T) {
	p := newTestParade()
	if p.ShowClosed {
		t.Fatal("ShowClosed should default to false")
	}

	// Count items before toggle (closed section collapsed)
	countBefore := len(p.Items)

	p.ToggleClosed()
	if !p.ShowClosed {
		t.Fatal("ToggleClosed did not flip ShowClosed to true")
	}
	if len(p.Items) <= countBefore {
		t.Fatalf("toggling closed on should add items: before=%d, after=%d", countBefore, len(p.Items))
	}

	p.ToggleClosed()
	if p.ShowClosed {
		t.Fatal("second ToggleClosed did not flip ShowClosed back to false")
	}
	if len(p.Items) != countBefore {
		t.Fatalf("toggling closed off should restore count: got=%d, want=%d", len(p.Items), countBefore)
	}
}

func TestToggleClosedPreservesSelection(t *testing.T) {
	p := newTestParade()

	// Select second issue
	p.MoveDown()
	selectedID := p.SelectedIssue.ID

	p.ToggleClosed()
	if p.SelectedIssue == nil || p.SelectedIssue.ID != selectedID {
		got := "<nil>"
		if p.SelectedIssue != nil {
			got = p.SelectedIssue.ID
		}
		t.Fatalf("ToggleClosed changed selection: got %s, want %s", got, selectedID)
	}
}

func TestToggleSelect(t *testing.T) {
	p := newTestParade()
	issueID := p.SelectedIssue.ID

	p.ToggleSelect()
	if !p.Selected[issueID] {
		t.Fatalf("ToggleSelect did not add %s to Selected", issueID)
	}
}

func TestToggleSelectDeselects(t *testing.T) {
	p := newTestParade()
	issueID := p.SelectedIssue.ID

	p.ToggleSelect()
	p.ToggleSelect()
	if p.Selected[issueID] {
		t.Fatalf("second ToggleSelect did not remove %s from Selected", issueID)
	}
}

func TestClearSelection(t *testing.T) {
	p := newTestParade()

	p.ToggleSelect()
	p.MoveDown()
	p.ToggleSelect()

	if len(p.Selected) != 2 {
		t.Fatalf("expected 2 selected, got %d", len(p.Selected))
	}

	p.ClearSelection()
	if len(p.Selected) != 0 {
		t.Fatalf("ClearSelection did not empty Selected: got %d", len(p.Selected))
	}
}

func TestSelectedIssues(t *testing.T) {
	p := newTestParade()
	firstID := p.SelectedIssue.ID
	p.ToggleSelect()

	p.MoveDown()
	secondID := p.SelectedIssue.ID
	p.ToggleSelect()

	result := p.SelectedIssues()
	if len(result) != 2 {
		t.Fatalf("expected 2 selected issues, got %d", len(result))
	}

	ids := map[string]bool{}
	for _, iss := range result {
		ids[iss.ID] = true
	}
	if !ids[firstID] || !ids[secondID] {
		t.Fatalf("SelectedIssues missing expected IDs: want %s and %s, got %v", firstID, secondID, ids)
	}
}

func TestSelectionCount(t *testing.T) {
	p := newTestParade()

	if p.SelectionCount() != 0 {
		t.Fatalf("expected 0, got %d", p.SelectionCount())
	}

	p.ToggleSelect()
	if p.SelectionCount() != 1 {
		t.Fatalf("expected 1, got %d", p.SelectionCount())
	}

	p.MoveDown()
	p.ToggleSelect()
	if p.SelectionCount() != 2 {
		t.Fatalf("expected 2, got %d", p.SelectionCount())
	}
}

func TestEnsureVisible(t *testing.T) {
	// Use a very small viewport height so scrolling is triggered
	issues := paradeIssues()
	p := NewParade(issues, 80, 3, data.DefaultBlockingTypes)

	// Move down enough to go past the viewport
	for i := 0; i < 10; i++ {
		p.MoveDown()
	}

	// Cursor should be visible: within [ScrollOffset, ScrollOffset+Height)
	if p.Cursor < p.ScrollOffset || p.Cursor >= p.ScrollOffset+p.Height {
		t.Fatalf("cursor %d not visible in viewport [%d, %d)", p.Cursor, p.ScrollOffset, p.ScrollOffset+p.Height)
	}
}

func TestParadeIndentUsesParentRelationships(t *testing.T) {
	issues := []data.Issue{
		{ID: "root-001", Title: "Root", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
		{ID: "child001", Title: "Child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, Dependencies: []data.Dependency{{IssueID: "child001", DependsOnID: "root-001", Type: "parent-child"}}},
		{ID: "past.001", Title: "Former child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	positions := make(map[string]int)
	for _, item := range p.Items {
		if item.Issue == nil {
			continue
		}
		row := ansi.Strip(p.renderIssue(item, false, 0))
		positions[item.Issue.ID] = strings.Index(row, item.Issue.ID)
	}

	if got, want := positions["child001"], positions["root-001"]+2; got != want {
		t.Errorf("actual child ID starts at column %d, want %d", got, want)
	}
	if got, want := positions["past.001"], positions["root-001"]; got != want {
		t.Errorf("dotted top-level ID starts at column %d, want %d", got, want)
	}
}

// Under an epic the dotted prefix is redundant — the row already sits in the
// epic's own section — so "mard-nob.7" renders as ".7". The parade does not
// decide this itself: data.RelativeDisplayID compares the issue's dotted prefix
// against its own parent-child EDGE, so a reparented issue keeps its stale
// dotted ID in full and an edge-only child keeps its undotted one. A root row
// is never compacted. The row stays keyed by its real ID either way; only the
// label changes.
func TestParadeRelativeDisplayID(t *testing.T) {
	epic := data.Issue{ID: "mard-nob", Title: "Epic", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeEpic}
	nested := data.Issue{
		ID: "mard-nob.7", Title: "Nested child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{{IssueID: "mard-nob.7", DependsOnID: "mard-nob", Type: "parent-child"}},
	}
	reparented := data.Issue{
		ID: "legacy.2", Title: "Reparented child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{{IssueID: "legacy.2", DependsOnID: "mard-nob", Type: "parent-child"}},
	}
	edgeOnly := data.Issue{
		ID: "edge-child", Title: "Edge-only child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{{IssueID: "edge-child", DependsOnID: "mard-nob", Type: "parent-child"}},
	}

	p := NewParade([]data.Issue{epic, nested, reparented, edgeOnly}, 100, 30, data.DefaultBlockingTypes)

	got := make(map[string]string)
	for _, item := range p.Items {
		if item.Issue == nil {
			continue
		}
		got[item.Issue.ID] = ansi.Strip(item.RenderedID)
		if item.Issue.ID == "mard-nob.7" {
			t.Logf("nested row: %s", ansi.Strip(p.renderIssue(item, false, 0)))
		}
	}

	want := map[string]string{
		"mard-nob":   "mard-nob",
		"mard-nob.7": ".7",
		"legacy.2":   "legacy.2",
		"edge-child": "edge-child",
	}
	for id, wantID := range want {
		if got[id] != wantID {
			t.Errorf("rendered ID for %s = %q, want %q", id, got[id], wantID)
		}
	}
	if len(got) != len(want) {
		t.Errorf("rendered %d issue rows, want %d: %v", len(got), len(want), got)
	}

	// A compacted label is display only. The row must keep its real ID — it is
	// the key the cursor, multi-select, and change indicators resolve through.
	for _, item := range p.Items {
		if item.Issue == nil || item.Issue.ID != "mard-nob.7" {
			continue
		}
		if got := item.Issue.ID; got != "mard-nob.7" {
			t.Errorf("nested row key = %q, want the full ID", got)
		}
		if item.Depth != 1 {
			t.Errorf("nested row depth = %d, want 1", item.Depth)
		}
		if !item.isSelectable() {
			t.Error("nested row must stay selectable")
		}
	}
}

// The Done section builds its rows through the same append, so a nested closed
// row compacts under its epic as well. The observable is the rendered row: the
// closed path re-renders the label in the muted style, and that re-render must
// not quietly discard the compaction.
func TestParadeRelativeDisplayIDClosed(t *testing.T) {
	epic := data.Issue{ID: "mard-nob", Title: "Epic", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeEpic}
	nested := data.Issue{
		ID: "mard-nob.7", Title: "Closed child", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{{IssueID: "mard-nob.7", DependsOnID: "mard-nob", Type: "parent-child"}},
	}
	p := NewParade([]data.Issue{epic, nested}, 100, 30, data.DefaultBlockingTypes)
	p.ToggleClosed()

	out := ansi.Strip(p.View())
	if strings.Contains(out, "mard-nob.7") {
		t.Errorf("closed nested row should not repeat the redundant prefix:\n%s", out)
	}
	// The compacted label is a substring of the full one — "mard-nob.7 Closed
	// child" satisfies a bare Contains(".7 Closed child") with no compaction
	// having happened at all — so assert the label the way the row actually
	// prints it: an ID token with its own leading separator, not a fragment of
	// the longer ID.
	row := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Closed child") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("no rendered row for the closed child:\n%s", out)
	}
	if !strings.Contains(row, " .7 Closed child") {
		t.Errorf("closed nested row does not print the compacted ID as a token: %q", row)
	}
	if !strings.Contains(out, "mard-nob Epic") {
		t.Errorf("depth-0 root should keep its full ID:\n%s", out)
	}
}

// A matching-prefix child whose parent is not in its own section renders at
// depth 0, where the full ID is the only resolvable label: nothing above it
// repeats the prefix, so ".7" would be a row the operator cannot look up. Real
// boards reach this shape — a closed child lands in Done while its open epic is
// still in Ready, and a blocked child lands in Waiting/Blocked while its epic
// is Ready — which is why relativeDisplayID takes depth at all instead of
// always deferring to data.RelativeDisplayID.
func TestParadeRelativeDisplayIDKeepsFullIDForOutOfSectionParent(t *testing.T) {
	epic := data.Issue{ID: "mard-nob", Title: "Epic", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeEpic}
	blocked := data.Issue{
		ID: "mard-nob.7", Title: "Blocked child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{
			{IssueID: "mard-nob.7", DependsOnID: "mard-nob", Type: "parent-child"},
			{IssueID: "mard-nob.7", DependsOnID: "mard-boss", Type: "blocks"},
		},
	}
	boss := data.Issue{ID: "mard-boss", Title: "Boss", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask}

	p := NewParade([]data.Issue{epic, blocked, boss}, 100, 30, data.DefaultBlockingTypes)

	var seen bool
	for _, item := range p.Items {
		if item.Issue == nil || item.Issue.ID != "mard-nob.7" {
			continue
		}
		seen = true
		if item.Depth != 0 {
			t.Fatalf("blocked child depth = %d, want 0 — its epic is in another section", item.Depth)
		}
		if got := ansi.Strip(item.RenderedID); got != "mard-nob.7" {
			t.Errorf("rendered ID = %q, want the full %q at depth 0", got, "mard-nob.7")
		}
	}
	if !seen {
		var ids []string
		for _, item := range p.Items {
			if item.Issue != nil {
				ids = append(ids, item.Issue.ID)
			}
		}
		t.Fatalf("blocked child missing from the parade: %v", ids)
	}
}

// An issue whose execution state cannot be derived is carried on the parade,
// not folded into a bucket. Rendering it is an open product question, so it
// must not appear as Ready work — and must not vanish from the data either.
func TestParadeCarriesUnmappedIssues(t *testing.T) {
	issues := []data.Issue{
		{ID: "open-1", Title: "Ready", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
		{ID: "draft-1", Title: "Draft", Status: data.StatusDraft, Priority: data.PriorityMedium, IssueType: data.TypeTask},
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)

	if len(p.Unmapped) != 1 || p.Unmapped[0].ID != "draft-1" {
		t.Fatalf("expected draft-1 carried as unmapped, got %v", p.Unmapped)
	}
	for _, state := range data.StateOrder() {
		for _, iss := range p.Groups[state] {
			if iss.ID == "draft-1" {
				t.Fatalf("unmapped issue counted as %s", state.Label())
			}
		}
	}
	if out := ansi.Strip(p.View()); strings.Contains(out, "draft-1") {
		t.Errorf("unmapped issue should not render as a state row:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Comment count badge
// ---------------------------------------------------------------------------

// comment_count rides along on `bd list --json`, so the parade can show that an
// issue carries discussion without the extra `bd comments` call the detail
// panel makes.
func TestParadeCommentBadge(t *testing.T) {
	issues := []data.Issue{testIssue("mg-1", data.StatusOpen)}
	issues[0].CommentCount = 3
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	out := ansi.Strip(p.View())

	if !strings.Contains(out, ui.SymComment+"3") {
		t.Errorf("expected comment badge %q in parade output:\n%s", ui.SymComment+"3", out)
	}
}

func TestParadeCommentBadgeAbsentWhenZero(t *testing.T) {
	// The overwhelmingly common case: no comments, no badge, no wasted width.
	issues := []data.Issue{testIssue("mg-1", data.StatusOpen)}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	out := ansi.Strip(p.View())

	if strings.Contains(out, ui.SymComment) {
		t.Errorf("expected no comment badge for CommentCount=0:\n%s", out)
	}
}

// The badge is width-budgeted like the due/deferred badges, so adding it must
// not push a row past the parade width.
func TestParadeCommentBadgeRespectsWidth(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120} {
		issues := []data.Issue{testIssue("mg-1", data.StatusOpen)}
		issues[0].Title = strings.Repeat("long title ", 12)
		issues[0].CommentCount = 42
		p := NewParade(issues, width, 20, data.DefaultBlockingTypes)
		for _, line := range strings.Split(ansi.Strip(p.View()), "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Errorf("width %d: row overflows to %d cells: %q", width, got, line)
			}
		}
	}
}
