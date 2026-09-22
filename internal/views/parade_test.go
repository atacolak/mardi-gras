package views

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

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
		row := ansi.Strip(p.renderIssue(item, false, false))
		byteIndex := strings.Index(row, item.Issue.ID)
		positions[item.Issue.ID] = ansi.StringWidth(row[:byteIndex])
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
			t.Logf("nested row: %s", ansi.Strip(p.renderIssue(item, false, false)))
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

// After t8 a closed epic lands collapsed in the Closed section, so the child
// is hidden on arrival. Expand first, then assert compaction: the muted
// re-render of the nested closed row must still print ".7", not the full ID.
func TestParadeRelativeDisplayIDClosed(t *testing.T) {
	epic := data.Issue{ID: "mard-nob", Title: "Epic", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeEpic}
	nested := data.Issue{
		ID: "mard-nob.7", Title: "Closed child", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{{IssueID: "mard-nob.7", DependsOnID: "mard-nob", Type: "parent-child"}},
	}
	p := NewParade([]data.Issue{epic, nested}, 100, 30, data.DefaultBlockingTypes)
	p.ToggleNode("mard-nob")

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

// After t6 there is no out-of-section parent for deferred either. Both the
// blocked and deferred children of a loaded epic sit in the same forest, so
// compacting to ".7" / ".8" is correct. Full ID at depth 0 was the deleted
// Waiting/Blocked and Deferred ghettos.
func TestParadeRelativeDisplayIDCompactsBlockedAndDeferredChildren(t *testing.T) {
	epic := data.Issue{ID: "mard-nob", Title: "Epic", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeEpic}
	blocked := data.Issue{
		ID: "mard-nob.7", Title: "Blocked child", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{
			{IssueID: "mard-nob.7", DependsOnID: "mard-nob", Type: "parent-child"},
			{IssueID: "mard-nob.7", DependsOnID: "mard-boss", Type: "blocks"},
		},
	}
	deferred := data.Issue{
		ID: "mard-nob.8", Title: "Deferred child", Status: data.StatusDeferred, Priority: data.PriorityMedium, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{
			{IssueID: "mard-nob.8", DependsOnID: "mard-nob", Type: "parent-child"},
		},
	}
	boss := data.Issue{ID: "mard-boss", Title: "Boss", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask}

	p := NewParade([]data.Issue{epic, blocked, deferred, boss}, 100, 30, data.DefaultBlockingTypes)

	var seenBlocked, seenDeferred bool
	for _, item := range p.Items {
		if item.Issue == nil {
			continue
		}
		switch item.Issue.ID {
		case "mard-nob.7":
			seenBlocked = true
			if item.Depth != 1 {
				t.Errorf("blocked child depth = %d, want 1 — its epic is in the same forest", item.Depth)
			}
			if item.Section != nil {
				t.Errorf("blocked child still belongs to section %q, want the open forest", item.Section.Title)
			}
			if got := ansi.Strip(item.RenderedID); got != ".7" {
				t.Errorf("blocked rendered ID = %q, want %q", got, ".7")
			}
			if item.State != data.StateWaitingBlocked {
				t.Errorf("blocked child state = %v, want StateWaitingBlocked", item.State)
			}
		case "mard-nob.8":
			seenDeferred = true
			if item.Depth != 1 {
				t.Errorf("deferred child depth = %d, want 1 — its epic is in the same forest", item.Depth)
			}
			if item.Section != nil {
				t.Errorf("deferred child still belongs to section %q, want the open forest", item.Section.Title)
			}
			if got := ansi.Strip(item.RenderedID); got != ".8" {
				t.Errorf("deferred rendered ID = %q, want %q", got, ".8")
			}
			if item.State != data.StateDeferred {
				t.Errorf("deferred child state = %v, want StateDeferred", item.State)
			}
		}
	}
	if !seenBlocked {
		t.Fatalf("blocked child missing from the parade: %v", paradeIssueIDs(p))
	}
	if !seenDeferred {
		t.Fatalf("deferred child missing from the parade: %v", paradeIssueIDs(p))
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

func paradeParentEdge(child, parent string) []data.Dependency {
	return []data.Dependency{{IssueID: child, DependsOnID: parent, Type: "parent-child"}}
}

func paradeIssueIDs(p Parade) []string {
	var ids []string
	for _, item := range p.Items {
		if item.Issue != nil {
			ids = append(ids, item.Issue.ID)
		}
	}
	return ids
}
func TestParadeBlockedChildRendersUnderParent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mard-mdr", Title: "Epic", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeEpic},
		{ID: "mard-mdr.3", Title: "Blocked child", Status: data.StatusBlocked, Priority: 0,
			Dependencies: paradeParentEdge("mard-mdr.3", "mard-mdr")},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)

	if got, want := paradeIssueIDs(p), []string{"mard-mdr", "mard-mdr.3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("row order = %v, want %v", got, want)
	}
	var child ParadeItem
	for _, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "mard-mdr.3" {
			child = item
		}
	}
	if child.Depth != 1 {
		t.Errorf("blocked child depth = %d, want 1", child.Depth)
	}
	if child.State != data.StateWaitingBlocked {
		t.Errorf("blocked child state = %v, want StateWaitingBlocked (red paint)", child.State)
	}
	if child.Section != nil {
		t.Errorf("blocked child still belongs to section %q, want the open forest", child.Section.Title)
	}
	if out := ansi.Strip(p.View()); strings.Contains(out, "Waiting/Blocked") {
		t.Fatalf("Waiting/Blocked section header still rendered:\n%s", out)
	}
}

