package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/matt-wright86/mardi-gras/internal/components"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
	"github.com/matt-wright86/mardi-gras/internal/views"
)

// ---------------------------------------------------------------------------
// Key state-transition tests
// ---------------------------------------------------------------------------

// setupModel creates a standard model with open and closed issues, sized to
// 100x20, ready for key dispatch tests.
func setupModel(t *testing.T) Model {
	t.Helper()
	issues := []data.Issue{
		testIssue("open-1", data.StatusOpen),
		testIssue("open-2", data.StatusOpen),
		testIssue("open-3", data.StatusOpen),
		testIssue("closed-1", data.StatusClosed),
	}
	m := New(issues, data.Source{}, data.DefaultBlockingTypes)
	m.startedAt = time.Now().Add(-time.Second) // bypass startup guard
	// Pin the Gas Town driver so key handling does not depend on what is
	// installed on the host: New() selects by environment, and a Gas City
	// driver makes orchestrator keys live even with gtEnv.Available false.
	m.driver = gastown.NewGTDriver()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	got := model.(Model)
	for i, item := range got.parade.Items {
		if item.Issue != nil && item.Issue.ID == "open-1" {
			got.parade.Cursor = i
			got.parade.SelectedIssue = item.Issue
			break
		}
	}
	return got
}

// ---------------------------------------------------------------------------
// 1. q quits
// ---------------------------------------------------------------------------

func TestKeyQQuits(t *testing.T) {
	got := setupModel(t)

	_, cmd := got.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit command from pressing q")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

// ---------------------------------------------------------------------------
// 2. ? opens help
// ---------------------------------------------------------------------------

func TestKeyQuestionOpensHelp(t *testing.T) {
	got := setupModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	got = model.(Model)

	if !got.showHelp {
		t.Fatal("expected showHelp to be true after pressing ?")
	}
}

// ---------------------------------------------------------------------------
// 3. / enters filter mode
// ---------------------------------------------------------------------------

func TestKeySlashEntersFilter(t *testing.T) {
	got := setupModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	got = model.(Model)

	if !got.filtering {
		t.Fatal("expected filtering to be true after pressing /")
	}
}

// ---------------------------------------------------------------------------
// 4. tab toggles panes
// ---------------------------------------------------------------------------

func TestKeyTabTogglesPanes(t *testing.T) {
	got := setupModel(t)

	if got.activPane != PaneParade {
		t.Fatalf("expected initial activPane to be PaneParade, got %d", got.activPane)
	}

	// Tab: Parade -> Detail
	model, _ := got.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got = model.(Model)
	if got.activPane != PaneDetail {
		t.Fatalf("expected activPane PaneDetail after first tab, got %d", got.activPane)
	}

	// Tab: Detail -> Parade
	model, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got = model.(Model)
	if got.activPane != PaneParade {
		t.Fatalf("expected activPane PaneParade after second tab, got %d", got.activPane)
	}
}

// ---------------------------------------------------------------------------
// 5. esc exits focus mode
// ---------------------------------------------------------------------------

func TestKeyEscExitsFocusMode(t *testing.T) {
	got := setupModel(t)
	got.focusMode = true

	model, _ := got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = model.(Model)

	if got.focusMode {
		t.Fatal("expected focusMode to be false after pressing esc")
	}
}

// ---------------------------------------------------------------------------
// 6. esc moves from detail to parade
// ---------------------------------------------------------------------------

func TestKeyEscDetailToParade(t *testing.T) {
	got := setupModel(t)
	got.activPane = PaneDetail

	model, _ := got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = model.(Model)

	if got.activPane != PaneParade {
		t.Fatalf("expected activPane PaneParade after esc from detail, got %d", got.activPane)
	}
}

// ---------------------------------------------------------------------------
// 7. f toggles focus mode
// ---------------------------------------------------------------------------

func TestKeyFTogglesFocus(t *testing.T) {
	got := setupModel(t)

	if got.focusMode {
		t.Fatal("expected focusMode to be false initially")
	}

	// Toggle ON
	model, cmd := got.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	got = model.(Model)
	if !got.focusMode {
		t.Fatal("expected focusMode to be true after pressing f")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (toast) after pressing f")
	}

	// Toggle OFF
	model, cmd = got.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	got = model.(Model)
	if got.focusMode {
		t.Fatal("expected focusMode to be false after pressing f again")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (toast) after pressing f again")
	}
}

// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 9. N opens create form
// ---------------------------------------------------------------------------

func TestKeyNOpensCreateForm(t *testing.T) {
	got := setupModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'N', Text: "N"})
	got = model.(Model)

	if !got.creating {
		t.Fatal("expected creating to be true after pressing N")
	}
}

// ---------------------------------------------------------------------------
// 10. enter switches to detail pane
// ---------------------------------------------------------------------------

func TestKeyEnterSwitchesToDetail(t *testing.T) {
	got := setupModel(t)

	if got.activPane != PaneParade {
		t.Fatal("expected activPane to be PaneParade initially")
	}

	model, _ := got.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = model.(Model)

	if got.activPane != PaneDetail {
		t.Fatalf("expected activPane PaneDetail after enter, got %d", got.activPane)
	}
}

// ---------------------------------------------------------------------------
// 11. g jumps to top (first selectable item)
// ---------------------------------------------------------------------------

func TestKeyGJumpsToTop(t *testing.T) {
	got := setupModel(t)

	// Move cursor down a few times first
	model, _ := got.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	got = model.(Model)
	model, _ = got.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	got = model.(Model)

	// Press g to jump to top
	model, _ = got.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	got = model.(Model)

	// The cursor should be on the first non-header item
	if got.parade.Cursor >= len(got.parade.Items) {
		t.Fatal("cursor out of range")
	}
	for i := 0; i < got.parade.Cursor; i++ {
		if !got.parade.Items[i].IsHeader {
			t.Fatalf("expected cursor at first non-header item, but item %d is not a header", i)
		}
	}
	if got.parade.Items[got.parade.Cursor].IsHeader {
		t.Fatal("expected cursor to be on a non-header item after pressing g")
	}
}

// ---------------------------------------------------------------------------
// 12. G jumps to bottom (last selectable item)
// ---------------------------------------------------------------------------

