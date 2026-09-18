package views

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// settledEpicIssues is the live mard-nob shape: an in_progress epic whose
// children are every one closed, linked only by parent-child edges. Raw status
// alone reads this as active work.
func settledEpicIssues(children int) []data.Issue {
	now := time.Now()
	issues := []data.Issue{{
		ID: "mard-nob", Title: "Operator observability", Status: data.StatusInProgress,
		Priority: data.PriorityHigh, IssueType: data.TypeEpic, CreatedAt: now, UpdatedAt: now,
	}}
	for i := 1; i <= children; i++ {
		id := "mard-nob." + strconv.Itoa(i)
		issues = append(issues, data.Issue{
			ID: id, Title: "child " + strconv.Itoa(i), Status: data.StatusClosed,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now, UpdatedAt: now,
			Dependencies: []data.Dependency{{IssueID: id, DependsOnID: "mard-nob", Type: "parent-child"}},
		})
	}
	return issues
}

// statusRow returns the detail panel's Status line, stripped of styling.
func statusRow(content string) string {
	for _, line := range strings.Split(ansi.Strip(content), "\n") {
		if strings.Contains(line, "Status:") {
			return line
		}
	}
	return ""
}

func TestSemanticStatusHardExample(t *testing.T) {
	issues := settledEpicIssues(7)
	d := NewDetail(80, 30, issues)
	d.SetIssue(&issues[0])

	row := statusRow(d.renderContent())
	if strings.Contains(row, "(") || !strings.Contains(row, "○ Operator Attention") {
		t.Fatalf("status row = %q", row)
	}
}

func TestSemanticStatusUnmappedRendersRawOnly(t *testing.T) {
	now := time.Now()
	issues := []data.Issue{{
		ID: "draft-1", Title: "unfinished thought", Status: data.StatusDraft,
		Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now, UpdatedAt: now,
	}}
	d := NewDetail(80, 30, issues)
	d.SetIssue(&issues[0])

	row := statusRow(d.renderContent())
	if !strings.Contains(row, string(data.StatusDraft)) {
		t.Fatalf("unmapped issue should still show its raw status, got %q", row)
	}
	for _, state := range data.StateOrder() {
		if strings.Contains(row, state.Label()) {
			t.Errorf("unmapped issue must not render the state label %q: %q", state.Label(), row)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		expect string
	}{
		{name: "short string", input: "hello", maxLen: 10, expect: "hello"},
		{name: "exact fit", input: "hello", maxLen: 5, expect: "hello"},
		{name: "needs truncation", input: "hello world", maxLen: 8, expect: "hello..."},
		{name: "very short max", input: "hello", maxLen: 2, expect: "he"},
		{name: "max 3", input: "hello", maxLen: 3, expect: "hel"},
		{name: "empty string", input: "", maxLen: 5, expect: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.input, tc.maxLen)
			if got != tc.expect {
				t.Fatalf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expect)
			}
		})
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		width  int
		expect string
	}{
		{name: "no wrap needed", input: "short text", width: 20, expect: "short text"},
		{name: "wraps at word boundary", input: "hello world foo bar", width: 11, expect: "hello world\nfoo bar"},
		{name: "single long word", input: "superlongword", width: 5, expect: "superlongword"},
		{name: "empty string", input: "", width: 10, expect: ""},
		{name: "zero width", input: "hello", width: 0, expect: "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := wordWrap(tc.input, tc.width)
			if got != tc.expect {
				t.Fatalf("wordWrap(%q, %d) = %q, want %q", tc.input, tc.width, got, tc.expect)
			}
		})
	}
}

func TestSetIssueUpdatesContent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue Title", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
	}
	d := NewDetail(60, 20, issues)
	d.SetIssue(&issues[0])

	content := d.Viewport.View()
	if !strings.Contains(content, "Test Issue Title") {
		t.Fatalf("viewport content should contain issue title, got: %s", content)
	}
}

