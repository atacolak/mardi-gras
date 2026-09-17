package views

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

func testIssue(id string, status data.Status) data.Issue {
	now := time.Now()
	return data.Issue{
		ID:        id,
		Title:     id,
		Status:    status,
		Priority:  data.PriorityMedium,
		IssueType: data.TypeTask,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func blockedIssue(id string, status data.Status) data.Issue {
	iss := testIssue(id, status)
	iss.Dependencies = []data.Dependency{
		{IssueID: id, DependsOnID: "missing-dep", Type: "blocks"},
	}
	return iss
}

// semanticParadeIssues returns one issue per derived semantic state — the
// six-way fixture the section renderer, the header, and the tmux widget are
// each checked against.
//
// The epic is the legacy compatibility case: in_progress with its only child
// closed remains Operator Review until the row is stored as review. The
// deferred issue is open with a future defer_until, which is the only way an
// open issue is not Ready.
func semanticParadeIssues() []data.Issue {
	until := time.Now().Add(48 * time.Hour)

	epic := testIssue("review-1", data.StatusInProgress)
	epic.IssueType = data.TypeEpic
	child := testIssue("review-1.1", data.StatusClosed)
	child.Dependencies = []data.Dependency{{IssueID: child.ID, DependsOnID: epic.ID, Type: "parent-child"}}

	deferred := testIssue("later-1", data.StatusOpen)
	deferred.DeferUntil = &until

	return []data.Issue{
		testIssue("work-1", data.StatusInProgress), // Working
		epic,                                    // Operator Review
		child,                                   // Done
		testIssue("ready-1", data.StatusOpen),   // Ready
		deferred,                                // Deferred
		testIssue("stop-1", data.StatusBlocked), // Waiting/Blocked
		testIssue("done-1", data.StatusClosed),  // Done
	}
}

func TestStatusSymbol(t *testing.T) {
	issues := append(semanticParadeIssues(),
		blockedIssue("blocked-ip-1", data.StatusInProgress),
		blockedIssue("blocked-open-1", data.StatusOpen),
	)
	issueMap := data.BuildIssueMap(issues)

	tests := []struct {
		id     string
		expect string
	}{
		{id: "work-1", expect: ui.SymExecWorking},
		{id: "review-1", expect: ui.SymExecOperatorReview},
		{id: "ready-1", expect: ui.SymExecReady},
		{id: "later-1", expect: ui.SymExecDeferred},
		// Raw blocked and blocked-by-dependency are the same state: an
		// unresolved blocker is what Waiting/Blocked means either way.
		{id: "stop-1", expect: ui.SymExecWaiting},
		{id: "blocked-ip-1", expect: ui.SymExecWaiting},
		{id: "blocked-open-1", expect: ui.SymExecWaiting},
		{id: "done-1", expect: ui.SymExecDone},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			state, ok := data.DeriveState(issueMap[tc.id], issueMap, data.DefaultBlockingTypes)
			if !ok {
				t.Fatalf("%s should be decidable", tc.id)
			}
			if got := statusSymbol(state); got != tc.expect {
				t.Fatalf("statusSymbol(%s) = %q, want %q", tc.id, got, tc.expect)
			}
		})
	}
}

func TestStatusColor(t *testing.T) {
	issues := append(semanticParadeIssues(),
		blockedIssue("blocked-ip-1", data.StatusInProgress),
		blockedIssue("blocked-open-1", data.StatusOpen),
	)
	issueMap := data.BuildIssueMap(issues)

	tests := []struct {
		id     string
		expect color.Color
	}{
		{id: "work-1", expect: ui.ExecWorking},
		{id: "review-1", expect: ui.ExecOperatorReview},
		{id: "ready-1", expect: ui.ExecReady},
		{id: "later-1", expect: ui.ExecDeferred},
		{id: "stop-1", expect: ui.ExecWaiting},
		{id: "blocked-ip-1", expect: ui.ExecWaiting},
		{id: "blocked-open-1", expect: ui.ExecWaiting},
		{id: "done-1", expect: ui.ExecDone},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			state, ok := data.DeriveState(issueMap[tc.id], issueMap, data.DefaultBlockingTypes)
			if !ok {
				t.Fatalf("%s should be decidable", tc.id)
			}
			if got := statusColor(state); got != tc.expect {
				t.Fatalf("statusColor(%s) = %v, want %v", tc.id, got, tc.expect)
			}
		})
	}
}