func TestKeyGGJumpsToBottom(t *testing.T) {
	got := setupModel(t)

	// First expand closed so we have more items
	model, _ := got.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	got = model.(Model)

	// Press G to jump to bottom
	model, _ = got.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	got = model.(Model)

	// The cursor should be on the last non-header item
	if got.parade.Cursor >= len(got.parade.Items) {
		t.Fatal("cursor out of range")
	}
	if got.parade.Items[got.parade.Cursor].IsHeader {
		t.Fatal("expected cursor to be on a non-header item after pressing G")
	}
	// Verify no non-header items exist after the cursor
	for i := got.parade.Cursor + 1; i < len(got.parade.Items); i++ {
		if !got.parade.Items[i].IsHeader {
			t.Fatalf("expected no non-header items after cursor at %d, but item %d is selectable", got.parade.Cursor, i)
		}
	}
}

// ---------------------------------------------------------------------------
// 13. j/k navigation
// ---------------------------------------------------------------------------

func TestKeyJKNavigation(t *testing.T) {
	got := setupModel(t)

	startCursor := got.parade.Cursor

	// j moves down
	model, _ := got.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	got = model.(Model)
	afterJ := got.parade.Cursor
	if afterJ <= startCursor {
		t.Fatalf("expected cursor to move down with j: start=%d, after=%d", startCursor, afterJ)
	}

	// k moves back up
	model, _ = got.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	got = model.(Model)
	afterK := got.parade.Cursor
	if afterK >= afterJ {
		t.Fatalf("expected cursor to move up with k: afterJ=%d, afterK=%d", afterJ, afterK)
	}
}

// ---------------------------------------------------------------------------
// 14. space toggles selection
// ---------------------------------------------------------------------------

func TestKeySpaceTogglesSelect(t *testing.T) {
	got := setupModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	got = model.(Model)

	if len(got.parade.Selected) == 0 {
		t.Fatal("expected parade.Selected to be non-empty after pressing space")
	}
}

// ---------------------------------------------------------------------------
// 15. X clears selection
// ---------------------------------------------------------------------------

func TestKeyXClearsSelection(t *testing.T) {
	got := setupModel(t)

	// Select an item first
	model, _ := got.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	got = model.(Model)
	if len(got.parade.Selected) == 0 {
		t.Fatal("expected parade.Selected to be non-empty after space")
	}

	// X clears all selections
	model, _ = got.Update(tea.KeyPressMsg{Code: 'X', Text: "X"})
	got = model.(Model)
	if len(got.parade.Selected) != 0 {
		t.Fatalf("expected parade.Selected to be empty after X, got %d", len(got.parade.Selected))
	}
}

// ---------------------------------------------------------------------------
// 16. J (shift+j) select + move down
// ---------------------------------------------------------------------------

func TestKeyJShiftSelectMoves(t *testing.T) {
	got := setupModel(t)

	startCursor := got.parade.Cursor

	model, _ := got.Update(tea.KeyPressMsg{Code: 'J', Text: "J"})
	got = model.(Model)

	if len(got.parade.Selected) == 0 {
		t.Fatal("expected parade.Selected to be non-empty after J")
	}
	if got.parade.Cursor <= startCursor {
		t.Fatalf("expected cursor to move down after J: start=%d, got=%d", startCursor, got.parade.Cursor)
	}
}

// ---------------------------------------------------------------------------
// 17. executePaletteAction returns cmd for valid action
// ---------------------------------------------------------------------------

func TestQuickActionReturnsCmd(t *testing.T) {
	got := setupModel(t)

	// Verify we have a selected issue
	if got.parade.SelectedIssue == nil {
		t.Fatal("expected a selected issue in the parade")
	}

	_, cmd := got.executePaletteAction(components.ActionSetInProgress)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from executePaletteAction(ActionSetInProgress) with selected issue")
	}
}

// ---------------------------------------------------------------------------
// 18. quickAction with no issues is a no-op
// ---------------------------------------------------------------------------

func TestQuickActionNilIssueNoop(t *testing.T) {
	m := New([]data.Issue{}, data.Source{}, data.DefaultBlockingTypes)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	got := model.(Model)

	_, cmd := got.quickAction(data.StatusInProgress, "in_progress")
	if cmd != nil {
		t.Fatal("expected nil cmd from quickAction with no issues")
	}
}

// ---------------------------------------------------------------------------
// 19. closeSelectedIssue returns cmd for open issue
// ---------------------------------------------------------------------------

func TestCloseSelectedIssueReturnsCmd(t *testing.T) {
	got := setupModel(t)

	if got.parade.SelectedIssue == nil {
		t.Fatal("expected a selected issue")
	}
	if got.parade.SelectedIssue.Status == data.StatusClosed {
		t.Fatal("expected selected issue to not already be closed")
	}

	_, cmd := got.closeSelectedIssue()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from closeSelectedIssue with open issue selected")
	}
}

// ---------------------------------------------------------------------------
// 20. setPriority returns cmd when priority differs
// ---------------------------------------------------------------------------

func TestSetPriorityReturnsCmd(t *testing.T) {
	got := setupModel(t)

	if got.parade.SelectedIssue == nil {
		t.Fatal("expected a selected issue")
	}
	// testIssue sets PriorityMedium, so setting High should produce a cmd
	if got.parade.SelectedIssue.Priority != data.PriorityMedium {
		t.Fatalf("expected selected issue priority to be PriorityMedium, got %d", got.parade.SelectedIssue.Priority)
	}

	_, cmd := got.setPriority(data.PriorityHigh)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from setPriority(PriorityHigh) with medium priority issue")
	}
}

// ---------------------------------------------------------------------------
// 21. 's' key with Gas Town spawns formula list fetch
// ---------------------------------------------------------------------------

func TestKeySGasTownFetchesFormulas(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true

	model, cmd := got.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	got = model.(Model)

	if cmd == nil {
		t.Fatal("expected non-nil cmd from pressing s with Gas Town available")
	}
	// The formulaTarget should be set to the selected issue ID
	if got.formulaTarget == "" {
		t.Fatal("expected formulaTarget to be set after pressing s")
	}
}

// ---------------------------------------------------------------------------
// 22. 's' key without Gas Town is a no-op
// ---------------------------------------------------------------------------

func TestKeySNoGasTownNoop(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = false

	_, cmd := got.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if cmd != nil {
		t.Fatal("expected nil cmd from pressing s without Gas Town")
	}
}

// ---------------------------------------------------------------------------
// 23. 'n' key opens nudge input when agent active
// ---------------------------------------------------------------------------