// Refresh polls call SetIssue with the same issue every tick. Scroll position
// must be preserved in that case, but reset when the user navigates to a
// different issue.
func TestSetIssuePreservesScrollOnSameIssue(t *testing.T) {
	longBody := strings.Repeat("paragraph line that wraps and contributes to height\n\n", 80)
	issues := []data.Issue{
		{ID: "mg-001", Title: "Long issue", Description: longBody, Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
		{ID: "mg-002", Title: "Other issue", Description: longBody, Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
	}
	d := NewDetail(60, 20, issues)
	d.SetIssue(&issues[0])

	d.Viewport.SetYOffset(15)
	want := d.Viewport.YOffset()
	if want == 0 {
		t.Fatalf("test setup: expected non-zero scroll offset after SetYOffset(15), got 0")
	}

	// Simulate a poll-driven re-render of the same issue (pointer may differ
	// even when ID is the same — use a fresh struct value with the same ID).
	same := issues[0]
	d.SetIssue(&same)
	if got := d.Viewport.YOffset(); got != want {
		t.Fatalf("scroll position not preserved on same-issue refresh: got YOffset=%d, want %d", got, want)
	}

	// Switching to a different issue must reset to top.
	d.SetIssue(&issues[1])
	if got := d.Viewport.YOffset(); got != 0 {
		t.Fatalf("scroll position not reset when switching issues: got YOffset=%d, want 0", got)
	}
}

func TestSetSizeUpdatesDimensions(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
	}
	d := NewDetail(60, 20, issues)

	d.SetSize(100, 30)
	if d.Width != 100 {
		t.Fatalf("Width = %d, want 100", d.Width)
	}
	if d.Height != 30 {
		t.Fatalf("Height = %d, want 30", d.Height)
	}
	if d.Viewport.Width() != 98 {
		t.Fatalf("Viewport.Width = %d, want 98 (width-2)", d.Viewport.Width())
	}
	if d.Viewport.Height() != 29 {
		t.Fatalf("Viewport.Height = %d, want 29 (height-1, scroll cue row)", d.Viewport.Height())
	}
}

func TestEpicProgressUsesDirectChildren(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-100", Title: "Platform migration", Status: data.StatusOpen, Priority: data.PriorityHigh, IssueType: data.TypeEpic, CreatedAt: time.Now()},
		{ID: "mg-100.1", Title: "Auth", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-100.1", DependsOnID: "mg-100", Type: "parent-child"}}},
		{ID: "mg-100.2", Title: "Billing", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-100.2", DependsOnID: "mg-100", Type: "parent-child"}}},
		{ID: "mg-100.2.1", Title: "Billing schema", Status: data.StatusClosed, Priority: data.PriorityLow, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-100.2.1", DependsOnID: "mg-100.2", Type: "parent-child"}}},
	}

	d := NewDetail(80, 30, issues)
	progress, ok := d.epicProgress(&issues[0])
	if !ok {
		t.Fatal("expected epic progress to be available")
	}
	if progress.Done != 1 || progress.Total != 2 {
		t.Fatalf("epicProgress() = %+v, want done=1 total=2", progress)
	}
	if progress.Label() != "1/2 (50%)" {
		t.Fatalf("progress.Label() = %q, want %q", progress.Label(), "1/2 (50%)")
	}
}

func TestEpicProgressRenderingInContent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-100", Title: "Platform migration", Status: data.StatusOpen, Priority: data.PriorityHigh, IssueType: data.TypeEpic, CreatedAt: time.Now()},
		{ID: "mg-100.1", Title: "Auth", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-100.1", DependsOnID: "mg-100", Type: "parent-child"}}},
		{ID: "mg-100.2", Title: "Billing", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-100.2", DependsOnID: "mg-100", Type: "parent-child"}}},
	}

	d := NewDetail(80, 30, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()
	if !strings.Contains(content, "Progress:") {
		t.Fatalf("content should contain Progress row, got: %s", content)
	}
	if !strings.Contains(content, "1/2 (50%)") {
		t.Fatalf("content should contain 1/2 progress label, got: %s", content)
	}
}

func TestSetMolecule(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 30, issues)
	d.SetIssue(&issues[0])

	dag := &gastown.DAGInfo{
		RootID:    "mg-001",
		RootTitle: "Test Issue",
		Nodes: map[string]*gastown.DAGNode{
			"s1": {ID: "s1", Title: "Design", Status: "done", Tier: 0},
			"s2": {ID: "s2", Title: "Implement", Status: "in_progress", Tier: 1},
		},
		TierGroups: [][]string{{"s1"}, {"s2"}},
	}
	progress := &gastown.MoleculeProgress{
		TotalSteps: 3,
		DoneSteps:  1,
		Percent:    33,
	}

	d.SetMolecule("mg-001", dag, progress)

	if d.MoleculeDAG != dag {
		t.Fatal("MoleculeDAG not set")
	}
	if d.MoleculeProgress != progress {
		t.Fatal("MoleculeProgress not set")
	}
	if d.MoleculeIssueID != "mg-001" {
		t.Fatalf("MoleculeIssueID = %q, want %q", d.MoleculeIssueID, "mg-001")
	}
}

func TestSetMoleculeClearsOnIssueChange(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Issue 1", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
		{ID: "mg-002", Title: "Issue 2", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 30, issues)
	d.SetIssue(&issues[0])

	dag := &gastown.DAGInfo{
		RootID: "mg-001",
		Nodes:  map[string]*gastown.DAGNode{"s1": {ID: "s1", Status: "done"}},
	}
	d.SetMolecule("mg-001", dag, nil)

	// Switch to a different issue
	d.SetIssue(&issues[1])

	if d.MoleculeDAG != nil {
		t.Fatal("MoleculeDAG should be cleared when switching issues")
	}
	if d.MoleculeIssueID != "" {
		t.Fatalf("MoleculeIssueID should be empty, got %q", d.MoleculeIssueID)
	}
}

func TestMoleculeRenderingInContent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	dag := &gastown.DAGInfo{
		RootID:    "mg-001",
		RootTitle: "Build Feature",
		Nodes: map[string]*gastown.DAGNode{
			"s1": {ID: "s1", Title: "Design", Status: "done", Tier: 0},
			"s2": {ID: "s2", Title: "Implement", Status: "in_progress", Tier: 1},
			"s3": {ID: "s3", Title: "Test", Status: "blocked", Tier: 2, Dependencies: []string{"s2"}},
		},
		TierGroups: [][]string{{"s1"}, {"s2"}, {"s3"}},
	}
	progress := &gastown.MoleculeProgress{
		TotalSteps: 3,
		DoneSteps:  1,
		Percent:    33,
	}
	d.SetMolecule("mg-001", dag, progress)

	content := d.renderContent()

	if !strings.Contains(content, "MOLECULE") {
		t.Error("content should contain MOLECULE section")
	}
	if !strings.Contains(content, "Design") {
		t.Error("content should contain step title 'Design'")
	}
	if !strings.Contains(content, "Implement") {
		t.Error("content should contain step title 'Implement'")
	}
	// DAG flow connectors between tiers
	if !strings.Contains(content, ui.SymDAGFlow) {
		t.Error("content should contain DAG flow connector between tiers")
	}
	// Step symbols
	if !strings.Contains(content, ui.SymStepDone) {
		t.Error("content should contain done step symbol")
	}
	if !strings.Contains(content, ui.SymStepActive) {
		t.Error("content should contain active step symbol")
	}
}

func TestMoleculeDAGParallelBranching(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	dag := &gastown.DAGInfo{
		RootID:    "mg-001",
		RootTitle: "Shiny Workflow",
		Nodes: map[string]*gastown.DAGNode{
			"s1": {ID: "s1", Title: "Design", Status: "done", Tier: 0},
			"s2": {ID: "s2", Title: "Implement A", Status: "in_progress", Tier: 1, Parallel: true},
			"s3": {ID: "s3", Title: "Implement B", Status: "in_progress", Tier: 1, Parallel: true},
			"s4": {ID: "s4", Title: "Test", Status: "blocked", Tier: 2},
			"s5": {ID: "s5", Title: "Submit", Status: "blocked", Tier: 3},
		},
		TierGroups: [][]string{{"s1"}, {"s2", "s3"}, {"s4"}, {"s5"}},
	}
	d.SetMolecule("mg-001", dag, nil)
	content := d.renderContent()

	// Parallel branch connectors
	if !strings.Contains(content, ui.SymDAGBranch) {
		t.Error("content should contain branch start connector for parallel nodes")
	}
	if !strings.Contains(content, ui.SymDAGJoin) {
		t.Error("content should contain branch end connector for parallel nodes")
	}
	if !strings.Contains(content, "Implement A") {
		t.Error("content should contain parallel step 'Implement A'")
	}
	if !strings.Contains(content, "Implement B") {
		t.Error("content should contain parallel step 'Implement B'")
	}
}

func TestMoleculeDAGFiveWayParallel(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	dag := &gastown.DAGInfo{
		RootID:    "mg-001",
		RootTitle: "Rule of Five",
		Nodes: map[string]*gastown.DAGNode{
			"s1": {ID: "s1", Title: "Implement", Status: "done", Tier: 0},
			"r1": {ID: "r1", Title: "Correctness", Status: "in_progress", Tier: 1, Parallel: true},
			"r2": {ID: "r2", Title: "Security", Status: "ready", Tier: 1, Parallel: true},
			"r3": {ID: "r3", Title: "Performance", Status: "ready", Tier: 1, Parallel: true},
			"r4": {ID: "r4", Title: "Maintainability", Status: "ready", Tier: 1, Parallel: true},
			"r5": {ID: "r5", Title: "Testing", Status: "ready", Tier: 1, Parallel: true},
			"s2": {ID: "s2", Title: "Submit", Status: "blocked", Tier: 2},
		},
		TierGroups: [][]string{{"s1"}, {"r1", "r2", "r3", "r4", "r5"}, {"s2"}},
	}
	d.SetMolecule("mg-001", dag, nil)
	content := d.renderContent()

	// Should have branch start, middle forks, and join
	if !strings.Contains(content, ui.SymDAGBranch) {
		t.Error("content should contain branch start for 5-way parallel")
	}
	if !strings.Contains(content, ui.SymDAGFork) {
		t.Error("content should contain fork connectors for middle parallel nodes")
	}
	if !strings.Contains(content, ui.SymDAGJoin) {
		t.Error("content should contain branch end for 5-way parallel")
	}
	// All five review aspects present
	for _, title := range []string{"Correctness", "Security", "Performance", "Maintainability", "Testing"} {
		if !strings.Contains(content, title) {
			t.Errorf("content should contain parallel step %q", title)
		}
	}
}

func TestMoleculeDAGCriticalPathTitles(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	dag := &gastown.DAGInfo{
		RootID:    "mg-001",
		RootTitle: "Feature",
		Nodes: map[string]*gastown.DAGNode{
			"s1": {ID: "s1", Title: "Design", Status: "done", Tier: 0},
			"s2": {ID: "s2", Title: "Implement", Status: "in_progress", Tier: 1},
			"s3": {ID: "s3", Title: "Submit", Status: "blocked", Tier: 2},
		},
		TierGroups:   [][]string{{"s1"}, {"s2"}, {"s3"}},
		CriticalPath: []string{"s1", "s2", "s3"},
	}
	d.SetMolecule("mg-001", dag, nil)
	content := d.renderContent()

	// Critical path shows titles, not IDs
	if !strings.Contains(content, "critical:") {
		t.Error("content should contain critical path line")
	}
	if !strings.Contains(content, "Design") {
		t.Error("critical path should use title 'Design' not ID 's1'")
	}
	if strings.Contains(content, "s1 ") {
		t.Error("critical path should not show raw IDs")
	}
	// Arrow separator
	if !strings.Contains(content, "→") {
		t.Error("critical path should use → separator")
	}
}

func TestActivityRenderingInContent(t *testing.T) {
	created := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2025, 1, 16, 14, 30, 0, 0, time.UTC)
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: created, UpdatedAt: updated},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "ACTIVITY") {
		t.Error("content should contain ACTIVITY section")
	}
	if !strings.Contains(content, "Created") {
		t.Error("content should contain 'Created' event")
	}
	if !strings.Contains(content, "Updated") {
		t.Error("content should contain 'Updated' event")
	}
}