func TestParadeDeferredChildRendersUnderParent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mard-mdr", Title: "Epic", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeEpic},
		{ID: "mard-mdr.4", Title: "Parked child", Status: data.StatusDeferred, Priority: 0,
			Dependencies: paradeParentEdge("mard-mdr.4", "mard-mdr")},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)

	var child ParadeItem
	for _, item := range p.Items {
		if item.IsHeader || item.IsFooter {
			t.Fatalf("open forest must have no section rows, got %+v", item.Section)
		}
		if item.Issue != nil && item.Issue.ID == "mard-mdr.4" {
			child = item
		}
	}
	if child.Issue == nil {
		t.Fatal("deferred child row missing")
	}
	if child.Depth != 1 || child.State != data.StateDeferred {
		t.Fatalf("deferred child depth/state = %d/%v, want 1/StateDeferred", child.Depth, child.State)
	}
	out := ansi.Strip(p.View())
	for _, header := range []string{"Waiting/Blocked", "Deferred ", "⏸ Deferred"} {
		if strings.Contains(out, header) {
			t.Fatalf("section header %q still rendered:\n%s", header, out)
		}
	}
}

func TestParadeTree(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic", Status: data.StatusReview, Priority: 1, IssueType: data.TypeEpic},
		{ID: "epic.1", Status: data.StatusInProgress, Priority: 2, Dependencies: paradeParentEdge("epic.1", "epic")},
		{ID: "epic.2", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("epic.2", "epic")},
		{ID: "epic.3", Status: data.StatusClosed, Priority: 3, Dependencies: paradeParentEdge("epic.3", "epic")},
		{ID: "epic.2.1", Status: data.StatusClosed, Priority: 1, Dependencies: paradeParentEdge("epic.2.1", "epic.2")},
		{ID: "blocked", Status: data.StatusBlocked, Priority: 0},
		{ID: "later", Status: data.StatusDeferred, Priority: 1},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)
	want := []string{"epic", "epic.1", "epic.2", "epic.2.1", "epic.3", "blocked", "later"}
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("issue order = %v, want %v", got, want)
	}
}

type paradeNodeToggler interface {
	ToggleNode(string)
}

func TestParadeCollapse(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic", Status: data.StatusReview, Priority: 1, IssueType: data.TypeEpic},
		{ID: "epic.2", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("epic.2", "epic")},
		{ID: "epic.2.1", Status: data.StatusClosed, Priority: 1, Dependencies: paradeParentEdge("epic.2.1", "epic.2")},
		{ID: "epic.1", Status: data.StatusInProgress, Priority: 2, Dependencies: paradeParentEdge("epic.1", "epic")},
		{ID: "attention", Status: data.StatusBlocked, Priority: 0},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)
	toggler, ok := any(&p).(paradeNodeToggler)
	if !ok {
		t.Fatal("Parade must expose ToggleNode")
	}
	beforeLeafToggle := paradeIssueIDs(p)
	toggler.ToggleNode("epic.2.1")
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, beforeLeafToggle) {
		t.Fatalf("leaf toggle changed rows = %v, want %v", got, beforeLeafToggle)
	}

	toggler.ToggleNode("epic.2")
	if got, want := paradeIssueIDs(p), []string{"epic", "epic.1", "epic.2", "attention"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after child collapse = %v, want %v", got, want)
	}
	for i, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "epic.2" {
			p.Cursor = i
			p.SelectedIssue = item.Issue
			break
		}
	}
	toggler.ToggleNode("epic")
	if got, want := paradeIssueIDs(p), []string{"epic", "attention"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after ancestor collapse = %v, want %v", got, want)
	}
	if p.SelectedIssue == nil || p.SelectedIssue.ID != "epic" {
		t.Fatalf("selection fallback = %v, want collapsed ancestor epic", p.SelectedIssue)
	}
}

func TestParadeCollapseAllEpics(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic-a", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "epic-a.1", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("epic-a.1", "epic-a")},
		{ID: "epic-b", Status: data.StatusInProgress, Priority: 1, IssueType: data.TypeEpic},
		{ID: "epic-b.1", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("epic-b.1", "epic-b")},
		{ID: "task", Status: data.StatusOpen, Priority: 2, IssueType: data.TypeTask},
		{ID: "task.1", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("task.1", "task")},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)
	for i, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "epic-a.1" {
			p.Cursor = i
			p.SelectedIssue = item.Issue
			break
		}
	}
	p.CollapseAllEpics()
	got := paradeIssueIDs(p)
	wantSet := map[string]bool{"epic-a": true, "epic-b": true, "task": true, "task.1": true}
	if len(got) != len(wantSet) {
		t.Fatalf("after CollapseAllEpics = %v, want %d rows %v", got, len(wantSet), wantSet)
	}
	for _, id := range got {
		if !wantSet[id] {
			t.Fatalf("after CollapseAllEpics unexpected row %q in %v", id, got)
		}
	}
	hidden := map[string]bool{"epic-a.1": true, "epic-b.1": true}
	for _, id := range got {
		if hidden[id] {
			t.Fatalf("epic child %q still visible after CollapseAllEpics: %v", id, got)
		}
	}
	if !p.Collapsed["epic-a"] || !p.Collapsed["epic-b"] {
		t.Fatalf("expected both epics collapsed, got %v", p.Collapsed)
	}
	if p.Collapsed["task"] {
		t.Fatal("CollapseAllEpics must not fold a non-epic parent")
	}
	if p.SelectedIssue == nil || p.SelectedIssue.ID != "epic-a" {
		t.Fatalf("caret = %v, want visible ancestor epic-a", p.SelectedIssue)
	}
}