func TestKeyNOpensNudgeInput(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true
	issueID := got.parade.SelectedIssue.ID
	got.activeAgents[issueID] = "Toast"

	model, cmd := got.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	got = model.(Model)

	if !got.nudging {
		t.Fatal("expected nudging to be true after pressing n with active agent")
	}
	if got.nudgeTarget != "Toast" {
		t.Fatalf("expected nudgeTarget 'Toast', got %q", got.nudgeTarget)
	}
	if cmd == nil {
		t.Fatal("expected blink cmd from nudge input")
	}
}

// ---------------------------------------------------------------------------
// 24. 'n' key no-op when no active agent
// ---------------------------------------------------------------------------

func TestKeyNNoActiveAgentNoop(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true

	model, cmd := got.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	got = model.(Model)

	if got.nudging {
		t.Fatal("expected nudging to be false when no active agent")
	}
	if cmd != nil {
		t.Fatal("expected nil cmd when no active agent for nudge")
	}
}

// ---------------------------------------------------------------------------
// 25. 'A' key with Gas Town dispatches unsling
// ---------------------------------------------------------------------------

func TestKeyAGasTownUnsling(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true
	issueID := got.parade.SelectedIssue.ID
	got.activeAgents[issueID] = "Toast"

	_, cmd := got.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from pressing A with Gas Town and active agent")
	}
	// Execute the cmd and verify it returns unslingResultMsg
	msg := cmd()
	if _, ok := msg.(unslingResultMsg); !ok {
		t.Fatalf("expected unslingResultMsg, got %T", msg)
	}
}

// ---------------------------------------------------------------------------
// 26. 'A' key no-op when no active agent
// ---------------------------------------------------------------------------

func TestKeyANoActiveAgentNoop(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true

	_, cmd := got.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	if cmd != nil {
		t.Fatal("expected nil cmd when no active agent for A key")
	}
}

// ---------------------------------------------------------------------------
// 27. formulaListMsg opens formula palette
// ---------------------------------------------------------------------------

func TestFormulaListMsgOpensPalette(t *testing.T) {
	got := setupModel(t)
	got.formulaTarget = "open-1"

	model, cmd := got.Update(formulaListMsg{formulas: []string{"shiny", "basic"}, err: nil})
	got = model.(Model)

	if !got.formulaPicking {
		t.Fatal("expected formulaPicking to be true")
	}
	if !got.showPalette {
		t.Fatal("expected showPalette to be true")
	}
	if cmd == nil {
		t.Fatal("expected palette init cmd")
	}
}

// ---------------------------------------------------------------------------
// 28. formulaListMsg with empty formulas falls back to plain sling
// ---------------------------------------------------------------------------

func TestFormulaListMsgEmptyFallback(t *testing.T) {
	got := setupModel(t)
	got.formulaTarget = "open-1"

	model, cmd := got.Update(formulaListMsg{formulas: nil, err: nil})
	got = model.(Model)

	if got.formulaPicking {
		t.Fatal("expected formulaPicking to be false on fallback")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (toast + sling) on fallback")
	}
}

// ---------------------------------------------------------------------------
// 29. multi-sling 'a' key with Gas Town and selection
// ---------------------------------------------------------------------------

func TestKeyAMultiSlingWithSelection(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true

	// Select current item
	model, _ := got.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	got = model.(Model)
	if got.parade.SelectionCount() == 0 {
		t.Fatal("expected items to be selected")
	}

	model, cmd := got.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	got = model.(Model)

	if cmd == nil {
		t.Fatal("expected non-nil cmd from multi-sling")
	}
	// Selection should be cleared
	if got.parade.SelectionCount() != 0 {
		t.Fatal("expected selection to be cleared after multi-sling")
	}
}

// TestKeyASlingsWithoutLocalRuntime pins the fix for an asymmetry in the `a`
// handler: the !agentAvail guard sat ahead of the Gas Town sling branch, so on
// a box with gt but no claude/cursor-agent/codex installed, `a` on a SINGLE
// issue did nothing while `a` on a multi-selection dispatched fine. Only direct
// launch needs a local runtime — the orchestrator starts the agent itself.
func TestKeyASlingsWithoutLocalRuntime(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = true
	got.agentAvail = false // no claude/cursor-agent/codex on PATH
	got.inTmux = false

	_, cmd := got.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if cmd == nil {
		t.Fatal("pressing a on a single issue produced no command; sling does not need a local runtime")
	}
}

// TestKeyADirectLaunchStillNeedsRuntime keeps the guard where it belongs: with
// no orchestrator, `a` execs the agent binary itself, so a missing runtime must
// still be a no-op rather than a failed exec.
func TestKeyADirectLaunchStillNeedsRuntime(t *testing.T) {
	got := setupModel(t)
	got.gtEnv.Available = false
	got.driver = gastown.NewGTDriver() // Backend() == "gastown", so no orchestrator
	got.agentAvail = false
	got.inTmux = false

	_, cmd := got.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if cmd != nil {
		t.Error("expected no command: direct launch needs an agent runtime on PATH")
	}
}

// TestKeyASingleAndMultiAgreeWithoutRuntime is the regression in its plainest
// form — the two selection modes must not disagree about whether `a` works.
func TestKeyASingleAndMultiAgreeWithoutRuntime(t *testing.T) {
	single := setupModel(t)
	single.gtEnv.Available = true
	single.agentAvail = false
	single.inTmux = false
	_, singleCmd := single.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})

	multi := setupModel(t)
	multi.gtEnv.Available = true
	multi.agentAvail = false
	multi.inTmux = false
	multi.parade.ToggleSelect() // select the issue under the cursor
	_, multiCmd := multi.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})

	if (singleCmd == nil) != (multiCmd == nil) {
		t.Errorf("single and multi-select disagree: singleCmd nil=%v, multiCmd nil=%v",
			singleCmd == nil, multiCmd == nil)
	}
}

// ---------------------------------------------------------------------------
// 30. E scopes the parade to the selected issue's epic subtree
// ---------------------------------------------------------------------------