func TestActivityWithAgent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.ActiveAgents = map[string]string{"mg-001": "polecat-1"}
	d.TownStatus = &gastown.TownStatus{
		Agents: []gastown.AgentRuntime{
			{Name: "polecat-1", Role: "polecat", State: "working", HookBead: "mg-001"},
		},
	}
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "polecat-1") {
		t.Error("content should show agent name in activity")
	}
}

func TestActivityWithClosedIssue(t *testing.T) {
	created := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	closed := time.Date(2025, 1, 17, 9, 0, 0, 0, time.UTC)
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusClosed,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: created, ClosedAt: &closed},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "Closed") {
		t.Error("content should contain 'Closed' event")
	}
}

func TestMoleculeProgressBar(t *testing.T) {
	bar := moleculeProgressBar(3, 10, 20)
	if bar == "" {
		t.Fatal("progress bar should not be empty")
	}
	if len([]rune(bar)) == 0 {
		t.Fatal("progress bar should have characters")
	}

	// Edge cases
	emptyBar := moleculeProgressBar(0, 0, 10)
	if emptyBar == "" {
		t.Fatal("zero-total bar should not be empty")
	}
}

func TestFormatTime(t *testing.T) {
	ts := time.Date(2025, 2, 15, 14, 30, 0, 0, time.UTC)
	got := formatTime(ts)
	if !strings.Contains(got, "Feb 15") {
		t.Errorf("formatTime should contain date, got %q", got)
	}

	// Zero time
	zero := formatTime(time.Time{})
	if strings.TrimSpace(zero) != "" {
		t.Errorf("zero time should be blank, got %q", zero)
	}
}