func TestParadeViewEmpty(t *testing.T) {
	p := NewParade(nil, 60, 20, data.DefaultBlockingTypes)
	out := p.View()
	if !strings.Contains(out, "No issues found") {
		t.Fatalf("empty parade should contain 'No issues found', got: %s", out)
	}
}

func TestRenderIssueCursor(t *testing.T) {
	issues := []data.Issue{
		testIssue("cursor-1", data.StatusOpen),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, true, 0)
	if !strings.Contains(out, ui.Cursor) {
		t.Fatalf("renderIssue with selected=true should contain cursor %q, got: %s", ui.Cursor, out)
	}
}

func TestRenderIssueNoCursor(t *testing.T) {
	issues := []data.Issue{
		testIssue("nocursor-1", data.StatusOpen),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if strings.Contains(out, ui.Cursor) {
		t.Fatalf("renderIssue with selected=false should not contain cursor %q, got: %s", ui.Cursor, out)
	}
}

func TestRenderIssueMultiSelect(t *testing.T) {
	issues := []data.Issue{
		testIssue("sel-1", data.StatusOpen),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	p.Selected = map[string]bool{"sel-1": true}

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymSelected) {
		t.Fatalf("renderIssue with multi-select should contain %q, got: %s", ui.SymSelected, out)
	}
}

func TestRenderIssueChangedDot(t *testing.T) {
	issues := []data.Issue{
		testIssue("chg-1", data.StatusOpen),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	p.ChangedIDs = map[string]bool{"chg-1": true}

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymChanged) {
		t.Fatalf("renderIssue with ChangedIDs should contain %q, got: %s", ui.SymChanged, out)
	}
}

func TestRenderIssueOrphanBadge(t *testing.T) {
	issues := []data.Issue{
		testIssue("orph-1", data.StatusInProgress),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	p.OrphanedIDs = map[string]bool{"orph-1": true}

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymDeadRig) {
		t.Fatalf("renderIssue with OrphanedIDs should contain %q, got: %s", ui.SymDeadRig, out)
	}
}

func TestRenderIssueNoOrphanBadge(t *testing.T) {
	issues := []data.Issue{
		testIssue("norm-1", data.StatusOpen),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if strings.Contains(out, ui.SymDeadRig) {
		t.Fatalf("renderIssue without OrphanedIDs should not contain %q", ui.SymDeadRig)
	}
}

func TestRenderIssueZombieBadge(t *testing.T) {
	issues := []data.Issue{
		testIssue("zomb-1", data.StatusInProgress),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)
	p.ZombieIDs = map[string]bool{"zomb-1": true}

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymZombie) {
		t.Fatalf("renderIssue with ZombieIDs should contain zombie symbol, got: %s", out)
	}
}

func TestRenderIssueNoZombieBadge(t *testing.T) {
	issues := []data.Issue{
		testIssue("alive-1", data.StatusInProgress),
	}
	p := NewParade(issues, 80, 20, data.DefaultBlockingTypes)

	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}

	out := p.renderIssue(item, false, 0)
	if strings.Contains(out, ui.SymZombie) {
		t.Fatalf("renderIssue without ZombieIDs should not contain zombie symbol")
	}
}

func TestDetailViewNilIssue(t *testing.T) {
	d := Detail{Width: 60, Height: 20}
	d.Viewport = viewport.New(viewport.WithWidth(58), viewport.WithHeight(20))
	out := d.View()

	if !strings.Contains(out, "No issue selected") {
		t.Fatalf("detail with nil issue should contain 'No issue selected', got: %s", out)
	}
}

func TestDetailRenderContentSections(t *testing.T) {
	d := Detail{Width: 80, Height: 40, BlockingTypes: data.DefaultBlockingTypes}
	iss := testIssue("test-1", data.StatusOpen)
	iss.Description = "desc text"
	iss.Notes = "notes text"
	iss.AcceptanceCriteria = "ac text"
	iss.Design = "design text"
	iss.CloseReason = "reason text"
	d.Issue = &iss
	d.IssueMap = data.BuildIssueMap([]data.Issue{iss})
	out := d.renderContent()

	for _, section := range []string{"DESCRIPTION", "NOTES", "ACCEPTANCE CRITERIA", "DESIGN", "CLOSE REASON"} {
		if !strings.Contains(out, section) {
			t.Errorf("renderContent should contain section %q, got: %s", section, out)
		}
	}
}