func TestParadeToggleAllEpicsExpands(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic-a", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "epic-a.1", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("epic-a.1", "epic-a")},
		{ID: "task", Status: data.StatusOpen, Priority: 2, IssueType: data.TypeTask},
		{ID: "task.1", Status: data.StatusOpen, Priority: 0, Dependencies: paradeParentEdge("task.1", "task")},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)
	if !p.ToggleAllEpics() {
		t.Fatal("first toggle should collapse")
	}
	if !p.Collapsed["epic-a"] {
		t.Fatal("expected epic-a collapsed")
	}
	if p.Collapsed["task"] {
		t.Fatal("toggle must not fold a non-epic parent")
	}
	if p.ToggleAllEpics() {
		t.Fatal("second toggle should expand")
	}
	if p.Collapsed["epic-a"] {
		t.Fatal("expected epic-a expanded after second toggle")
	}
	got := paradeIssueIDs(p)
	want := map[string]bool{"epic-a": true, "epic-a.1": true, "task": true, "task.1": true}
	for _, id := range got {
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("after expand missing %v from %v", want, got)
	}
}

func TestParadeClosedEpicSection(t *testing.T) {
	issues := []data.Issue{
		{ID: "open", Title: "Open epic", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "open.1", Title: "Done child of open epic", Status: data.StatusClosed, Priority: 0,
			Dependencies: paradeParentEdge("open.1", "open")},
		{ID: "shut", Title: "Closed epic", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeEpic},
		{ID: "shut.1", Title: "Child of closed epic", Status: data.StatusClosed, Priority: 0,
			Dependencies: paradeParentEdge("shut.1", "shut")},
		{ID: "loose", Title: "Closed loose task", Status: data.StatusClosed, Priority: 2, IssueType: data.TypeTask},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)

	want := []string{"open", "open.1", "loose", "shut"}
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("default rows = %v, want %v (closed epic collapsed in its section; done child of an open epic stays in the forest)", got, want)
	}
	out := ansi.Strip(p.View())
	if !strings.Contains(out, "Closed"+ui.Superscript(1)) {
		t.Fatalf("expected a Closed section header counting one closed epic:\n%s", out)
	}
	if strings.Contains(out, "Child of closed epic") {
		t.Fatalf("closed epic must default to collapsed:\n%s", out)
	}

	p.ToggleNode("shut")
	want = []string{"open", "open.1", "loose", "shut", "shut.1"}
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("after expanding the closed epic = %v, want %v", got, want)
	}

	bare := NewParade([]data.Issue{{ID: "solo", Title: "Solo", Status: data.StatusOpen}}, 100, 20, data.DefaultBlockingTypes)
	for _, item := range bare.Items {
		if item.IsHeader || item.IsFooter {
			t.Fatalf("no closed epic on the board must mean no section rows, got %+v", item.Section)
		}
	}
}

func TestParadeClosedHeaderKeepsTopRightCorner(t *testing.T) {
	issues := []data.Issue{
		{ID: "shut", Title: "Closed epic", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeEpic},
	}
	for _, width := range []int{40, 60, 80, 100, 140} {
		p := NewParade(issues, width, 12, data.DefaultBlockingTypes)
		found := false
		for _, line := range strings.Split(p.View(), "\n") {
			plain := ansi.Strip(line)
			if !strings.Contains(plain, "Closed") || !strings.Contains(plain, ui.BoxTopLeft) {
				continue
			}
			found = true
			if got := lipgloss.Width(line); got != width {
				t.Fatalf("width %d Closed header width = %d, want %d: %q", width, got, width, plain)
			}
			if !strings.HasSuffix(plain, ui.BoxTopRight) {
				t.Fatalf("width %d Closed header = %q, want it to end with %s", width, plain, ui.BoxTopRight)
			}
			if !strings.HasPrefix(strings.TrimLeft(plain, " "), ui.BoxTopLeft) {
				t.Fatalf("width %d Closed header = %q, want it to start with %s", width, plain, ui.BoxTopLeft)
			}
		}
		if !found {
			t.Fatalf("width %d: Closed header not rendered", width)
		}
	}
}

func TestParadeCollapsedClosedHeaderIsStraight(t *testing.T) {
	issues := []data.Issue{
		{ID: "shut", Title: "Closed epic", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeEpic},
	}
	for _, width := range []int{40, 60, 80, 100, 140} {
		p := NewParade(issues, width, 12, data.DefaultBlockingTypes)
		p.ToggleClosedSection()
		found := false
		for _, line := range strings.Split(p.View(), "\n") {
			plain := ansi.Strip(line)
			if !strings.Contains(plain, "Closed") {
				continue
			}
			found = true
			if got := lipgloss.Width(line); got != width {
				t.Fatalf("width %d collapsed Closed header width = %d, want %d: %q", width, got, width, plain)
			}
			if strings.Contains(plain, ui.BoxTopLeft) || strings.Contains(plain, ui.BoxTopRight) {
				t.Fatalf("width %d collapsed Closed header = %q, want no ╭/╮", width, plain)
			}
			if !strings.HasPrefix(strings.TrimLeft(plain, " "), ui.BoxHorizontal) {
				t.Fatalf("width %d collapsed Closed header = %q, want it to start with %s", width, plain, ui.BoxHorizontal)
			}
			if !strings.HasSuffix(plain, ui.BoxHorizontal) {
				t.Fatalf("width %d collapsed Closed header = %q, want it to end with %s", width, plain, ui.BoxHorizontal)
			}
		}
		if !found {
			t.Fatalf("width %d: collapsed Closed header not rendered", width)
		}
	}
}