func TestGateStatusRendering(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Gated Issue", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.TownStatus = &gastown.TownStatus{
		Agents: []gastown.AgentRuntime{
			{Name: "Toast", Role: "polecat", State: "awaiting-gate", HookBead: "mg-001"},
		},
	}
	d.ActiveAgents = map[string]string{"mg-001": "Toast"}
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "GATE") {
		t.Error("content should contain GATE section when agent is awaiting-gate")
	}
	if !strings.Contains(content, "Waiting on gate") {
		t.Error("content should show 'Waiting on gate' indicator")
	}
	if !strings.Contains(content, "Toast") {
		t.Error("content should show agent name in gate section")
	}
}

func TestGateStatusNotShownWhenWorking(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Working Issue", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.TownStatus = &gastown.TownStatus{
		Agents: []gastown.AgentRuntime{
			{Name: "Toast", Role: "polecat", State: "working", HookBead: "mg-001"},
		},
	}
	d.SetIssue(&issues[0])

	gate := d.renderGateStatus()
	if gate != "" {
		t.Error("gate section should not render when agent state is 'working'")
	}
}

func TestGateStatusNotShownWithoutTownStatus(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "No GT", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	gate := d.renderGateStatus()
	if gate != "" {
		t.Error("gate section should not render without TownStatus")
	}
}

func TestCommentsRendering(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Commented Issue", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	comments := []gastown.Comment{
		{ID: "c-1", Author: "claude (Toast)", Body: "JWT validation needs refresh", Time: "2025-02-22T10:30:00Z"},
		{ID: "c-2", Author: "overseer", Body: "Approved, ship it", Time: "2025-02-22T11:15:00Z"},
	}
	d.SetComments("mg-001", comments)

	content := d.renderContent()

	if !strings.Contains(content, "COMMENTS (2)") {
		t.Error("content should contain 'COMMENTS (2)' section header")
	}
	if !strings.Contains(content, "claude (Toast)") {
		t.Error("content should contain comment author")
	}
	if !strings.Contains(content, "JWT validation") {
		t.Error("content should contain comment body")
	}
	if !strings.Contains(content, "overseer") {
		t.Error("content should contain second comment author")
	}
}

func TestCommentsNotShownWhenEmpty(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "No Comments", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if strings.Contains(content, "COMMENTS") {
		t.Error("content should not contain COMMENTS section when no comments")
	}
}