// setupEpicScopeModel builds the three-issue graph the epic-scope tests need: an
// epic, one child carrying a parent-child edge to it, and an unrelated root. The
// cursor starts on the child, so E has an epic ancestor reachable through the
// edge — not through the dotted ID.
func setupEpicScopeModel(t *testing.T) Model {
	t.Helper()
	epic := testIssue("epic", data.StatusInProgress)
	epic.IssueType = data.TypeEpic
	child := testIssue("epic.1", data.StatusOpen)
	child.Dependencies = []data.Dependency{{IssueID: "epic.1", DependsOnID: "epic", Type: "parent-child"}}
	grandchild := testIssue("epic.1.1", data.StatusOpen)
	grandchild.Dependencies = []data.Dependency{{IssueID: "epic.1.1", DependsOnID: "epic.1", Type: "parent-child"}}
	unrelated := testIssue("unrelated", data.StatusOpen)

	m := New([]data.Issue{epic, child, grandchild, unrelated}, data.Source{}, data.DefaultBlockingTypes)
	m.startedAt = time.Now().Add(-time.Second)
	m.driver = gastown.NewGTDriver()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	got := model.(Model)

	for i, item := range got.parade.Items {
		if item.Issue != nil && item.Issue.ID == "epic.1" {
			got.parade.Cursor = i
			got.parade.SelectedIssue = item.Issue
		}
	}
	if got.parade.SelectedIssue == nil || got.parade.SelectedIssue.ID != "epic.1" {
		t.Fatalf("setup: expected cursor on epic.1, got %+v", got.parade.SelectedIssue)
	}
	return got
}

// visibleParadeIDs returns the IDs of the issue rows currently rendered.
func visibleParadeIDs(m Model) map[string]bool {
	ids := make(map[string]bool)
	for _, item := range m.parade.Items {
		if item.Issue != nil {
			ids[item.Issue.ID] = true
		}
	}
	return ids
}
func TestKeyGreaterTogglesSelectedTreeNode(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	got = model.(Model)

	if !got.parade.Collapsed["epic.1"] {
		t.Fatalf("expected selected child epic.1 to be collapsed, got %v", got.parade.Collapsed)
	}
	visible := visibleParadeIDs(got)
	if visible["epic.1.1"] {
		t.Fatalf("expected collapsed grandchild to be hidden, got visible IDs %v", visible)
	}
	if !visible["epic"] || !visible["epic.1"] || !visible["unrelated"] {
		t.Fatalf("expected epic, child, and unrelated root to remain visible, got %v", visible)
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	got = model.(Model)
	if got.parade.Collapsed["epic.1"] {
		t.Fatalf("expected second > to expand epic.1, got %v", got.parade.Collapsed)
	}
	if !visibleParadeIDs(got)["epic.1.1"] {
		t.Fatalf("expected grandchild to return after expansion, got %v", visibleParadeIDs(got))
	}
}

func TestKeyEScopesToEpicSubtree(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)

	if got.scopeRootID != "epic" {
		t.Fatalf("expected scopeRootID %q after E, got %q", "epic", got.scopeRootID)
	}
	visible := visibleParadeIDs(got)
	if !visible["epic"] {
		t.Error("expected the epic itself to stay visible under its own scope")
	}
	if !visible["epic.1"] {
		t.Error("expected the epic's child to stay visible under its own scope")
	}
	if visible["unrelated"] {
		t.Error("expected the unrelated issue to be scoped out")
	}
}

func TestKeyEscClearsEpicScope(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if visibleParadeIDs(got)["unrelated"] {
		t.Fatal("precondition: unrelated issue should be scoped out after E")
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = model.(Model)

	if got.scopeRootID != "" {
		t.Fatalf("expected scopeRootID cleared after esc, got %q", got.scopeRootID)
	}
	if !visibleParadeIDs(got)["unrelated"] {
		t.Error("expected the unrelated issue restored after esc")
	}
}

func TestKeyEScopeRequiresEpicAncestor(t *testing.T) {
	got := setupModel(t) // plain tasks: no epic anywhere in the graph
	before := visibleParadeIDs(got)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)

	if got.scopeRootID != "" {
		t.Fatalf("expected no scope for an issue with no epic ancestor, got %q", got.scopeRootID)
	}
	after := visibleParadeIDs(got)
	if len(after) != len(before) {
		t.Fatalf("expected the visible set unchanged, before=%v after=%v", before, after)
	}
	for id := range before {
		if !after[id] {
			t.Fatalf("expected issue %s to stay visible, got %v", id, after)
		}
	}
}