func TestParadeToggleClosedSectionHidesEpics(t *testing.T) {
	issues := []data.Issue{
		{ID: "open", Title: "Open epic", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "shut", Title: "Closed epic", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeEpic},
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	if !containsParadeID(p, "shut") {
		t.Fatal("precondition: closed epic must be visible")
	}
	headerRow := closedHeaderRow(t, &p)
	if !p.ClosedHeaderHit(headerRow, 0) || !p.ClosedHeaderHit(headerRow, 2) {
		t.Fatal("╭─ cells must be a Closed header hit")
	}
	if p.ClosedHeaderHit(headerRow, 5) {
		t.Fatal("Closed title text must not be a hit")
	}
	p.ToggleClosedSection()
	if !p.ClosedCollapsed {
		t.Fatal("ToggleClosedSection must collapse")
	}
	if containsParadeID(p, "shut") {
		t.Fatal("collapsed Closed list must hide the epics")
	}
	if !strings.Contains(ansi.Strip(p.View()), "Closed") {
		t.Fatal("collapsed Closed list must keep the header")
	}
	p.ToggleClosedSection()
	if p.ClosedCollapsed || !containsParadeID(p, "shut") {
		t.Fatal("second toggle must show the closed epics again")
	}
}

func TestParadeClosedTitleDarkerThanDeferred(t *testing.T) {
	if ui.ClosedTitle.GetForeground() != ui.Dim {
		t.Fatalf("closed titles must be Dim (darker grey), got %v", ui.ClosedTitle.GetForeground())
	}
	if ui.DeferredStyle.GetForeground() != ui.Muted {
		t.Fatalf("deferred titles must be Muted (lighter grey), got %v", ui.DeferredStyle.GetForeground())
	}
	if ui.DoneTitle.GetForeground() != ui.TitleDone {
		t.Fatalf("done titles must be TitleDone (between deferred and closed), got %v", ui.DoneTitle.GetForeground())
	}
}

func TestParadeClosedSectionUsesDeferredColor(t *testing.T) {
	issues := []data.Issue{
		{ID: "shut", Title: "Closed epic", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeEpic},
	}
	p := NewParade(issues, 80, 12, data.DefaultBlockingTypes)
	var header *paradeSection
	for _, item := range p.Items {
		if item.IsHeader {
			header = item.Section
			break
		}
	}
	if header == nil {
		t.Fatal("Closed header missing")
	}
	if header.Color != ui.ExecColor(int(data.StateDeferred)) {
		t.Fatalf("Closed border color = %v, want deferred", header.Color)
	}
}

func containsParadeID(p Parade, id string) bool {
	for _, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == id {
			return true
		}
	}
	return false
}

func closedHeaderRow(t *testing.T, p *Parade) int {
	t.Helper()
	for r := 0; r < p.Height; r++ {
		idx := p.ScrollOffset + r
		if idx >= 0 && idx < len(p.Items) && p.Items[idx].IsHeader {
			return r
		}
	}
	t.Fatal("Closed header not visible")
	return -1
}

func TestParadeSemanticColor(t *testing.T) {
	old := testIssue("old-ready", data.StatusOpen)
	old.CreatedAt = time.Now().Add(-90 * 24 * time.Hour)
	fresh := testIssue("fresh-ready", data.StatusOpen)
	p := NewParade([]data.Issue{old, fresh}, 100, 20, data.DefaultBlockingTypes)
	wantOld := lipgloss.NewStyle().Foreground(ui.ExecColor(int(data.StateReady))).Render(old.ID)
	wantFresh := lipgloss.NewStyle().Foreground(ui.ExecColor(int(data.StateReady))).Render(fresh.ID)
	for _, item := range p.Items {
		if item.Issue == nil {
			continue
		}
		want := wantOld
		if item.Issue.ID == fresh.ID {
			want = wantFresh
		}
		if item.RenderedID != want {
			t.Errorf("%s RenderedID = %q, want semantic style %q", item.Issue.ID, item.RenderedID, want)
		}
	}
}

func TestParadePinnedBadge(t *testing.T) {
	issue := testIssue("pinned", data.StatusOpen)
	issue.Pinned = true
	p := NewParade([]data.Issue{issue}, 100, 20, data.DefaultBlockingTypes)
	out := ansi.Strip(p.View())
	if !strings.Contains(out, "PIN") {
		t.Fatalf("pinned row should contain PIN badge:\n%s", out)
	}
	state, ok := data.DeriveState(&issue, data.BuildIssueMap([]data.Issue{issue}), data.DefaultBlockingTypes)
	if !ok || state != data.StateReady {
		t.Fatalf("pinned issue state = %v/%v, want Ready/true", state, ok)
	}
}

func TestParadeIssueAtViewportRowUsesScrollOffset(t *testing.T) {
	issues := []data.Issue{
		{ID: "ready", Status: data.StatusOpen, Priority: 0},
		{ID: "later", Status: data.StatusDeferred, Priority: 0},
	}
	p := NewParade(issues, 100, 2, data.DefaultBlockingTypes)
	p.ScrollOffset = 1
	atRow, ok := any(&p).(interface{ IssueAtViewportRow(int) *data.Issue })
	if !ok {
		t.Fatal("Parade must expose IssueAtViewportRow")
	}
	// After t6 there is no Deferred header. Both rows are forest issues.
	// Attention rank puts ready (rank 3) before later (rank 4). ScrollOffset=1
	// makes viewport row 0 the second forest issue (later), proving the
	// resolver honors the offset rather than always returning the first row.
	if got := atRow.IssueAtViewportRow(0); got == nil || got.ID != "later" {
		t.Fatalf("scrolled viewport row 0 = %v, want later", got)
	}
	if got := atRow.IssueAtViewportRow(1); got != nil {
		t.Fatalf("viewport row 1 past the forest = %v, want nil", got)
	}
	if got := atRow.IssueAtViewportRow(-1); got != nil {
		t.Fatalf("negative viewport row = %v, want nil", got)
	}
	if got := atRow.IssueAtViewportRow(2); got != nil {
		t.Fatalf("padding viewport row = %v, want nil", got)
	}
}