func TestCommentsClearedOnIssueSwitch(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Issue 1", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
		{ID: "mg-002", Title: "Issue 2", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	comments := []gastown.Comment{
		{ID: "c-1", Author: "test", Body: "Hello"},
	}
	d.SetComments("mg-001", comments)

	if len(d.Comments) != 1 {
		t.Fatal("comments should be set")
	}

	// Switch to different issue — comments should clear
	d.SetIssue(&issues[1])

	if d.Comments != nil {
		t.Error("comments should be cleared when switching issues")
	}
	if d.CommentsIssueID != "" {
		t.Errorf("CommentsIssueID should be empty, got %q", d.CommentsIssueID)
	}
}

func TestFormulaRecommendationRendered(t *testing.T) {
	issues := []data.Issue{
		{ID: "bd-001", Title: "Add authentication middleware", Status: data.StatusOpen,
			Priority: data.PriorityHigh, IssueType: data.TypeFeature, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "FORMULA") {
		t.Error("content should contain FORMULA section for open issue")
	}
	if !strings.Contains(content, "security-audit") {
		t.Error("content should contain security-audit recommendation for auth issue")
	}
}

func TestCrossRigDepsRendered(t *testing.T) {
	issues := []data.Issue{
		{ID: "bd-001", Title: "Fix token validation", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeBug, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{
				{IssueID: "bd-001", DependsOnID: "external:gastown:gt-c3f2", Type: "blocks"},
				{IssueID: "bd-001", DependsOnID: "external:wyvern:wy-e5f6", Type: "related"},
			},
		},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "CROSS-RIG") {
		t.Error("content should contain CROSS-RIG section")
	}
	if !strings.Contains(content, "gastown") {
		t.Error("content should contain rig name 'gastown'")
	}
	if !strings.Contains(content, "wyvern") {
		t.Error("content should contain rig name 'wyvern'")
	}
	if !strings.Contains(content, "gt-c3f2") {
		t.Error("content should contain external issue ID")
	}
}

func TestCrossRigDepsNotRenderedForLocalDeps(t *testing.T) {
	issues := []data.Issue{
		{ID: "bd-001", Title: "Local issue", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{
				{IssueID: "bd-001", DependsOnID: "bd-002", Type: "blocks"},
			},
		},
		{ID: "bd-002", Title: "Another local", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if strings.Contains(content, "CROSS-RIG") {
		t.Error("content should not contain CROSS-RIG section for local-only deps")
	}
}

func detailReferenceIssues() []data.Issue {
	now := time.Now()
	return []data.Issue{
		{
			ID: "detail-selected", Title: "Selected issue", Status: data.StatusClosed,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now,
			Dependencies: []data.Dependency{
				{IssueID: "detail-selected", DependsOnID: "detail-unresolved", Type: "blocks"},
				{IssueID: "detail-selected", DependsOnID: "detail-missing", Type: "blocks"},
				{IssueID: "detail-selected", DependsOnID: "detail-resolved", Type: "blocks"},
				{IssueID: "detail-selected", DependsOnID: "detail-related", Type: "related"},
				{IssueID: "detail-selected", DependsOnID: "detail-parent", Type: "parent-child"},
				{IssueID: "detail-selected", DependsOnID: "external:other-rig:detail-cross-rig", Type: "blocks"},
			},
		},
		{ID: "detail-unresolved", Title: "Unresolved blocker", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now},
		{ID: "detail-resolved", Title: "Resolved blocker", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now},
		{ID: "detail-related", Title: "Related issue", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now},
		{ID: "detail-parent", Title: "Parent issue", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeEpic, CreatedAt: now},
		{
			ID: "detail-dependent", Title: "Reverse dependent", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: now,
			Dependencies: []data.Dependency{
				{IssueID: "detail-dependent", DependsOnID: "detail-selected", Type: "blocks"},
			},
		},
	}
}

func detailReferenceLine(t *testing.T, content, id string) int {
	t.Helper()
	for line, text := range strings.Split(ansi.Strip(content), "\n") {
		if strings.Contains(text, id) {
			return line
		}
	}
	t.Fatalf("rendered content does not contain reference ID %q:\n%s", id, content)
	return -1
}

func TestDetailReferenceAt(t *testing.T) {
	issues := detailReferenceIssues()
	d := NewDetail(100, 80, issues)
	d.SetIssue(&issues[0])
	content := d.renderContent()

	for _, target := range []string{
		"detail-unresolved", "detail-resolved", "detail-related", "detail-parent", "detail-dependent",
	} {
		line := detailReferenceLine(t, content, target)
		if got := d.ReferenceAt(line - d.Viewport.YOffset()); got == nil || got.ID != target {
			t.Fatalf("ReferenceAt(%q) = %#v, want loaded issue", target, got)
		}
	}
}

func TestDetailReferenceAtScrolled(t *testing.T) {
	issues := detailReferenceIssues()
	d := NewDetail(100, 10, issues)
	d.SetIssue(&issues[0])
	content := d.renderContent()

	line := detailReferenceLine(t, content, "detail-dependent")
	d.Viewport.SetYOffset(line - 1)
	if d.Viewport.YOffset() == 0 {
		t.Fatal("test setup: expected a non-zero viewport offset")
	}
	if got := d.ReferenceAt(line - d.Viewport.YOffset()); got == nil || got.ID != "detail-dependent" {
		t.Fatalf("ReferenceAt(scrolled row) = %#v, want detail-dependent", got)
	}
}

func TestDetailMissingReference(t *testing.T) {
	issues := detailReferenceIssues()
	d := NewDetail(100, 80, issues)
	d.SetIssue(&issues[0])
	content := d.renderContent()

	for _, id := range []string{"detail-missing", "detail-cross-rig"} {
		line := detailReferenceLine(t, content, id)
		if got := d.ReferenceAt(line - d.Viewport.YOffset()); got != nil {
			t.Fatalf("ReferenceAt(%q) = %#v, want nil for unloaded target", id, got)
		}
	}
}
func TestFormulaRecommendationNotRenderedForClosed(t *testing.T) {
	issues := []data.Issue{
		{ID: "bd-001", Title: "Add authentication middleware", Status: data.StatusClosed,
			Priority: data.PriorityHigh, IssueType: data.TypeFeature, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if strings.Contains(content, "FORMULA") {
		t.Error("content should not contain FORMULA section for closed issue")
	}
}

func TestSetCommentsUpdatesContent(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	comments := []gastown.Comment{
		{ID: "c-1", Author: "reviewer", Body: "Looks good"},
	}
	d.SetComments("mg-001", comments)

	if d.CommentsIssueID != "mg-001" {
		t.Fatalf("CommentsIssueID = %q, want %q", d.CommentsIssueID, "mg-001")
	}
	if len(d.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(d.Comments))
	}
}

func TestMetadataSchemaRendered(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Test Issue", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	min0 := 0.0
	max100 := 100.0
	d.MetadataSchema = &data.MetadataSchema{
		Mode: "warn",
		Fields: map[string]data.MetadataFieldSchema{
			"team": {
				Type:     data.MetaEnum,
				Required: true,
				Values:   []string{"platform", "frontend", "backend"},
			},
			"priority_score": {
				Type: data.MetaInt,
				Min:  &min0,
				Max:  &max100,
			},
		},
	}
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "METADATA") {
		t.Error("content should contain METADATA section")
	}
	if !strings.Contains(content, "warn") {
		t.Error("content should contain mode 'warn'")
	}
	if !strings.Contains(content, "team") {
		t.Error("content should contain field name 'team'")
	}
	if !strings.Contains(content, "enum") {
		t.Error("content should contain field type 'enum'")
	}
	if !strings.Contains(content, "priority_score") {
		t.Error("content should contain field name 'priority_score'")
	}
}

func TestMetadataNotRenderedWithoutSchema(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "No Metadata", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if strings.Contains(content, "METADATA") {
		t.Error("content should not contain METADATA section without schema")
	}
}

func TestMetadataWithIssueValues(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "With Metadata", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"team":   "frontend",
				"urgent": true,
			},
		},
	}
	d := NewDetail(80, 40, issues)
	d.MetadataSchema = &data.MetadataSchema{
		Mode: "warn",
		Fields: map[string]data.MetadataFieldSchema{
			"team": {
				Type:     data.MetaEnum,
				Required: true,
				Values:   []string{"platform", "frontend", "backend"},
			},
			"urgent": {
				Type: data.MetaBool,
			},
		},
	}
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "METADATA") {
		t.Error("content should contain METADATA section")
	}
	if !strings.Contains(content, "frontend") {
		t.Error("content should contain metadata value 'frontend'")
	}
	if !strings.Contains(content, "true") {
		t.Error("content should contain metadata value 'true'")
	}
}

func TestMetadataRawValuesWithoutSchema(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Raw Metadata", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"custom_field": "value123",
			},
		},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	content := d.renderContent()

	if !strings.Contains(content, "METADATA") {
		t.Error("content should contain METADATA section for raw metadata")
	}
	if !strings.Contains(content, "custom_field") {
		t.Error("content should contain raw metadata key")
	}
	if !strings.Contains(content, "value123") {
		t.Error("content should contain raw metadata value")
	}
}