// TestKeyEscStillClearsScopeBeforeTreeFocus pins the esc ordering: leaving a
// scope is one press and must not also drop focus mode or tree state.
func TestKeyEscStillClearsScopeBeforeTreeFocus(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	got = model.(Model)
	if !got.parade.Collapsed["epic.1"] {
		t.Fatal("precondition: expected epic.1 to be collapsed")
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	got = model.(Model)
	model, _ = got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if got.scopeRootID != "epic" || !got.focusMode {
		t.Fatalf("precondition: expected scope %q and focusMode true, got %q/%v",
			"epic", got.scopeRootID, got.focusMode)
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = model.(Model)

	if got.scopeRootID != "" {
		t.Errorf("expected esc to clear the scope, got %q", got.scopeRootID)
	}
	if !got.focusMode {
		t.Error("expected focus mode to survive the press that exits a scope")
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = model.(Model)
	if got.focusMode {
		t.Error("expected a second esc to clear focus mode")
	}
}

// TestKeyEScopeStaleRootEmptiesParade pins the load-bearing nil from
// data.ScopeToSubtree: when the scoped epic leaves the loaded set, the scope
// root is unknown and the parade must go EMPTY rather than silently widen back
// to every issue.
func TestKeyEScopeStaleRootEmptiesParade(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if got.scopeRootID != "epic" {
		t.Fatalf("precondition: expected scope set to epic, got %q", got.scopeRootID)
	}

	model, _ = got.Update(data.FileChangedMsg{Issues: []data.Issue{testIssue("unrelated", data.StatusOpen)}})
	got = model.(Model)

	if n := got.parade.VisibleIssues(); n != 0 {
		t.Fatalf("expected an empty parade for a stale scope root, got %d visible issue(s)", n)
	}
}

// ---------------------------------------------------------------------------
// 31. the scope survives a resize — header and parade keep one narrowing pipeline
// ---------------------------------------------------------------------------

// headerIssueIDs collects every issue ID the header currently tallies, across
// all six semantic states. The header is the operator's board-level tally, so
// it must describe the same set the parade shows.
func headerIssueIDs(m Model) map[string]bool {
	ids := make(map[string]bool)
	for _, state := range data.StateOrder() {
		for _, issue := range m.header.Groups[state] {
			ids[issue.ID] = true
		}
	}
	return ids
}

// TestScopeSurvivesResize pins the single narrowing pipeline behind the header
// and the parade. layout() runs on every WindowSizeMsg (and on every layout
// preset switch), so a resize with a scope lit must not re-tally the full board
// into the header while the parade stays scoped — that would light the SCOPE
// chip beside counts for issues the scope excludes.
func TestScopeSurvivesResize(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if got.scopeRootID != "epic" {
		t.Fatalf("precondition: expected scope set to epic, got %q", got.scopeRootID)
	}

	model, _ = got.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	got = model.(Model)

	visible := visibleParadeIDs(got)
	if visible["unrelated"] {
		t.Errorf("expected the unrelated issue to stay scoped out after resize, got %v", visible)
	}
	if !visible["epic"] || !visible["epic.1"] {
		t.Errorf("expected the epic subtree to stay visible after resize, got %v", visible)
	}

	tallied := headerIssueIDs(got)
	if tallied["unrelated"] {
		t.Errorf("expected the header to keep tallying only the scoped set, but it counted the unrelated issue: %v", tallied)
	}
	if len(tallied) != len(visible) {
		t.Errorf("header and parade disagree after resize: header tallies %v, parade shows %v", tallied, visible)
	}
	if n := len(got.header.Groups[data.StateReady]); n != 2 {
		t.Errorf("expected the header to count 2 Ready issues under the scope (epic.1 and epic.1.1), got %d", n)
	}
}

// TestStaleScopeStaysEmptyAcrossResize carries the stale-scope state from
// TestKeyEScopeStaleRootEmptiesParade through a resize. The spec forbids
// silently widening back to everything, so an empty scoped parade must stay
// empty when layout() runs.
func TestStaleScopeStaysEmptyAcrossResize(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	model, _ = got.Update(data.FileChangedMsg{Issues: []data.Issue{testIssue("unrelated", data.StatusOpen)}})
	got = model.(Model)
	if n := got.parade.VisibleIssues(); n != 0 {
		t.Fatalf("precondition: expected an empty parade for a stale scope root, got %d visible issue(s)", n)
	}

	model, _ = got.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	got = model.(Model)

	if n := got.parade.VisibleIssues(); n != 0 {
		t.Fatalf("expected the stale scope to stay empty across a resize, got %d visible issue(s): %v",
			n, visibleParadeIDs(got))
	}
	if visibleParadeIDs(got)["unrelated"] {
		t.Error("expected a resize not to widen a stale scope back to the whole board")
	}
	if got.scopeRootID != "epic" {
		t.Errorf("expected the stale scope root to stay pinned, got %q", got.scopeRootID)
	}
	if tallied := headerIssueIDs(got); tallied["unrelated"] {
		t.Errorf("expected the header to stay empty with the parade, got %v", tallied)
	}
}

// ---------------------------------------------------------------------------
// 32. the scope's lifecycle: reload, `c`, and esc against a committed filter
// ---------------------------------------------------------------------------

// TestKeyEScopeReloadsWhenRootReturns pins the reload half of the scope's
// lifecycle. TestKeyEScopeStaleRootEmptiesParade proves the root's departure
// empties the parade; nothing proved the scope recovers. A refresh is the
// normal way a root comes back — a `git pull`, a `br sync`, another agent
// closing a blocker — and the scope is an ID, so every reload must re-narrow to
// it instead of staying empty forever or widening to the board.
func TestKeyEScopeReloadsWhenRootReturns(t *testing.T) {
	epic := testIssue("epic", data.StatusInProgress)
	epic.IssueType = data.TypeEpic
	child := testIssue("epic.1", data.StatusOpen)
	child.Dependencies = []data.Dependency{{IssueID: "epic.1", DependsOnID: "epic", Type: "parent-child"}}
	full := []data.Issue{epic, child, testIssue("unrelated", data.StatusOpen)}

	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if got.scopeRootID != "epic" {
		t.Fatalf("precondition: expected scope set to epic, got %q", got.scopeRootID)
	}

	// assertScoped is the whole claim: the scope is still lit, its subtree is
	// on screen, and nothing outside it leaked in — to the parade or the header.
	assertScoped := func(stage string) {
		t.Helper()
		if got.scopeRootID != "epic" {
			t.Fatalf("%s: expected the scope root to survive, got %q", stage, got.scopeRootID)
		}
		visible := visibleParadeIDs(got)
		if !visible["epic"] || !visible["epic.1"] {
			t.Errorf("%s: expected the epic subtree visible, got %v", stage, visible)
		}
		if visible["unrelated"] {
			t.Errorf("%s: expected the reload to stay scoped, got %v", stage, visible)
		}
		if tallied := headerIssueIDs(got); tallied["unrelated"] {
			t.Errorf("%s: expected the header to tally only the scoped set, got %v", stage, tallied)
		}
	}

	// A plain refresh — the board reloads, same root — stays narrowed.
	model, _ = got.Update(data.FileChangedMsg{Issues: full})
	got = model.(Model)
	assertScoped("plain refresh")

	// Root leaves the loaded set: the scope empties rather than widening.
	model, _ = got.Update(data.FileChangedMsg{Issues: []data.Issue{testIssue("unrelated", data.StatusOpen)}})
	got = model.(Model)
	if n := got.parade.VisibleIssues(); n != 0 {
		t.Fatalf("expected an empty parade for a stale scope root, got %d visible issue(s)", n)
	}

	// Same scope ID, and the reload brings the root and its child back.
	model, _ = got.Update(data.FileChangedMsg{Issues: full})
	got = model.(Model)
	assertScoped("reload after the root returned")
}

func TestCollapseSurvivesEpicScopeRebuild(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	got = model.(Model)
	if !got.parade.Collapsed["epic.1"] {
		t.Fatal("precondition: expected epic.1 to be collapsed")
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if got.scopeRootID != "epic" {
		t.Fatalf("expected epic scope, got %q", got.scopeRootID)
	}
	if !got.parade.Collapsed["epic.1"] {
		t.Fatalf("expected collapse to survive epic scope rebuild, got %v", got.parade.Collapsed)
	}
	if visibleParadeIDs(got)["epic.1.1"] {
		t.Fatalf("expected collapsed grandchild to stay hidden after scope rebuild, got %v", visibleParadeIDs(got))
	}
}

// TestKeyEscWithCommittedFilterClearsScopeFirst pins esc's precedence once a
// filter query is committed but the input is no longer focused. The README
// promises esc clears the scope first and that a scope stacks on top of the
// filter instead of resetting it, so the press must drop the scope and leave
// the query — and the parade — under the filter alone. Only while the input
// itself has focus is esc the input's own key.
func TestKeyEscWithCommittedFilterClearsScopeFirst(t *testing.T) {
	got := setupEpicScopeModel(t)

	model, _ := got.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got = model.(Model)
	if got.scopeRootID != "epic" {
		t.Fatalf("precondition: expected scope set to epic, got %q", got.scopeRootID)
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	got = model.(Model)
	for _, r := range "unrelated" {
		model, _ = got.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		got = model.(Model)
	}
	model, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = model.(Model)

	if got.filtering {
		t.Fatal("precondition: enter should have released the filter input")
	}
	if got.filterInput.Value() != "unrelated" {
		t.Fatalf("precondition: expected the committed query %q, got %q", "unrelated", got.filterInput.Value())
	}
	// The scope and the filter intersect: nothing is left to show.
	if n := got.parade.VisibleIssues(); n != 0 {
		t.Fatalf("precondition: expected scope ∩ filter to be empty, got %d visible issue(s): %v",
			n, visibleParadeIDs(got))
	}

	model, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = model.(Model)

	if got.scopeRootID != "" {
		t.Errorf("expected esc to clear the scope, got %q", got.scopeRootID)
	}
	if got.filterInput.Value() != "unrelated" {
		t.Errorf("expected the committed filter to survive the esc, got %q", got.filterInput.Value())
	}
	visible := visibleParadeIDs(got)
	if !visible["unrelated"] {
		t.Errorf("expected the filter to still narrow the parade after esc, got %v", visible)
	}
	if visible["epic"] || visible["epic.1"] {
		t.Errorf("expected the scope's subtree to stay filtered out, got %v", visible)
	}
}

func setupMouseModel(t *testing.T) Model {
	t.Helper()
	issues := make([]data.Issue, 14)
	for i := range issues {
		issues[i] = testIssue(fmt.Sprintf("mouse-%02d", i+1), data.StatusOpen)
		issues[i].Description = strings.Repeat("detail line\n", 30)
	}
	m := New(issues, data.Source{}, data.DefaultBlockingTypes)
	m.startedAt = time.Now().Add(-time.Second)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	return model.(Model)
}

func TestMouseClickParadeSelectsScrolledRowAndFocusesParade(t *testing.T) {
	m := setupMouseModel(t)
	m.parade.ScrollOffset = 3
	target := m.parade.IssueAtViewportRow(1)
	if target == nil {
		t.Fatal("precondition: expected a selectable scrolled parade row")
	}
	m.activPane = PaneDetail
	m.detail.Focused = true

	model, _ := m.Update(tea.MouseClickMsg{X: 1, Y: headerHeight + 1, Button: tea.MouseLeft})
	got := model.(Model)
	if got.parade.SelectedIssue == nil || got.parade.SelectedIssue.ID != target.ID {
		t.Fatalf("selected issue = %v, want %s", got.parade.SelectedIssue, target.ID)
	}
	if got.activPane != PaneParade || got.detail.Focused {
		t.Fatalf("mouse parade click focus = pane %d/detail %v, want parade/false", got.activPane, got.detail.Focused)
	}
}

func TestMouseClickDetailReferenceNavigatesAndFocusesDetail(t *testing.T) {
	source := testIssue("mouse-source", data.StatusOpen)
	source.Dependencies = []data.Dependency{{IssueID: source.ID, DependsOnID: "mouse-dep", Type: "blocks"}}
	dep := testIssue("mouse-dep", data.StatusOpen)
	m := New([]data.Issue{source, dep}, data.Source{}, data.DefaultBlockingTypes)
	m.startedAt = time.Now().Add(-time.Second)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = model.(Model)
	if !m.restoreParadeSelection(source.ID) {
		t.Fatal("precondition: source issue was not found in parade")
	}
	m.syncSelection()
	dependencyRow := -1
	for row := 0; row < m.detail.Viewport.Height(); row++ {
		if got := m.detail.ReferenceAt(row); got != nil && got.ID == dep.ID {
			dependencyRow = row
			break
		}
	}
	if dependencyRow < 0 {
		t.Fatalf("precondition: dependency row for %s was not visible", dep.ID)
	}

	model, _ = m.Update(tea.MouseClickMsg{X: m.parade.Width + 1, Y: headerHeight + dependencyRow, Button: tea.MouseRight})
	got := model.(Model)
	if got.detail.Issue == nil || got.detail.Issue.ID != dep.ID {
		t.Fatalf("detail issue = %v, want %s", got.detail.Issue, dep.ID)
	}
	if got.parade.SelectedIssue == nil || got.parade.SelectedIssue.ID != dep.ID {
		t.Fatalf("parade selection = %v, want %s", got.parade.SelectedIssue, dep.ID)
	}
	if got.activPane != PaneDetail || !got.detail.Focused {
		t.Fatalf("mouse detail click focus = pane %d/detail %v, want detail/true", got.activPane, got.detail.Focused)
	}
}

func TestMouseWheelScrollsPaneUnderPointerWithoutStealingFocus(t *testing.T) {
	m := setupMouseModel(t)
	m.activPane = PaneDetail
	m.detail.Focused = true
	for i := min(m.parade.Height-1, len(m.parade.Items)-1); i >= 0; i-- {
		if m.parade.Items[i].Issue != nil {
			m.parade.Cursor = i
			m.parade.SelectedIssue = m.parade.Items[i].Issue
			break
		}
	}
	m.syncSelection()
	paradeOffset := m.parade.ScrollOffset
	model, _ := m.Update(tea.MouseWheelMsg{X: 1, Y: headerHeight + 1, Button: tea.MouseWheelDown})
	m = model.(Model)
	if m.parade.ScrollOffset <= paradeOffset {
		t.Fatalf("parade offset = %d, want > %d", m.parade.ScrollOffset, paradeOffset)
	}
	if m.activPane != PaneDetail || !m.detail.Focused {
		t.Fatalf("left wheel stole focus: pane %d/detail %v", m.activPane, m.detail.Focused)
	}

	detailOffset := m.detail.Viewport.YOffset()
	model, _ = m.Update(tea.MouseWheelMsg{X: m.parade.Width + 1, Y: headerHeight + 1, Button: tea.MouseWheelDown})
	m = model.(Model)
	if m.detail.Viewport.YOffset() <= detailOffset {
		t.Fatalf("detail offset = %d, want > %d", m.detail.Viewport.YOffset(), detailOffset)
	}
	if m.activPane != PaneDetail || !m.detail.Focused {
		t.Fatalf("right wheel changed focus: pane %d/detail %v", m.activPane, m.detail.Focused)
	}
}

func TestMouseIgnoresHeaderFooterPaddingAndScrollCue(t *testing.T) {
	m := setupMouseModel(t)
	selected := m.parade.SelectedIssue.ID
	offset := m.parade.ScrollOffset
	clicks := []tea.MouseClickMsg{
		{X: 1, Y: 0, Button: tea.MouseLeft},
		{X: 1, Y: m.height - 1, Button: tea.MouseLeft},
		{X: m.width - 1, Y: headerHeight + m.detail.Viewport.Height(), Button: tea.MouseRight},
	}
	for _, click := range clicks {
		model, _ := m.Update(click)
		m = model.(Model)
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != selected || m.parade.ScrollOffset != offset {
		t.Fatalf("ignored-area click changed parade selection/offset: %v/%d", m.parade.SelectedIssue, m.parade.ScrollOffset)
	}
}

func TestMouseDoesNotLeakThroughHelpOrForms(t *testing.T) {
	m := setupMouseModel(t)
	selected := m.parade.SelectedIssue.ID
	m.showHelp = true
	model, _ := m.Update(tea.MouseClickMsg{X: 1, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != selected {
		t.Fatalf("help click leaked to parade: %v", m.parade.SelectedIssue)
	}
	m.showHelp = false
	m.creating = true
	model, _ = m.Update(tea.MouseClickMsg{X: 1, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != selected {
		t.Fatalf("form click leaked to parade: %v", m.parade.SelectedIssue)
	}
}

func TestDividerDragResizesPanes(t *testing.T) {
	m := setupMouseModel(t)
	divider := m.parade.Width
	if divider != m.width*2/5 {
		t.Fatalf("precondition: parade width = %d, want the default %d", divider, m.width*2/5)
	}

	model, _ := m.Update(tea.MouseClickMsg{X: divider, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if !m.draggingDivider {
		t.Fatal("a left press on the divider column must start a drag")
	}

	model, _ = m.Update(tea.MouseMotionMsg{X: 62, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.parade.Width != 62 || m.detail.Width != m.width-62 {
		t.Fatalf("after drag parade/detail = %d/%d, want 62/%d", m.parade.Width, m.detail.Width, m.width-62)
	}

	model, _ = m.Update(tea.MouseReleaseMsg{X: 62, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.draggingDivider {
		t.Fatal("release must end the drag")
	}

	m.rebuildParade()
	if m.parade.Width != 62 {
		t.Fatalf("dragged width = %d after rebuild, want it to persist for the session", m.parade.Width)
	}
}

func TestDividerDragClampsToPaneFloors(t *testing.T) {
	m := setupMouseModel(t)
	m.draggingDivider = true

	model, _ := m.Update(tea.MouseMotionMsg{X: 2, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.parade.Width != minPaneWidth {
		t.Fatalf("left clamp = %d, want %d", m.parade.Width, minPaneWidth)
	}

	m.draggingDivider = true
	model, _ = m.Update(tea.MouseMotionMsg{X: m.width - 1, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.parade.Width != m.width-minPaneWidth {
		t.Fatalf("right clamp = %d, want %d", m.parade.Width, m.width-minPaneWidth)
	}
	if m.detail.Width < minPaneWidth {
		t.Fatalf("detail pane width = %d, want at least %d", m.detail.Width, minPaneWidth)
	}
}

func TestDividerPressDoesNotChangeParadeSelection(t *testing.T) {
	m := setupMouseModel(t)
	selected := m.parade.SelectedIssue.ID
	model, _ := m.Update(tea.MouseClickMsg{X: m.parade.Width, Y: headerHeight + 1, Button: tea.MouseLeft})
	m = model.(Model)
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != selected {
		t.Fatalf("divider press changed selection to %v, want %s", m.parade.SelectedIssue, selected)
	}
}

func TestSortKeyCyclesParadeSortMode(t *testing.T) {
	m := setupMouseModel(t)
	if m.parade.SortMode != views.SortAttention {
		t.Fatalf("startup sort mode = %v, want SortAttention", m.parade.SortMode)
	}

	model, _ := m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	m = model.(Model)
	if m.parade.SortMode != views.SortPriority {
		t.Fatalf("after S sort mode = %v, want SortPriority", m.parade.SortMode)
	}

	m.rebuildParade()
	if m.parade.SortMode != views.SortPriority {
		t.Fatal("sort mode must survive a parade rebuild")
	}

	model, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	m = model.(Model)
	if m.parade.SortMode != views.SortAttention {
		t.Fatalf("second S sort mode = %v, want SortAttention", m.parade.SortMode)
	}

	var found bool
	for _, cmd := range m.buildPaletteCommands() {
		if cmd.Action == components.ActionCycleSort {
			found = true
		}
	}
	if !found {
		t.Fatal("command palette must offer the sort toggle")
	}
}

func setupParentChildMouseModel(t *testing.T) Model {
	t.Helper()
	issues := []data.Issue{
		{ID: "epic", Title: "Epic", Status: data.StatusOpen, Priority: 0, IssueType: data.TypeEpic},
		{ID: "epic.1", Title: "Child one", Status: data.StatusOpen, Priority: 0,
			Dependencies: []data.Dependency{{IssueID: "epic.1", DependsOnID: "epic", Type: "parent-child"}}},
		{ID: "other", Title: "Other root", Status: data.StatusOpen, Priority: 1, IssueType: data.TypeTask},
	}
	m := New(issues, data.Source{}, data.DefaultBlockingTypes)
	m.startedAt = time.Now().Add(-time.Second)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return model.(Model)
}

// paradeViewportRowOf resolves an issue's visible row.
func paradeViewportRowOf(t *testing.T, m Model, id string) int {
	t.Helper()
	for row := 0; row < m.parade.Height; row++ {
		if iss := m.parade.IssueAtViewportRow(row); iss != nil && iss.ID == id {
			return row
		}
	}
	t.Fatalf("row for %s is not visible", id)
	return -1
}

// titleClick clicks well right of the glyph column, on the title, so the
// gutter branch never claims it.
func titleClick(row int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: 20, Y: headerHeight + row, Button: tea.MouseLeft}
}

func TestMouseGutterClickTogglesWithoutSelecting(t *testing.T) {
	m := setupParentChildMouseModel(t)
	if !m.restoreParadeSelection("other") {
		t.Fatal("precondition: other root not found")
	}
	m.syncSelection()
	beforeCursor := m.parade.Cursor
	gutter := tea.MouseClickMsg{X: 0, Y: headerHeight + paradeViewportRowOf(t, m, "epic"), Button: tea.MouseLeft}

	model, _ := m.Update(gutter)
	m = model.(Model)
	if !m.parade.Collapsed["epic"] {
		t.Fatal("a gutter click on a parent must collapse it")
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "other" {
		t.Fatalf("gutter click moved selection to %v, want other untouched", m.parade.SelectedIssue)
	}
	if m.parade.Cursor != beforeCursor {
		t.Fatalf("gutter click moved the cursor to %d, want %d", m.parade.Cursor, beforeCursor)
	}

	model, _ = m.Update(gutter)
	m = model.(Model)
	if m.parade.Collapsed["epic"] {
		t.Fatal("a second gutter click must expand the parent again")
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "other" {
		t.Fatalf("second gutter click moved selection to %v, want other", m.parade.SelectedIssue)
	}
}

func TestMouseTitleClickSelectsWithoutToggling(t *testing.T) {
	m := setupParentChildMouseModel(t)
	m.parade.ToggleNode("epic")
	if !m.parade.Collapsed["epic"] {
		t.Fatal("precondition: epic should start collapsed")
	}

	model, _ := m.Update(tea.MouseClickMsg{X: 20, Y: headerHeight + paradeViewportRowOf(t, m, "epic"), Button: tea.MouseLeft})
	m = model.(Model)
	if !m.parade.Collapsed["epic"] {
		t.Fatal("a single click on the title must never expand — the old blanket auto-expand is gone")
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "epic" {
		t.Fatalf("title click selection = %v, want epic", m.parade.SelectedIssue)
	}
	if m.activPane != PaneParade || m.detail.Focused {
		t.Fatalf("title click focus = pane %d/detail %v, want parade/false", m.activPane, m.detail.Focused)
	}
}

func TestKeyboardGreaterThanStillToggles(t *testing.T) {
	m := setupParentChildMouseModel(t)
	if !m.restoreParadeSelection("epic") {
		t.Fatal("precondition: epic row not found")
	}
	m.syncSelection()
	visible := func(m Model, id string) bool {
		for _, item := range m.parade.Items {
			if item.Issue != nil && item.Issue.ID == id {
				return true
			}
		}
		return false
	}

	model, _ := m.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	m = model.(Model)
	if visible(m, "epic.1") {
		t.Fatal("keyboard > must still collapse the selected branch")
	}
	model, _ = m.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	m = model.(Model)
	if !visible(m, "epic.1") {
		t.Fatal("keyboard > must still expand the selected branch")
	}
}

func TestMouseDoubleClickExpandsAndSelectsUnselectedParent(t *testing.T) {
	m := setupParentChildMouseModel(t)
	m.parade.ToggleNode("epic")
	if !m.parade.Collapsed["epic"] {
		t.Fatal("precondition: epic should start collapsed")
	}
	if !m.restoreParadeSelection("other") {
		t.Fatal("precondition: other root not found")
	}
	m.syncSelection()
	click := titleClick(paradeViewportRowOf(t, m, "epic"))

	model, _ := m.Update(click)
	m = model.(Model)
	if m.parade.Collapsed["epic"] {
		// First of the pair is an ordinary selection click.
		if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "epic" {
			t.Fatalf("single click selection = %v, want epic", m.parade.SelectedIssue)
		}
	}

	model, _ = m.Update(click)
	m = model.(Model)
	if m.parade.Collapsed["epic"] {
		t.Fatal("double click on a collapsed parent must expand it")
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "epic" {
		t.Fatalf("double click selection = %v, want epic selected too", m.parade.SelectedIssue)
	}
}

func TestMouseDoubleClickCollapsesExpandedSelectedParent(t *testing.T) {
	m := setupParentChildMouseModel(t)
	if !m.restoreParadeSelection("epic") {
		t.Fatal("precondition: epic row not found")
	}
	m.syncSelection()
	if m.parade.Collapsed["epic"] {
		t.Fatal("precondition: epic should start expanded")
	}
	if m.detail.Issue == nil || m.detail.Issue.ID != "epic" {
		t.Fatalf("precondition: detail pane should show epic, got %v", m.detail.Issue)
	}
	click := titleClick(paradeViewportRowOf(t, m, "epic"))

	model, _ := m.Update(click)
	m = model.(Model)
	model, _ = m.Update(click)
	m = model.(Model)

	if !m.parade.Collapsed["epic"] {
		t.Fatal("double click on an already-expanded parent must collapse it")
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "epic" {
		t.Fatalf("selection after collapse = %v, want epic still selected", m.parade.SelectedIssue)
	}
}

func TestMouseSlowSecondClickDoesNotToggle(t *testing.T) {
	m := setupParentChildMouseModel(t)
	click := titleClick(paradeViewportRowOf(t, m, "epic"))

	model, _ := m.Update(click)
	m = model.(Model)
	m.lastClickAt = time.Now().Add(-2 * time.Second)
	model, _ = m.Update(click)
	m = model.(Model)
	if m.parade.Collapsed["epic"] {
		t.Fatal("clicks outside the double-click window must not toggle")
	}
}

func TestMouseDoubleClickOnLeafOnlySelects(t *testing.T) {
	m := setupParentChildMouseModel(t)
	click := titleClick(paradeViewportRowOf(t, m, "epic.1"))

	model, _ := m.Update(click)
	m = model.(Model)
	model, _ = m.Update(click)
	m = model.(Model)
	if m.parade.Collapsed["epic.1"] {
		t.Fatal("a leaf has nothing to collapse")
	}
	if m.parade.SelectedIssue == nil || m.parade.SelectedIssue.ID != "epic.1" {
		t.Fatalf("leaf double click selection = %v, want epic.1", m.parade.SelectedIssue)
	}
}