func TestParadeRowHasNoDisclosureGlyph(t *testing.T) {
	parent := data.Issue{ID: "parent", Title: "Parent", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic}
	child := data.Issue{
		ID: "parent.1", Title: "Child", Status: data.StatusOpen, Priority: 1,
		Dependencies: paradeParentEdge("parent.1", "parent"),
	}
	leaf := data.Issue{ID: "leaf", Title: "Leaf", Status: data.StatusOpen, Priority: 2}
	p := NewParade([]data.Issue{parent, child, leaf}, 100, 20, data.DefaultBlockingTypes)

	expanded := ansi.Strip(p.View())
	for _, glyph := range []string{"▼", "▶"} {
		if strings.Contains(expanded, glyph) {
			t.Fatalf("expanded parade row contains disclosure glyph %q:\n%s", glyph, expanded)
		}
	}

	p.ToggleNode("parent")
	collapsed := ansi.Strip(p.View())
	for _, glyph := range []string{"▼", "▶"} {
		if strings.Contains(collapsed, glyph) {
			t.Fatalf("collapsed parade row contains disclosure glyph %q:\n%s", glyph, collapsed)
		}
	}
	if strings.Contains(collapsed, "Child") {
		t.Fatalf("collapsed parent should still hide its child:\n%s", collapsed)
	}
}
func TestParadeRowHasNoNextBlockerHint(t *testing.T) {
	blocker := data.Issue{ID: "mard-4vh", Title: "Ghost blocker", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeTask}
	blocked := data.Issue{
		ID: "mard-t29", Title: "Blocked work", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeTask,
		Dependencies: []data.Dependency{{IssueID: "mard-t29", DependsOnID: "mard-4vh", Type: "blocks"}},
	}
	issues := []data.Issue{blocker, blocked}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)

	eval := blocked.EvaluateDependencies(data.BuildIssueMap(issues), data.DefaultBlockingTypes)
	if !eval.IsBlocked || eval.NextBlockerID == "" {
		t.Fatalf("precondition: expected a blocked issue with a next blocker, got %+v", eval)
	}

	out := ansi.Strip(p.View())
	if strings.Contains(out, "next") || strings.Contains(out, "Ghost blocker…") {
		t.Fatalf("parade row still carries the next-blocker hint:\n%s", out)
	}
}

func paradeSortFixture() []data.Issue {
	return []data.Issue{
		{ID: "epic", Title: "Epic", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "epic.attn", Title: "Attention", Status: data.StatusReview, Priority: 0,
			Dependencies: paradeParentEdge("epic.attn", "epic")},
		{ID: "epic.blocked", Title: "Blocked", Status: data.StatusBlocked, Priority: 0,
			Dependencies: paradeParentEdge("epic.blocked", "epic")},
		{ID: "epic.defer", Title: "Deferred", Status: data.StatusDeferred, Priority: 0,
			Dependencies: paradeParentEdge("epic.defer", "epic")},
		{ID: "epic.done", Title: "Done", Status: data.StatusClosed, Priority: 0,
			Dependencies: paradeParentEdge("epic.done", "epic")},
		{ID: "epic.ready", Title: "Ready", Status: data.StatusOpen, Priority: 0,
			Dependencies: paradeParentEdge("epic.ready", "epic")},
		{ID: "epic.work", Title: "Working", Status: data.StatusInProgress, Priority: 0,
			Dependencies: paradeParentEdge("epic.work", "epic")},
	}
}

func TestParadeSortModes(t *testing.T) {
	p := NewParade(paradeSortFixture(), 100, 30, data.DefaultBlockingTypes)
	if p.EpicSortMode != SortAttention || p.BeadSortMode != SortAttention {
		t.Fatalf("default sort = epic %v bead %v, want attention", p.EpicSortMode, p.BeadSortMode)
	}
	want := []string{"epic", "epic.attn", "epic.blocked", "epic.work", "epic.ready", "epic.defer", "epic.done"}
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("attention order = %v, want %v", got, want)
	}

	p.BeadSortMode = SortPriority
	p.RebuildItems()
	want = []string{"epic", "epic.attn", "epic.blocked", "epic.defer", "epic.done", "epic.ready", "epic.work"}
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("priority order = %v, want %v (equal P falls back to ID)", got, want)
	}

	p.BeadSortMode = SortChronological
	p.RebuildItems()
	want = []string{"epic", "epic.attn", "epic.blocked", "epic.defer", "epic.done", "epic.ready", "epic.work"}
	if got := paradeIssueIDs(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("chronological order = %v, want %v (zero created_at → natural ID)", got, want)
	}
	if SortAttention.Next() != SortPriority || SortPriority.Next() != SortChronological || SortChronological.Next() != SortAttention {
		t.Fatal("S must cycle attention → priority → chronological → attention")
	}
	if SortChronological.Label() != "chronological" {
		t.Fatalf("label = %q, want chronological", SortChronological.Label())
	}
}