func TestDetailRenderContentDependencies(t *testing.T) {
	blocker := testIssue("dep-1", data.StatusOpen)
	blocked := testIssue("blocked-1", data.StatusOpen)
	blocked.Dependencies = []data.Dependency{
		{IssueID: "blocked-1", DependsOnID: "dep-1", Type: "blocks"},
	}

	allIssues := []data.Issue{blocker, blocked}
	issueMap := data.BuildIssueMap(allIssues)

	d := Detail{
		Width:         80,
		Height:        40,
		BlockingTypes: data.DefaultBlockingTypes,
		Issue:         issueMap["blocked-1"],
		IssueMap:      issueMap,
		AllIssues:     allIssues,
	}
	out := d.renderContent()

	if !strings.Contains(out, "DEPENDENCIES") {
		t.Fatalf("renderContent for blocked issue should contain 'DEPENDENCIES', got: %s", out)
	}
	if !strings.Contains(out, "waiting on") {
		t.Fatalf("renderContent for blocked issue should contain 'waiting on', got: %s", out)
	}
}

func TestRenderIssueDueBadge(t *testing.T) {
	issues := []data.Issue{
		testIssue("due-1", data.StatusOpen),
	}
	// Set due in 2 days — should show SymDueDate badge
	due := time.Now().Add(2 * 24 * time.Hour)
	issues[0].DueAt = &due

	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}
	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymDueDate) {
		t.Fatalf("renderIssue with upcoming due should contain %q, got: %s", ui.SymDueDate, out)
	}
}

func TestRenderIssueOverdueBadge(t *testing.T) {
	issues := []data.Issue{
		testIssue("overdue-1", data.StatusOpen),
	}
	past := time.Now().Add(-3 * 24 * time.Hour)
	issues[0].DueAt = &past

	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}
	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymOverdue) {
		t.Fatalf("renderIssue with overdue should contain %q, got: %s", ui.SymOverdue, out)
	}
}

func TestRenderIssueDeferredDim(t *testing.T) {
	issues := []data.Issue{
		testIssue("defer-1", data.StatusOpen),
	}
	future := time.Now().Add(5 * 24 * time.Hour)
	issues[0].DeferUntil = &future

	p := NewParade(issues, 100, 20, data.DefaultBlockingTypes)
	var item ParadeItem
	for _, it := range p.Items {
		if it.Issue != nil {
			item = it
			break
		}
	}
	if item.Issue == nil {
		t.Fatal("no selectable item found")
	}
	out := p.renderIssue(item, false, 0)
	if !strings.Contains(out, ui.SymDeferred) {
		t.Fatalf("renderIssue with deferred should contain %q, got: %s", ui.SymDeferred, out)
	}
}

func TestRenderIssueHierarchicalIndent(t *testing.T) {
	parent := testIssue("mg-007", data.StatusOpen)
	child := testIssue("mg-007.1", data.StatusOpen)
	child.Dependencies = []data.Dependency{{IssueID: child.ID, DependsOnID: parent.ID, Type: "parent-child"}}
	grandchild := testIssue("mg-007.1.1", data.StatusOpen)
	grandchild.Dependencies = []data.Dependency{{IssueID: grandchild.ID, DependsOnID: child.ID, Type: "parent-child"}}

	issues := []data.Issue{parent, child, grandchild}
	p := NewParade(issues, 100, 30, data.DefaultBlockingTypes)

	// Collect rendered output for child and grandchild.
	// Strip ANSI codes before checking indent because lipgloss v2 wraps
	// each styled segment in its own escape sequences.
	for _, it := range p.Items {
		if it.Issue == nil {
			continue
		}
		out := ansi.Strip(p.renderIssue(it, false, 0))
		switch it.Issue.ID {
		case "mg-007":
			// Parent should not have extra indent (no leading spaces before sym)
		case "mg-007.1":
			// Depth 1 → 2 spaces of indent
			if !strings.Contains(out, "  "+ui.SymExecReady) {
				t.Errorf("child issue should be indented, got: %s", out)
			}
		case "mg-007.1.1":
			// Depth 2 → 4 spaces of indent
			if !strings.Contains(out, "    "+ui.SymExecReady) {
				t.Errorf("grandchild issue should be double-indented, got: %s", out)
			}
		}
	}
}