func TestSetRichDetailWritesMatchingID(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Stale", Status: data.StatusOpen,
			Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	rich := &data.Issue{
		ID:                 "mg-001",
		Notes:              "freshly fetched notes",
		Design:             "design doc",
		AcceptanceCriteria: "must roll",
	}
	d.SetRichDetail("mg-001", rich)

	if d.Issue.Notes != "freshly fetched notes" {
		t.Errorf("Notes not written: %q", d.Issue.Notes)
	}
	if d.Issue.Design != "design doc" {
		t.Errorf("Design not written: %q", d.Issue.Design)
	}
	if d.Issue.AcceptanceCriteria != "must roll" {
		t.Errorf("AcceptanceCriteria not written: %q", d.Issue.AcceptanceCriteria)
	}
	if d.RichIssueID != "mg-001" {
		t.Errorf("RichIssueID not set: %q", d.RichIssueID)
	}
}

func TestSetRichDetailIgnoresMismatchedID(t *testing.T) {
	// User scrolled to mg-002 while a `bd show mg-001` fetch was in flight.
	// The late-arriving rich detail must NOT clobber the now-displayed issue.
	issues := []data.Issue{
		{ID: "mg-001", Notes: "should-stay-empty"},
		{ID: "mg-002", Title: "Selected"},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[1]) // user is now on mg-002

	rich := &data.Issue{ID: "mg-001", Notes: "stale fetch result"}
	d.SetRichDetail("mg-001", rich)

	if d.Issue.Notes != "" {
		t.Errorf("mg-002 Notes was clobbered by stale mg-001 fetch: %q", d.Issue.Notes)
	}
}

func TestSetRichDetailEmptyFieldsDoNotClobber(t *testing.T) {
	// Each field has a non-empty guard so a sparse `bd show` response
	// (e.g. AcceptanceCriteria not yet authored) does not wipe content
	// that was already populated from the parade JSONL.
	issues := []data.Issue{
		{ID: "mg-001", Notes: "existing notes", Design: "existing design"},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])

	rich := &data.Issue{ID: "mg-001"} // all rich fields empty
	d.SetRichDetail("mg-001", rich)

	if d.Issue.Notes != "existing notes" {
		t.Errorf("empty Notes clobbered existing: %q", d.Issue.Notes)
	}
	if d.Issue.Design != "existing design" {
		t.Errorf("empty Design clobbered existing: %q", d.Issue.Design)
	}
}

func TestSetRichDetailNoIssueIsNoOp(t *testing.T) {
	d := NewDetail(80, 40, nil)
	// No SetIssue call — d.Issue is nil. SetRichDetail must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SetRichDetail panicked with nil Issue: %v", r)
		}
	}()
	d.SetRichDetail("mg-001", &data.Issue{ID: "mg-001", Notes: "x"})
}

// TestFocusChangesBorder verifies the detail pane's border (the divider + focus
// cue) renders differently when focused vs not — gold when focused, dim when not.
func TestFocusChangesBorder(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Focus test", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask},
	}
	d := NewDetail(60, 20, issues)
	d.SetIssue(&issues[0])

	d.Focused = false
	unfocused := d.View()
	d.Focused = true
	focused := d.View()

	if unfocused == focused {
		t.Fatal("detail View() should differ between focused and unfocused (border cue)")
	}
}

// TestRenderMarkdownPreservesAngleBracketSpans guards against silent text loss
// in issue bodies: a tag-shaped span like "<your-name>" must render verbatim.
// See ui.NewMarkdownRenderer for why the raw-HTML parsers are left out, and
// why that is done at the parser rather than by escaping around the render.
func TestRenderMarkdownPreservesAngleBracketSpans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "intra-line tag-shaped spans survive",
			input: "LINE1: replace <your-name> with the rig\n" +
				"LINE2: literal a<b and c>d\n" +
				"LINE3: html-ish <em>text</em> tail\n",
			want: []string{
				"LINE1: replace <your-name> with the rig",
				"LINE2: literal a<b and c>d",
				"LINE3: html-ish <em>text</em> tail",
			},
		},
		{
			// The worst case: an unclosed tag used to swallow every line up
			// to the next ">", not just its own span.
			name: "unclosed tag-shaped token does not eat following lines",
			input: "P: start <unclosed\n" +
				"Q: middle line one\n" +
				"R: middle line two\n" +
				"S: end > tail\n",
			want: []string{
				"P: start <unclosed",
				"Q: middle line one",
				"R: middle line two",
				"S: end > tail",
			},
		},
		{
			name:  "non-tag-shaped angle brackets stay intact",
			input: "E1: use Map<string, int> and <123> for the cache",
			want:  []string{"Map<string, int>", "<123>"},
		},
		{
			// Code spans and syntax-highlighted fences are where an escaping
			// approach breaks down, so they get their own case.
			name: "angle brackets survive code spans and fences",
			input: "inline `foo <bar> baz` span\n\n" +
				"```go\nif a < b && c > d { f() }\n```\n",
			want: []string{"foo <bar> baz", "if a < b && c > d { f() }"},
		},
	}

	d := &Detail{Width: 80, Height: 24}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := ansi.Strip(d.renderMarkdown(tt.input))
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("expected rendered output to contain %q\ngot: %q", want, out)
				}
			}
			// No escaping happens any more, so an entity in the output would
			// mean someone reintroduced a round-trip that does not round-trip.
			if strings.Contains(out, "&lt;") || strings.Contains(out, "&gt;") {
				t.Errorf("HTML entity leaked into rendered output\ngot: %q", out)
			}
		})
	}
}