func TestParadeSortNeverConsultsRecency(t *testing.T) {
	stale := data.Issue{ID: "b-stale", Title: "Stale", Status: data.StatusOpen, Priority: 2,
		CreatedAt: time.Now().Add(-90 * 24 * time.Hour), UpdatedAt: time.Now().Add(-90 * 24 * time.Hour)}
	fresh := data.Issue{ID: "a-fresh", Title: "Fresh", Status: data.StatusOpen, Priority: 2,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	for _, mode := range []SortMode{SortAttention, SortPriority} {
		p := NewParade([]data.Issue{stale, fresh}, 100, 20, data.DefaultBlockingTypes)
		p.BeadSortMode = mode
		p.RebuildItems()
		if got, want := paradeIssueIDs(p), []string{"a-fresh", "b-stale"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("%s order = %v, want %v (ID tiebreak, never recency)", mode.Label(), got, want)
		}
	}
}

func TestParadeSortChronologicalUsesCreatedAt(t *testing.T) {
	older := data.Issue{ID: "z-old", Title: "Old", Status: data.StatusOpen, Priority: 2,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	newer := data.Issue{ID: "a-new", Title: "New", Status: data.StatusOpen, Priority: 2,
		CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}
	p := NewParade([]data.Issue{newer, older}, 100, 20, data.DefaultBlockingTypes)
	p.BeadSortMode = SortChronological
	p.RebuildItems()
	if got, want := paradeIssueIDs(p), []string{"z-old", "a-new"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("chronological = %v, want %v (created_at, not ID)", got, want)
	}
}

func TestParadeSortChronologicalNaturalIDOnChildren(t *testing.T) {
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	issues := []data.Issue{
		{ID: "omp-ayn", Title: "Epic", Status: data.StatusInProgress, Priority: 1, IssueType: data.TypeEpic, CreatedAt: now},
		{ID: "omp-ayn.102", Title: "later", Status: data.StatusClosed, Priority: 1, CreatedAt: now,
			Dependencies: paradeParentEdge("omp-ayn.102", "omp-ayn")},
		{ID: "omp-ayn.10", Title: "tenth", Status: data.StatusClosed, Priority: 1, CreatedAt: now,
			Dependencies: paradeParentEdge("omp-ayn.10", "omp-ayn")},
		{ID: "omp-ayn.2", Title: "second", Status: data.StatusOpen, Priority: 1, CreatedAt: now,
			Dependencies: paradeParentEdge("omp-ayn.2", "omp-ayn")},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	p.BeadSortMode = SortChronological
	p.RebuildItems()
	if got, want := paradeIssueIDs(p), []string{"omp-ayn", "omp-ayn.2", "omp-ayn.10", "omp-ayn.102"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("child chronological = %v, want %v (natural ID under the epic)", got, want)
	}
}

func TestParadeEpicAndBeadSortIndependent(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	issues := []data.Issue{
		{ID: "z-epic", Title: "Older epic", Status: data.StatusOpen, Priority: 2, IssueType: data.TypeEpic, CreatedAt: t0},
		{ID: "a-epic", Title: "Newer epic", Status: data.StatusOpen, Priority: 2, IssueType: data.TypeEpic, CreatedAt: t1},
		{ID: "z-epic.work", Title: "Working child", Status: data.StatusInProgress, Priority: 2, CreatedAt: t1,
			Dependencies: paradeParentEdge("z-epic.work", "z-epic")},
		{ID: "z-epic.ready", Title: "Ready child", Status: data.StatusOpen, Priority: 2, CreatedAt: t0,
			Dependencies: paradeParentEdge("z-epic.ready", "z-epic")},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	p.EpicSortMode = SortChronological
	p.BeadSortMode = SortAttention
	p.RebuildItems()
	if got, want := paradeIssueIDs(p), []string{"z-epic", "z-epic.work", "z-epic.ready", "a-epic"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("independent sorts = %v, want %v (epics by created_at, beads by attention)", got, want)
	}
}

func TestCompareBeadIDNaturalOrder(t *testing.T) {
	if compareBeadID("omp-x.2", "omp-x.10") >= 0 {
		t.Fatal("omp-x.2 must precede omp-x.10")
	}
	if compareBeadID("omp-x.10", "omp-x.102") >= 0 {
		t.Fatal("omp-x.10 must precede omp-x.102")
	}
	if compareBeadID("a", "a") != 0 {
		t.Fatal("equal IDs")
	}
}

func TestParadeSettledEpicTitleDimsAndUndims(t *testing.T) {
	epic := data.Issue{ID: "e", Title: "Campaign", Status: data.StatusInProgress, Priority: 1, IssueType: data.TypeEpic}
	child := data.Issue{ID: "e.1", Title: "Kid", Status: data.StatusClosed, Priority: 1,
		Dependencies: paradeParentEdge("e.1", "e")}
	p := NewParade([]data.Issue{epic, child}, 100, 20, data.DefaultBlockingTypes)
	if got := p.titleStyle(&epic); got.GetForeground() != ui.TitleSettled {
		t.Fatalf("n/n epic title = %v, want TitleSettled", got.GetForeground())
	}
	if !gotBold(p.titleStyle(&epic)) {
		t.Fatal("n/n epic must stay bold")
	}

	child.Status = data.StatusOpen
	p = NewParade([]data.Issue{epic, child}, 100, 20, data.DefaultBlockingTypes)
	if got := p.titleStyle(&epic); got.GetForeground() != ui.White {
		t.Fatalf("reopened child must restore live epic ink, got %v", got.GetForeground())
	}
}

func TestParadeBeadTitleQuieterThanEpic(t *testing.T) {
	epic := data.Issue{ID: "e", Title: "Epic", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeEpic}
	bead := data.Issue{ID: "t", Title: "Task", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask}
	p := NewParade([]data.Issue{epic, bead}, 100, 20, data.DefaultBlockingTypes)
	es, bs := p.titleStyle(&epic), p.titleStyle(&bead)
	if !gotBold(es) {
		t.Fatal("epic title must be bold")
	}
	if gotBold(bs) {
		t.Fatal("bead title must not be bold")
	}
	if bs.GetForeground() != ui.TitleBead {
		t.Fatalf("bead title = %v, want TitleBead", bs.GetForeground())
	}
}

func gotBold(s lipgloss.Style) bool {
	return s.GetBold()
}

func TestParadeFamilyGlyphHighlight(t *testing.T) {
	issues := []data.Issue{
		{ID: "a", Title: "Epic A", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "a.1", Title: "A one", Status: data.StatusOpen, Priority: 0,
			Dependencies: paradeParentEdge("a.1", "a")},
		{ID: "a.2", Title: "A two", Status: data.StatusInProgress, Priority: 1,
			Dependencies: paradeParentEdge("a.2", "a")},
		{ID: "a.1.1", Title: "A one one", Status: data.StatusInProgress, Priority: 0,
			Dependencies: paradeParentEdge("a.1.1", "a.1")},
		{ID: "b", Title: "Epic B", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeEpic},
		{ID: "b.1", Title: "B one", Status: data.StatusInProgress, Priority: 0,
			Dependencies: paradeParentEdge("b.1", "b")},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)
	for i, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "a.1" {
			p.Cursor, p.SelectedIssue = i, item.Issue
		}
	}

	// Cursor on a child still lights the selected epic and every bead under it,
	// not the old same-depth sibling window and not the neighboring epic.
	want := map[string]bool{"a": true, "a.1": true, "a.2": true, "a.1.1": true}
	if got := p.familyGlyphIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("family glyph set = %v, want %v (selected tree, not same-depth siblings)", got, want)
	}

	out := p.View()
	if got := strings.Count(out, ui.ExecIndicator(int(data.StateWorking))); got != 2 {
		t.Errorf("full-color Working glyphs = %d, want 2 (a.2 and a.1.1; b.1 recedes)", got)
	}
	if got := strings.Count(out, ui.ExecIndicatorDim(int(data.StateWorking))); got != 1 {
		t.Errorf("receded Working glyphs = %d, want 1 (b.1)", got)
	}

	for i, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "a" {
			p.Cursor, p.SelectedIssue = i, item.Issue
		}
	}
	if got := p.familyGlyphIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("family glyph set with cursor on epic = %v, want %v", got, want)
	}
}