func TestDetailRenderContentDueDate(t *testing.T) {
	iss := testIssue("due-detail-1", data.StatusOpen)
	due := time.Now().Add(2 * 24 * time.Hour)
	iss.DueAt = &due

	d := Detail{
		Width:         80,
		Height:        40,
		BlockingTypes: data.DefaultBlockingTypes,
		Issue:         &iss,
		IssueMap:      data.BuildIssueMap([]data.Issue{iss}),
	}
	out := d.renderContent()
	if !strings.Contains(out, "Due:") {
		t.Fatalf("renderContent with DueAt set should contain 'Due:', got: %s", out)
	}
	if !strings.Contains(out, ui.SymDueDate) {
		t.Fatalf("renderContent with upcoming due should contain %q, got: %s", ui.SymDueDate, out)
	}
}

func TestDetailRenderContentRichDeps(t *testing.T) {
	target := testIssue("dep-target", data.StatusOpen)
	iss := testIssue("rich-dep-1", data.StatusOpen)
	iss.Dependencies = []data.Dependency{
		{IssueID: "rich-dep-1", DependsOnID: "dep-target", Type: "related"},
	}

	allIssues := []data.Issue{target, iss}
	issueMap := data.BuildIssueMap(allIssues)

	d := Detail{
		Width:         80,
		Height:        40,
		BlockingTypes: data.DefaultBlockingTypes,
		Issue:         issueMap["rich-dep-1"],
		IssueMap:      issueMap,
		AllIssues:     allIssues,
	}
	out := d.renderContent()
	if !strings.Contains(out, ui.SymRelated) {
		t.Fatalf("renderContent with related dep should contain %q, got: %s", ui.SymRelated, out)
	}
	if !strings.Contains(out, "related to") {
		t.Fatalf("renderContent with related dep should contain 'related to', got: %s", out)
	}
}

func TestDepTypeDisplay(t *testing.T) {
	tests := []struct {
		depType  string
		wantSym  string
		wantVerb string
	}{
		{"related", ui.SymRelated, "related to"},
		{"duplicates", ui.SymDuplicates, "duplicates"},
		{"supersedes", ui.SymSupersedes, "supersedes"},
		{"discovered-from", ui.SymNonBlocking, "discovered from"},
		{"waits-for", ui.SymStalled, "waits for"},
		{"parent-child", ui.DepTree, "child of"},
		{"replies-to", ui.SymNonBlocking, "replies to"},
		{"unknown-type", ui.SymNonBlocking, "unknown-type"},
	}
	for _, tc := range tests {
		t.Run(tc.depType, func(t *testing.T) {
			sym, verb, _ := depTypeDisplay(tc.depType)
			if sym != tc.wantSym {
				t.Errorf("depTypeDisplay(%q) sym = %q, want %q", tc.depType, sym, tc.wantSym)
			}
			if verb != tc.wantVerb {
				t.Errorf("depTypeDisplay(%q) verb = %q, want %q", tc.depType, verb, tc.wantVerb)
			}
		})
	}
}

func TestDetailRenderContentOwnerAssignee(t *testing.T) {
	iss := testIssue("owner-1", data.StatusOpen)
	iss.Owner = "alice"
	iss.Assignee = "bob"

	d := Detail{
		Width:         80,
		Height:        40,
		BlockingTypes: data.DefaultBlockingTypes,
		Issue:         &iss,
		IssueMap:      data.BuildIssueMap([]data.Issue{iss}),
	}
	out := d.renderContent()

	if !strings.Contains(out, "Owner:") {
		t.Fatalf("renderContent should contain 'Owner:', got: %s", out)
	}
	if !strings.Contains(out, "Assignee:") {
		t.Fatalf("renderContent should contain 'Assignee:', got: %s", out)
	}
}