// TestRenderMarkdownStylesCodeFenceOperators pins the styling half of the fix.
// TestRenderMarkdownPreservesAngleBracketSpans strips ANSI, so it cannot see a
// bracket that survives as text but renders as a chroma lexer error — which is
// what any escape sentinel inside a fence produces. Compare the SGR sequence
// around "<" with the one around a comparison operator chroma always accepts.
func TestRenderMarkdownStylesCodeFenceOperators(t *testing.T) {
	d := &Detail{Width: 80, Height: 24}
	angle := d.renderMarkdown("```go\nif a < b { f() }\n```\n")
	baseline := d.renderMarkdown("```go\nif a != b { f() }\n```\n")

	// Chroma's Error token carries a background color; a normal operator does
	// not. Any "48;5;" (background) in the angle-bracket render that the
	// baseline lacks means "<" was lexed as an error.
	if strings.Contains(angle, "\x1b[48;5;") && !strings.Contains(baseline, "\x1b[48;5;") {
		t.Errorf("'<' in a code fence rendered with an error background\nangle:    %q\nbaseline: %q", angle, baseline)
	}
}

// TestRenderMarkdownShowsBlockHTMLLiterally pins the deliberate trade-off of
// dropping goldmark's HTML parsers: real HTML in an issue body now renders as
// literal text instead of being sanitized away. That is the point. With the
// parsers installed, glamour ran every HTML node through bluemonday's
// StrictPolicy, so "<img src=...>" alone rendered as an *empty document* and an
// HTML comment vanished without trace — the same silent-deletion class as the
// bug this file exists to prevent. Showing the markup is the honest failure
// mode: bd issue bodies are never re-rendered as HTML anywhere in mg.
func TestRenderMarkdownShowsBlockHTMLLiterally(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"image tag is not swallowed", `<img src="x.png" alt="pic">`, `<img src="x.png" alt="pic">`},
		{"html comment stays visible", "before\n\n<!-- a hidden note -->\n\nafter", "<!-- a hidden note -->"},
		{"details block keeps its tags", "<details>\n<summary>Click</summary>\n\nbody\n\n</details>", "<summary>Click</summary>"},
		{"line break tag is literal", "line one<br>line two", "line one<br>line two"},
	}

	d := &Detail{Width: 80, Height: 24}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := ansi.Strip(d.renderMarkdown(tt.input))
			if !strings.Contains(out, tt.want) {
				t.Errorf("expected rendered output to contain %q\ngot: %q", tt.want, out)
			}
		})
	}
}

// TestEpicProgressIgnoresReparentedChild pins the #110 fix: epic progress used
// to count children by dotted-ID prefix, so an issue reparented away from an
// epic kept counting toward it. `bd create --parent` writes both the dotted ID
// and a parent-child edge; removing the edge while keeping the ID is exactly
// the reparenting case, and the edge is what decides membership now.
func TestEpicProgressIgnoresReparentedChild(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-100", Title: "Platform migration", Status: data.StatusOpen, Priority: data.PriorityHigh, IssueType: data.TypeEpic, CreatedAt: time.Now()},
		{ID: "mg-100.1", Title: "Still a child", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-100.1", DependsOnID: "mg-100", Type: "parent-child"}}},
		// Reparented to the top level: dotted ID retained, edge gone.
		{ID: "mg-100.2", Title: "Moved out", Status: data.StatusOpen, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now()},
	}

	d := NewDetail(80, 30, issues)
	progress, ok := d.epicProgress(&issues[0])
	if !ok {
		t.Fatal("expected epic progress to be available")
	}
	if progress.Total != 1 || progress.Done != 1 {
		t.Fatalf("epicProgress() = %+v, want done=1 total=1 — the reparented issue must not count", progress)
	}
}

// TestEpicProgressCountsEdgeOnlyChild covers the converse: a child linked only
// by a parent-child edge, with no dotted ID, never counted before.
func TestEpicProgressCountsEdgeOnlyChild(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-100", Title: "Platform migration", Status: data.StatusOpen, Priority: data.PriorityHigh, IssueType: data.TypeEpic, CreatedAt: time.Now()},
		{ID: "mg-777", Title: "Adopted child", Status: data.StatusClosed, Priority: data.PriorityMedium, IssueType: data.TypeTask, CreatedAt: time.Now(),
			Dependencies: []data.Dependency{{IssueID: "mg-777", DependsOnID: "mg-100", Type: "parent-child"}}},
	}

	d := NewDetail(80, 30, issues)
	progress, ok := d.epicProgress(&issues[0])
	if !ok {
		t.Fatal("expected epic progress for an edge-only child")
	}
	if progress.Total != 1 || progress.Done != 1 {
		t.Fatalf("epicProgress() = %+v, want done=1 total=1", progress)
	}
}

func TestCommentsRenderingMarkdown(t *testing.T) {
	issues := []data.Issue{
		{ID: "mg-001", Title: "Commented Issue", Status: data.StatusInProgress,
			Priority: data.PriorityMedium, IssueType: data.TypeTask,
			CreatedAt: time.Now()},
	}
	d := NewDetail(60, 40, issues)
	d.SetIssue(&issues[0])

	d.SetComments("mg-001", []gastown.Comment{
		{ID: "c-1", Author: "reviewer", Time: "2025-02-22T10:30:00Z",
			Body: "Needs **refresh token** handling:\n\n- rotate on use\n- revoke on logout\n\n" +
				strings.Repeat("long ", 40)},
	})

	plain := ansi.Strip(d.renderComments())

	// Markdown is rendered, not shown as raw markup.
	if strings.Contains(plain, "**") {
		t.Errorf("comment body should render bold, not show literal ** markers:\n%s", plain)
	}
	if !strings.Contains(plain, "refresh token") {
		t.Errorf("comment body text missing from rendered output:\n%s", plain)
	}
	if !strings.Contains(plain, "rotate on use") || !strings.Contains(plain, "revoke on logout") {
		t.Errorf("list items missing from rendered output:\n%s", plain)
	}
	if strings.Contains(plain, "- rotate") {
		t.Errorf("list should render with a bullet, not a literal dash:\n%s", plain)
	}

	// Indented, wrapped comment lines must still fit inside the viewport
	// (panel width minus the left border and padding).
	limit := d.Width - 2
	for _, line := range strings.Split(plain, "\n") {
		if w := ansi.StringWidth(line); w > limit {
			t.Errorf("comment line width %d exceeds viewport width %d: %q", w, limit, line)
		}
	}
}