func TestParadeHasNoPositionalFade(t *testing.T) {
	issues := make([]data.Issue, 20)
	for i := range issues {
		issues[i] = data.Issue{
			ID: fmt.Sprintf("fade-%02d", i), Title: "Row", Status: data.StatusOpen, Priority: 2,
		}
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	if out := p.View(); strings.Contains(out, "\x1b[2m") {
		t.Fatal("parade still renders faint rows; the ±6 positional fade must be gone")
	}
}

func TestParadeRowRightAlignsPriorityBadge(t *testing.T) {
	iss := data.Issue{
		ID: "align-1", Title: strings.Repeat("long title ", 12), Status: data.StatusOpen,
		Priority: 2, IssueType: data.TypeTask, CommentCount: 3, Pinned: true,
	}
	for _, width := range []int{60, 80, 100, 140} {
		p := NewParade([]data.Issue{iss}, width, 10, data.DefaultBlockingTypes)
		var item ParadeItem
		for _, it := range p.Items {
			if it.Issue != nil {
				item = it
			}
		}
		row := ansi.Strip(p.renderIssue(item, false, false))
		if !strings.HasSuffix(row, "P2") {
			t.Fatalf("width %d row = %q, want it to end with the P badge", width, row)
		}
		if got := lipgloss.Width(row); got != width {
			t.Fatalf("width %d row width = %d, want %d so the badge sits in the last cell", width, got, width)
		}
	}
}

func TestParadeNestedPriorityBadgeInsets(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic", Title: "Epic", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeEpic},
		{ID: "epic.1", Title: "Child", Status: data.StatusOpen, Priority: 2,
			Dependencies: paradeParentEdge("epic.1", "epic")},
		{ID: "epic.1.1", Title: "Grand", Status: data.StatusOpen, Priority: 3,
			Dependencies: paradeParentEdge("epic.1.1", "epic.1")},
	}
	for _, width := range []int{60, 80, 100, 140} {
		p := NewParade(issues, width, 10, data.DefaultBlockingTypes)
		byID := map[string]ParadeItem{}
		for _, it := range p.Items {
			if it.Issue != nil {
				byID[it.Issue.ID] = it
			}
		}
		epic := ansi.Strip(p.renderIssue(byID["epic"], false, false))
		child := ansi.Strip(p.renderIssue(byID["epic.1"], false, false))
		grand := ansi.Strip(p.renderIssue(byID["epic.1.1"], false, false))
		if !strings.HasSuffix(epic, "P1") {
			t.Fatalf("width %d epic = %q, want flush P1", width, epic)
		}
		if !strings.HasSuffix(child, "P2  ") {
			t.Fatalf("width %d child = %q, want P2 inset by two spaces", width, child)
		}
		if !strings.HasSuffix(grand, "P3    ") {
			t.Fatalf("width %d grand = %q, want P3 inset by four spaces", width, grand)
		}
		for _, row := range []string{epic, child, grand} {
			if got := lipgloss.Width(row); got != width {
				t.Fatalf("width %d row width = %d, want %d: %q", width, got, width, row)
			}
		}
	}
}

func TestParadePinCursorToSelectedKeepsHiddenChild(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic", Title: "Epic", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "epic.1", Title: "Child", Status: data.StatusOpen, Priority: 0,
			Dependencies: paradeParentEdge("epic.1", "epic")},
		{ID: "other", Title: "Other", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	for i, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "epic.1" {
			p.Cursor, p.SelectedIssue = i, item.Issue
		}
	}
	stale := p.Cursor
	p.ToggleNode("epic")
	p.SelectedIssue = p.IssueByID("epic.1")
	p.Cursor = stale
	p.PinCursorToSelected()
	if p.SelectedIssue == nil || p.SelectedIssue.ID != "epic.1" {
		t.Fatalf("selected = %v, want hidden child epic.1", p.SelectedIssue)
	}
	if p.Items[p.Cursor].Issue == nil || p.Items[p.Cursor].Issue.ID != "epic" {
		t.Fatalf("caret after collapse = %v, want ancestor epic, not the neighbor at the old index", p.Items[p.Cursor].Issue)
	}
	p.ToggleNode("epic")
	p.SelectedIssue = p.IssueByID("epic.1")
	p.PinCursorToSelected()
	if p.Items[p.Cursor].Issue == nil || p.Items[p.Cursor].Issue.ID != "epic.1" {
		t.Fatalf("caret after expand = %v, want epic.1 again", p.Items[p.Cursor].Issue)
	}
}