func TestDetailBodyMentionIsClickable(t *testing.T) {
	now := time.Now()
	issues := []data.Issue{
		{ID: "src-aa", Title: "Source", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask,
			CreatedAt: now, Description: "See other-bb and also src-aa itself."},
		{ID: "other-bb", Title: "Other", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask, CreatedAt: now},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])
	content := d.renderContent()
	plain := ansi.Strip(content)
	line := -1
	col := -1
	selfCol := -1
	for i, text := range strings.Split(plain, "\n") {
		if j := strings.Index(text, "other-bb"); j >= 0 {
			line, col = i, j
		}
		if j := strings.Index(text, "src-aa"); j >= 0 && !strings.Contains(text, "ID:") {
			selfCol = j
			if line < 0 {
				line = i
			}
		}
	}
	if line < 0 || col < 0 {
		t.Fatalf("other-bb not in body:\n%s", plain)
	}
	if got := d.ReferenceAtXY(line-d.Viewport.YOffset(), col); got == nil || got.ID != "other-bb" {
		t.Fatalf("click other-bb = %#v, want other-bb", got)
	}
	if selfCol >= 0 {
		if got := d.ReferenceAtXY(line-d.Viewport.YOffset(), selfCol); got != nil {
			t.Fatalf("self mention must not be a link, got %#v", got)
		}
	}
	idLine := detailReferenceLine(t, content, "src-aa")
	// The metadata ID row contains src-aa; that field is not a mention span.
	// Walk the ID: row and assert no XY hit on the current ID.
	for i, text := range strings.Split(plain, "\n") {
		if strings.Contains(text, "ID:") && strings.Contains(text, "src-aa") {
			j := strings.Index(text, "src-aa")
			if got := d.ReferenceAtXY(i-d.Viewport.YOffset(), j); got != nil {
				t.Fatalf("ID field must not be a link, got %#v", got)
			}
			_ = idLine
			break
		}
	}
}

func TestDetailSelfMentionIsGoldNotUnderlined(t *testing.T) {
	now := time.Now()
	issues := []data.Issue{
		{ID: "only-me", Title: "Solo", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask,
			CreatedAt: now, Description: "This bead is only-me and names no one else."},
	}
	d := NewDetail(80, 24, issues)
	d.SetIssue(&issues[0])
	var styled string
	for _, line := range strings.Split(d.renderContent(), "\n") {
		if strings.Contains(ansi.Strip(line), "only-me") && !strings.Contains(ansi.Strip(line), "ID:") {
			styled = line
			break
		}
	}
	if styled == "" {
		t.Fatal("self mention line missing")
	}
	if strings.Contains(styled, "\x1b[4m") || strings.Contains(styled, "4m") && strings.Contains(styled, "only-me") && strings.Contains(styled, "[4") {
		// underline SGR is 4. Gold self-mentions must not be underlined.
		if strings.Contains(styled, "[4m") {
			t.Fatalf("self mention should not be underlined: %q", styled)
		}
	}
}

func TestDetailClosedMentionIsClickable(t *testing.T) {
	now := time.Now()
	issues := []data.Issue{
		{ID: "src-aa", Title: "Source", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask,
			CreatedAt: now, Description: "See closed-zz after it shipped."},
		{ID: "closed-zz", Title: "Done work", Status: data.StatusClosed, Priority: 1, IssueType: data.TypeTask, CreatedAt: now},
	}
	d := NewDetail(80, 40, issues)
	d.SetIssue(&issues[0])
	plain := ansi.Strip(d.renderContent())
	line, col := -1, -1
	for i, text := range strings.Split(plain, "\n") {
		if j := strings.Index(text, "closed-zz"); j >= 0 {
			line, col = i, j
			break
		}
	}
	if line < 0 {
		t.Fatalf("closed-zz not in body:\n%s", plain)
	}
	if got := d.ReferenceAtXY(line-d.Viewport.YOffset(), col); got == nil || got.ID != "closed-zz" {
		t.Fatalf("closed mention = %#v, want closed-zz", got)
	}
}

func TestDetailDependencyIDIsLinked(t *testing.T) {
	issues := detailReferenceIssues()
	d := NewDetail(100, 80, issues)
	d.SetIssue(&issues[0])
	content := d.renderContent()
	plain := ansi.Strip(content)
	var line, col int
	found := false
	for i, text := range strings.Split(plain, "\n") {
		if strings.Contains(text, "child of") {
			if j := strings.Index(text, "detail-parent"); j >= 0 {
				line, col, found = i, j, true
				break
			}
		}
	}
	if !found {
		t.Fatalf("child-of row missing:\n%s", plain)
	}
	if got := d.ReferenceAtXY(line-d.Viewport.YOffset(), col); got == nil || got.ID != "detail-parent" {
		t.Fatalf("click parent ID = %#v, want detail-parent", got)
	}
	// Closed resolved blocker must also be a link.
	found = false
	for i, text := range strings.Split(plain, "\n") {
		if strings.Contains(text, "resolved") {
			if j := strings.Index(text, "detail-resolved"); j >= 0 {
				line, col, found = i, j, true
				break
			}
		}
	}
	if !found {
		t.Fatalf("resolved row missing:\n%s", plain)
	}
	if got := d.ReferenceAtXY(line-d.Viewport.YOffset(), col); got == nil || got.ID != "detail-resolved" {
		t.Fatalf("click closed resolved ID = %#v, want detail-resolved", got)
	}
}