func TestParadeRowIsSelectedIgnoresStaleCursor(t *testing.T) {
	issues := []data.Issue{
		{ID: "a", Title: "A", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "b", Title: "B", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask},
	}
	p := NewParade(issues, 100, 10, data.DefaultBlockingTypes)
	var a, b ParadeItem
	for _, item := range p.Items {
		if item.Issue == nil {
			continue
		}
		if item.Issue.ID == "a" {
			a = item
			p.SelectedIssue = item.Issue
		}
		if item.Issue.ID == "b" {
			b = item
			p.Cursor = 1
		}
	}
	if !p.rowIsSelected(a, 0) {
		t.Fatal("visible selected issue must keep the caret")
	}
	if p.rowIsSelected(b, p.Cursor) {
		t.Fatal("stale cursor index must not mark a different bead selected")
	}
}

func TestParadeGutterHit(t *testing.T) {
	issues := []data.Issue{
		{ID: "epic", Title: "Epic", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "epic.1", Title: "Mid", Status: data.StatusOpen, Priority: 0,
			Dependencies: paradeParentEdge("epic.1", "epic")},
		{ID: "epic.1.1", Title: "Leaf", Status: data.StatusOpen, Priority: 0,
			Dependencies: paradeParentEdge("epic.1.1", "epic.1")},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	row := func(id string) int {
		t.Helper()
		for r := 0; r < p.Height; r++ {
			if iss := p.IssueAtViewportRow(r); iss != nil && iss.ID == id {
				return r
			}
		}
		t.Fatalf("row for %s not visible", id)
		return -1
	}

	// Depth 0 parent: the two cursor-prefix cells are the gutter, column 2 is
	// the status glyph and already belongs to selection.
	for _, x := range []int{0, 1} {
		if id, ok := p.GutterHit(row("epic"), x); !ok || id != "epic" {
			t.Errorf("GutterHit(epic row, %d) = %q/%v, want epic/true", x, id, ok)
		}
	}
	for _, x := range []int{2, 3, 20} {
		if id, ok := p.GutterHit(row("epic"), x); ok {
			t.Errorf("GutterHit(epic row, %d) = %q/true, want no hit — that is the glyph/title", x, id)
		}
	}

	// Depth 1 parent: two prefix cells plus two indent cells.
	for _, x := range []int{0, 1, 2, 3} {
		if id, ok := p.GutterHit(row("epic.1"), x); !ok || id != "epic.1" {
			t.Errorf("GutterHit(epic.1 row, %d) = %q/%v, want epic.1/true", x, id, ok)
		}
	}
	if id, ok := p.GutterHit(row("epic.1"), 4); ok {
		t.Errorf("GutterHit(epic.1 row, 4) = %q/true, want no hit — that is the glyph", id)
	}

	// A leaf has nothing to collapse, at any column.
	for _, x := range []int{0, 1, 2, 3, 4, 5} {
		if id, ok := p.GutterHit(row("epic.1.1"), x); ok {
			t.Errorf("GutterHit(leaf row, %d) = %q/true, want no hit", x, id)
		}
	}
	if _, ok := p.GutterHit(-1, 0); ok {
		t.Error("GutterHit on an out-of-range row must not hit")
	}

	if !p.HasChildrenAtViewportRow(row("epic")) {
		t.Error("HasChildrenAtViewportRow(epic) = false, want true")
	}
	if p.HasChildrenAtViewportRow(row("epic.1.1")) {
		t.Error("HasChildrenAtViewportRow(leaf) = true, want false")
	}
}

func TestParadeEpicGlyphShowsChildSuperscript(t *testing.T) {
	issues := []data.Issue{
		{ID: "a", Title: "Epic A", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "a.1", Title: "A one", Status: data.StatusOpen, Priority: 0,
			Dependencies: paradeParentEdge("a.1", "a")},
		{ID: "a.2", Title: "A two", Status: data.StatusInProgress, Priority: 1,
			Dependencies: paradeParentEdge("a.2", "a")},
		{ID: "a.1.1", Title: "A one one", Status: data.StatusInProgress, Priority: 0,
			Dependencies: paradeParentEdge("a.1.1", "a.1")},
		{ID: "leaf", Title: "Leaf", Status: data.StatusOpen, Priority: 2, IssueType: data.TypeTask},
	}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)
	var epic, leaf *ParadeItem
	for i := range p.Items {
		if p.Items[i].Issue == nil {
			continue
		}
		switch p.Items[i].Issue.ID {
		case "a":
			epic = &p.Items[i]
		case "leaf":
			leaf = &p.Items[i]
		}
	}
	if epic == nil || epic.ChildCount != 2 {
		t.Fatalf("epic ChildCount = %v, want 2 (direct children, same as Progress Total)", epic)
	}
	if leaf == nil || leaf.ChildCount != 0 {
		t.Fatalf("non-epic ChildCount = %v, want 0", leaf)
	}
	out := ansi.Strip(p.View())
	want := ui.ExecSymbol(int(data.StateReady)) + ui.Superscript(2)
	if !strings.Contains(out, want) {
		t.Fatalf("parade missing epic superscript %q:\n%s", want, out)
	}
	if strings.Contains(out, ui.ExecSymbol(int(data.StateReady))+ui.Superscript(3)) {
		t.Fatal("epic superscript counted descendants, not direct children")
	}
}

func TestParadeNonEpicHasNoChildSuperscript(t *testing.T) {
	issues := []data.Issue{
		{ID: "task", Title: "Just a task", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeTask},
		{ID: "task.1", Title: "Should not count on a task", Status: data.StatusOpen, Priority: 1,
			Dependencies: paradeParentEdge("task.1", "task")},
	}
	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	for _, item := range p.Items {
		if item.Issue != nil && item.Issue.ID == "task" && item.ChildCount != 0 {
			t.Fatalf("task ChildCount = %d, want 0 (superscript is epic-only)", item.ChildCount)
		}
	}
	out := ansi.Strip(p.View())
	if strings.Contains(out, ui.Superscript(1)) {
		t.Fatalf("non-epic row rendered a child superscript:\n%s", out)
	}
}
