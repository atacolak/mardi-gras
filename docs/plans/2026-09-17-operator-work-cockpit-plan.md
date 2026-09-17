# Operator Work Cockpit Implementation Plan

**Technical Spec:** `docs/plans/2026-09-17-operator-work-cockpit-spec.md`
**Design Brief:** `/home/sf/worlds/personal/designs/mardi-gras/operator-work-cockpit.md` (via spec; do not bypass)

> **For the project lead:** first `br where` in this repo. Use the existing campaign epic `mard-nfy`; do not create or mutate beads during planning. After the four operator decisions below are recorded in both documents, mint only the current executable frontier and dispatch one `builder` at a time on the existing `feat/operator-observability` shared tree with isolation off. Do not implement tasks inline or parallelize builders. Paste each task's full text and settled interfaces into its packet; do not make workers read this file. After builder evidence is complete, dispatch an isolated `verifier`; only after verifier pass and the lead's evidence gate, persist the task checkbox and close its task bead. Do not merge, push, install a PATH binary, close the campaign epic, or remove harness-owned worktrees.

**Goal:** Replace the six-section parade with a priority-sorted work tree, real Operator Review semantics, semantic row color, per-node collapse, and mouse-selectable panes while preserving keyboard and epic scope.

**Architecture:** `internal/data.DeriveState` remains the single semantic classifier and consumes the approved stored review representation. `internal/views.Parade` flattens an edge-derived cross-state forest plus only Waiting/Blocked and Deferred attention sections, while `Detail` records its existing bead-reference lines for hit testing. `internal/app` owns mouse geometry, pane focus, collapse persistence, and existing scope/filter composition.

**Tech stack:** Go 1.25, Bubble Tea v2.0.9, Bubbles viewport v2.2.1, Lip Gloss v2.0.6, existing Beads CLI/JSONL seams; no new dependencies.

---

## Execution gate and shared contract

This plan is file-exact and Task 1 is executable. Operator answers recorded 2026-09-17 (Ata: "yeah sounds good") in [Open questions](#open-questions-preserved-from-the-spec):

- stored representation: `review` in `.beads/policy.yaml`; ready group stays `open` only
- flip: explicit `br update --status review`; named shim `legacyConvergedEpicOperatorReview` for already-converged epics
- right-reference target set: loaded DEPENDENCIES + local cross-rig IDs only
- glyphs: Ready `○` Working `●` Waiting/Blocked `⊘` Deferred `⏸` Operator Review `◐` Done `✓`

Hover-scroll (same turn): wheel over a pane scrolls that pane and does **not** steal focus. Task 6 owns it. Builders do not reopen these.

All tasks run **serially on one shared tree** because isolation is off. Do not parallelize builders. Task ordering deliberately keeps the high-collision files single-owner at a time:

1. Task 1 owns `.beads/policy.yaml` plus `semantic.go` and the complete `Exec*` rename.
2. Task 2 owns data sorting and `pinned` ingestion.
3. Task 3 owns the only major `parade.go` rewrite.
4. Task 4 owns `app.go` for keyboard/collapse preservation.
5. Task 5 owns `detail.go` only for reference metadata and status copy.
6. Task 6 owns `app.go` for mouse routing after Task 5's interface exists.
7. Task 7 owns final docs and built-binary acceptance.

Shared contracts after Task 1:

```go
const (
    StateReady SemanticState = iota
    StateWorking
    StateWaitingBlocked
    StateDeferred
    StateOperatorReview
    StateDone
)

const StatusReview Status = "review"

// UI remains a leaf package; these helpers accept the integer order above.
func ExecSymbol(int) string
func ExecColor(int) color.Color
func ExecSectionStyle(int) lipgloss.Style
func ExecIndicator(int) string
```

No task may add a seventh state, a second classifier, dotted-ID ancestry, a default newest-first key, a new right-pane children/widget language, or a permanent `in_progress` paint alias for Operator Review.

---

## File structure

| Files | Responsibility |
|---|---|
| `.beads/policy.yaml` | Project declaration for the approved custom review status and ready group; not a br implementation |
| `internal/data/{issue,semantic}.go`, tests | Stored review constant, exact six-state order/names, sole classifier, named legacy compatibility |
| `internal/ui/{symbols,theme,styles}.go`, `exec_test.go` | Confirmed glyphs and renamed/reordered dark/light `Exec*` vocabulary |
| `internal/components/header.go`, `header_test.go` | Six tally values in exact state order |
| `internal/tmux/status.go`, `status_test.go` | Headless six-state tally in exact state order |
| `internal/data/loader.go`, `contract_test.go` | Default priority + stable-ID sort, never recency |
| `internal/data/issue.go`, contract tests | Orthogonal `pinned` boolean ingestion |
| `internal/views/parade.go`, `parade_test.go`, `render_test.go` | Main cross-state forest, two attention sections, per-ID collapse, semantic row color, pin badge, left hit testing |
| `internal/app/app.go`, app tests | Collapse preservation, `>` keyboard route, mouse enable/routing, pane focus, selection synchronization |
| `internal/views/detail.go`, `detail_test.go` | Primary status without raw parenthetical; line-to-bead metadata for existing references |
| `internal/components/{help,palette}.go`, tests | Replace global closed-fold affordance with node collapse |
| `docs/keybindings.md`, `README.md`, `docs/ARCHITECTURE.md` | Operator Review, tree/collapse, mouse, glyphs, explicit review workflow |
| `testdata/mard-nob-awaiting-review.jsonl` or approved replacement | Legacy compatibility / stored-review end-to-end fixtures |
| `cmd/mg/main_test.go` | Built-binary six-state acceptance |

---

### Task 1: Real Operator Review, exact state order, and confirmed `Exec*` vocabulary

**Owner:** builder

**Files:**
- Create: `.beads/policy.yaml`
- Modify: `internal/data/issue.go:11-26`
- Modify: `internal/data/semantic.go:1-184`
- Test: `internal/data/semantic_test.go`
- Test: `internal/data/contract_test.go`
- Modify: `internal/ui/symbols.go:109-145`
- Modify: `internal/ui/theme.go:125-132,177-182,354-379`
- Modify: `internal/ui/styles.go:24-38,198-228,518-564`
- Test: `internal/ui/exec_test.go`
- Modify: `internal/components/header_test.go`
- Modify: `internal/tmux/status.go`, `internal/tmux/status_test.go`
- Modify: `internal/views/render_test.go:36-205`
- Modify: `internal/views/detail_test.go:15-74`

**Verification (anti-gameable):** in an isolated temporary copy of the board, `br update <probe> --status review` succeeds under the new policy; `br list --json --limit 0 --all` returns that probe with `status:"review"`; Go tests prove only that exact status derives Operator Review, while an arbitrary custom status remains hidden. The live board is not mutated by this proof.

- [x] **Step 1: Write failing classifier and vocabulary tests**

Update `TestStateOrderAndLabel` to require exact integer order and labels:

```go
want := []struct {
    state SemanticState
    label string
}{
    {StateReady, "Ready"},
    {StateWorking, "Working"},
    {StateWaitingBlocked, "Waiting/Blocked"},
    {StateDeferred, "Deferred"},
    {StateOperatorReview, "Operator Review"},
    {StateDone, "Done"},
}
```

Add direct storage and negative-custom rows:

```go
func TestDeriveStateOperatorReviewIsStored(t *testing.T) {
    issue := Issue{ID: "epic", Status: StatusReview, IssueType: TypeEpic}
    got, ok := DeriveState(&issue, BuildIssueMap([]Issue{issue}), DefaultBlockingTypes)
    if !ok || got != StateOperatorReview {
        t.Fatalf("review = (%v, %v), want Operator Review", got, ok)
    }
}

func TestDeriveStateUnknownCustomStatusStaysHidden(t *testing.T) {
    issue := Issue{ID: "x", Status: Status("qa_gate")}
    got, ok := DeriveState(&issue, BuildIssueMap([]Issue{issue}), DefaultBlockingTypes)
    if ok || got == StateReady || got == StateOperatorReview {
        t.Fatalf("unknown custom = (%v, %v), want hidden", got, ok)
    }
}
```

Rename the hard-example test to `TestDeriveStateLegacyConvergedEpicCompatibility` and require the approved named shim. Add a test that raw `in_progress` with any open descendant is Working and raw `review` is Operator Review even with no descendants. Update `TestContractAllStatusValues` to include `StatusReview`.

Update `execStates` to the operator-confirmed glyphs and order. Under the recommended decision:

```go
var execStates = []struct {
    name string
    state int
    glyph string
    palette func() color.Color
}{
    {"ready", 0, "○", func() color.Color { return BrightGold }},
    {"working", 1, "●", func() color.Color { return BrightGreen }},
    {"waiting/blocked", 2, "⊘", func() color.Color { return StatusStalled }},
    {"deferred", 3, "⏸", func() color.Color { return Dim }},
    {"operator review", 4, "◐", func() color.Color { return Orange }},
    {"done", 5, "✓", func() color.Color { return Muted }},
}
```

Update header/tmux/render fixtures to expect exact ordered pairs, e.g. sample counts `12○ 3● 3⊘ 0⏸ 0◐ 3✓` under the recommended order. Leave process-level `cmd/mg` acceptance and fixture changes to Task 7. Update detail expectation to `◐ Operator Review` with no raw parenthetical.

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/data ./internal/ui ./internal/components ./internal/tmux ./internal/views -run 'Test(DeriveState|StateOrder|ContractAllStatusValues|ExecVocabulary|ExecIndicator|HeaderCounts|StatusLine|StatusSymbol|StatusColor|ParadeViewSections|SemanticStatus)' -v
```

Expected: compile failures for `StateOperatorReview` / `StatusReview`, old order/glyph expectations, and old `Awaiting Review (in_progress)` detail copy.

- [x] **Step 3: Add the policy and implement the classifier cutover**

Write the exact operator-approved policy. Under the recommended decision it must declare every status the board is allowed to use and keep `review` out of ready:

```yaml
workflow:
  statuses:
    - open
    - in_progress
    - blocked
    - deferred
    - draft
    - closed
    - tombstone
    - pinned
    - review
  status_groups:
    ready:
      - open
```

If the approved policy schema requires a different list/object shape, use the exact verified `br` schema and keep the same semantics; do not guess transitions. Add `StatusReview`, rename/reorder the semantic constants, and make direct `review` classification precede blocking. Rename the compatibility helper:

```go
func legacyConvergedEpicOperatorReview(i *Issue, issueMap map[string]*Issue) (bool, bool)
```

Call it only for non-review legacy rows after direct review and blocked classification. Preserve hidden draft/tombstone/raw-pinned/arbitrary custom behavior. Update comments to give the approved deletion condition.

- [x] **Step 4: Rename/reorder the complete UI vocabulary atomically**

Rename `AwaitingReview` identifiers to `OperatorReview` throughout `internal/ui`, bind the confirmed glyphs, and reorder every switch to the new integer contract. Update tmux colors in the same semantic order:

```go
case data.StateReady:          return "colour220"
case data.StateWorking:        return "colour42"
case data.StateWaitingBlocked: return "colour196"
case data.StateDeferred:       return "colour240"
case data.StateOperatorReview: return "colour208"
case data.StateDone:           return "colour244"
```

Do not change palette primitives. Preserve the theme-rebake proof in `TestExecIndicatorRebakesOnThemeSwitch`.

- [x] **Step 5: Prove the Beads representation without mutating this board**

Create a temporary directory outside the repo, copy `.beads` and the new policy into it without hard-linking, then run the approved commands there:

```bash
br --db /tmp/mg-review-proof/.beads/beads.db update <copied-probe-id> --status review --json
br --db /tmp/mg-review-proof/.beads/beads.db list --json --limit 0 --all
```

Expected: update succeeds and list output contains the same ID with `"status":"review"`. Confirm the real repo's `br show <copied-probe-id>` is unchanged. If `br --db` does not make policy discovery use the temp directory, run from `/tmp/mg-review-proof` with its copied `.beads`; never point a mutation at the real board.

- [x] **Step 6: Run targeted tests and stale-vocabulary search**

Run:

```bash
go test ./internal/data ./internal/ui ./internal/components ./internal/tmux ./internal/views -run 'Test(DeriveState|StateOrder|ContractAllStatusValues|ExecVocabulary|ExecIndicator|HeaderCounts|StatusLine|StatusSymbol|StatusColor|SemanticStatus)' -v
rg -n 'StateAwaitingReview|ExecAwaitingReview|SymExecAwaitingReview|Awaiting Review|♪' internal README.md docs/ARCHITECTURE.md
```

Expected: tests PASS. Search may still find only docs/fixtures explicitly scheduled for Task 7; no production Go identifier or execution glyph remains stale.

- [x] **Step 7: Commit**

```bash
git add .beads/policy.yaml internal/data/issue.go internal/data/semantic.go internal/data/semantic_test.go internal/data/contract_test.go internal/ui internal/components/header_test.go internal/tmux internal/views/render_test.go internal/views/detail_test.go
git commit -m "feat: store and render operator review as a real state"
```

---

### Task 2: Priority-only default sort and orthogonal pin ingestion

**Owner:** builder
**Depends on:** Task 1

**Files:**
- Modify: `internal/data/loader.go:46-60`
- Test: `internal/data/contract_test.go:630-653`
- Modify: `internal/data/issue.go:93-122`
- Test: `internal/data/contract_test.go`
- Test: `internal/data/source_test.go`

**Verification (anti-gameable):** a fixture with equal-priority IDs and deliberately reversed timestamps sorts by ID after both JSONL and `br` envelope parsing; changing only `UpdatedAt` cannot change output. A JSON row with `pinned:true` round-trips into `Issue.Pinned` without changing `Status`.

- [x] **Step 1: Write failing sort and pin tests**

Replace the weak sort contract with:

```go
func TestSortIssuesUsesPriorityThenIDNeverRecency(t *testing.T) {
    old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
    fresh := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
    issues := []Issue{
        {ID: "z", Priority: PriorityHigh, Status: StatusClosed, UpdatedAt: fresh},
        {ID: "b", Priority: PriorityHigh, Status: StatusOpen, UpdatedAt: fresh},
        {ID: "a", Priority: PriorityHigh, Status: StatusInProgress, UpdatedAt: old},
        {ID: "p0", Priority: PriorityCritical, Status: StatusClosed, UpdatedAt: old},
    }
    SortIssues(issues)
    if got := issueIDs(issues); !reflect.DeepEqual(got, []string{"p0", "a", "b", "z"}) {
        t.Fatalf("order = %v", got)
    }
}
```

Add equivalent parse tests for `parseBrListOutput` and `LoadIssues`. Add:

```go
func TestContractPinnedIsOrthogonal(t *testing.T) {
    var issue Issue
    if err := json.Unmarshal([]byte(`{"id":"x","status":"open","pinned":true}`), &issue); err != nil { t.Fatal(err) }
    if !issue.Pinned || issue.Status != StatusOpen { t.Fatalf("%+v", issue) }
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/data -run 'Test(SortIssuesUsesPriorityThenIDNeverRecency|ParseBrList.*Sort|LoadIssues.*Sort|ContractPinnedIsOrthogonal)' -v
```

Expected: recency/active partition produces the wrong order; `Issue.Pinned` is undefined.

- [x] **Step 3: Implement minimal data changes**

Add:

```go
Pinned bool `json:"pinned,omitempty"`
```

Change `SortIssues` to priority ascending then ID ascending. Use `sort.SliceStable`; do not read status, `UpdatedAt`, `CreatedAt`, or semantic state. Do not sort pin badges ahead of unpinned rows.

- [x] **Step 4: Run targeted tests and recency search**

Run:

```bash
go test ./internal/data -run 'Test(SortIssues|ContractSort|ParseBrList|LoadIssues|ContractPinned)' -v
rg -n 'UpdatedAt\.After|then by recency|active first' internal/data/loader.go internal/data/*_test.go
```

Expected: tests PASS; search is empty in the sort contract (unrelated timestamp tests may remain outside the selected files).

- [x] **Step 5: Commit**

```bash
git add internal/data/loader.go internal/data/issue.go internal/data/contract_test.go internal/data/source_test.go
git commit -m "fix: sort default work by graph-ready priority"
```

---

### Task 3: Cross-state work tree, attention sections, semantic color, pin badge, and row hit testing

**Owner:** builder
**Depends on:** Task 2

**Files:**
- Modify: `internal/views/parade.go:19-750`
- Test: `internal/views/parade_test.go`
- Test: `internal/views/render_test.go`
- Modify only if a pure shared helper is necessary: `internal/data/hierarchy.go`, `internal/data/hierarchy_test.go`

**Verification (anti-gameable):** one rendered fixture contains an Operator Review epic with Working, Ready, nested Done, and nested-grandchild rows contiguous beneath it; no Working/Ready/Operator Review/Done section headers exist; only populated Waiting/Blocked and Deferred sections exist; an old Done child is not displaced by timestamp. ANSI-aware tests verify every row ID/glyph uses its semantic color and no age gradient symbol remains.

- [x] **Step 1: Write failing tree and section tests**

Create a fixture with:

```go
issues := []data.Issue{
    {ID: "epic", Status: data.StatusReview, Priority: 1, IssueType: data.TypeEpic},
    {ID: "epic.1", Status: data.StatusInProgress, Priority: 2, Dependencies: parentEdge("epic.1", "epic")},
    {ID: "epic.2", Status: data.StatusOpen, Priority: 0, Dependencies: parentEdge("epic.2", "epic")},
    {ID: "epic.3", Status: data.StatusClosed, Priority: 3, Dependencies: parentEdge("epic.3", "epic")},
    {ID: "epic.2.1", Status: data.StatusClosed, Priority: 1, Dependencies: parentEdge("epic.2.1", "epic.2")},
    {ID: "blocked", Status: data.StatusBlocked, Priority: 0},
    {ID: "later", Status: data.StatusDeferred, Priority: 1},
}
```

Assert issue order exactly:

```go
[]string{"epic", "epic.2", "epic.2.1", "epic.1", "epic.3", "blocked", "later"}
```

The parent comes first; siblings use priority then ID; the grandchild follows its parent; states do not split the family. Require output to omit headers for Ready, Working, Operator Review, Done and include exactly one Waiting/Blocked plus one Deferred header. Add an empty-deferred case where Deferred is absent.

Add collapse tests:

```go
p.ToggleNode("epic.2") // hides only epic.2.1
p.ToggleNode("epic")   // hides every epic descendant
```

Require `Collapsed` state by full ID, selection fallback to collapsed ancestor, and unrelated attention rows unchanged.

- [x] **Step 2: Write failing semantic color, pin, and hit-test tests**

Add a very old Ready issue and a fresh Ready issue; compare their rendered ID SGR color to `ui.ExecColor(int(data.StateReady))` and require equality. Add `Pinned:true` and require `PIN` in the row while `DeriveState` remains Ready and the group count is unchanged.

Add:

```go
func TestParadeIssueAtViewportRowUsesScrollOffset(t *testing.T) {
    // force scroll; header/footer rows return nil; issue row returns the exact full ID
}
```

Require one line per item and no padding row target.

- [x] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/views -run 'TestParade(Tree|AttentionSections|Collapse|SemanticColor|PinnedBadge|IssueAtViewportRow|RelativeDisplayID)' -v
```

Expected: six section headers/global Done fold, no per-node collapse, age-based ID color, no pin badge/hit helper.

- [x] **Step 4: Implement the tree model in `parade.go`**

Replace global-section state with:

```go
type ParadeItem struct {
    IsHeader bool
    IsFooter bool
    Section *paradeSection // nil for main-tree rows
    Issue *data.Issue
    Eval *data.DepEval
    State data.SemanticState
    RenderedID string
    Depth int
    HasChildren bool
}

type Parade struct {
    // existing fields...
    Collapsed map[string]bool
}
```

Implement pure helpers local to the view unless data reuse is real:

```go
func orderForest(issues []data.Issue) (ordered []*data.Issue, depth map[string]int, hasChildren map[string]bool)
func (p *Parade) appendForest(issues []data.Issue)
func (p *Parade) ToggleNode(issueID string)
func (p *Parade) IssueAtViewportRow(row int) *data.Issue
```

The main-tree set is `{Ready, Working, OperatorReview, Done}`. Append it first without a header. Append Waiting/Blocked and Deferred through the existing border style only when non-empty. For a child whose parent is outside the same display set, render depth 0 and full ID.

`orderForest` sorts root and child slices by priority then ID before depth-first walking. Preserve cycle/missing-parent safety. Do not use timestamps or dotted text.

- [x] **Step 5: Implement collapse, semantic color, compact IDs, and pin badge**

Disclosure marker:

```go
marker := " "
if item.HasChildren {
    marker = ui.Expanded
    if p.Collapsed[issue.ID] { marker = ui.Collapsed }
}
```

`ToggleNode` is a no-op for leaves. Collapsed ancestors suppress descendants during flattening. Delete `ShowClosed`, `ToggleClosed`, `renderLegend`, and `idStyleForAge`. Render every ID with:

```go
idStyle := lipgloss.NewStyle().Foreground(statusColor(item.State))
```

Use `data.RelativeDisplayID` whenever the actual parent is the visible ancestor in this forest, including a Done child under an open/review epic. Render `PIN` with an existing badge style or a new `internal/ui` badge only if required by theme conventions; do not change state or ordering.

- [x] **Step 6: Run targeted tests and deletion search**

Run:

```bash
go test ./internal/views -run 'TestParade(Tree|AttentionSections|Collapse|SemanticColor|PinnedBadge|IssueAtViewportRow|RelativeDisplayID|Indent)' -v
rg -n 'idStyleForAge|ShowClosed|ToggleClosed|renderLegend|GradientHeat' internal/views/parade.go internal/views/*_test.go
```

Expected: tests PASS; search empty. `ui.GradientHeat` may remain for unrelated non-parade visuals.

- [x] **Step 7: Commit**

```bash
git add internal/views/parade.go internal/views/parade_test.go internal/views/render_test.go internal/data/hierarchy.go internal/data/hierarchy_test.go
git commit -m "feat: render work as a collapsible semantic tree"
```

---

### Task 4: Preserve collapse state and bind `>` without regressing `E` / `esc`

**Owner:** builder
**Depends on:** Task 3

**Files:**
- Modify: `internal/app/app.go:2050-2619,2870-2990,3423-3593`
- Test: `internal/app/keys_test.go`
- Test: `internal/app/app_test.go`
- Modify: `internal/components/help.go:70-134`
- Modify: `internal/components/palette.go`
- Test: `internal/components/help_test.go`
- Test: `internal/app/palette_test.go`
- Test: `internal/app/update_test.go`

**Verification (anti-gameable):** Bubble Tea key messages collapse a nested branch, then exercise `FileChangedMsg`, `WindowSizeMsg`, and `E` scope; the same branch remains collapsed, unrelated rows stay visible, and `esc` still clears epic scope before anything else. No global Done toggle remains in keys or palette.

- [ ] **Step 1: Write failing key and rebuild tests**

Add:

```go
func TestKeyGreaterTogglesSelectedTreeNode(t *testing.T)
func TestCollapseSurvivesFileReloadAndResize(t *testing.T)
func TestCollapseSurvivesEpicScopeRebuild(t *testing.T)
func TestKeyEscStillClearsScopeBeforeTreeFocus(t *testing.T)
```

Use an epic, child-with-grandchild, and unrelated root. Dispatch:

```go
tea.KeyPressMsg{Code: '>', Text: ">"}
tea.WindowSizeMsg{Width: 120, Height: 30}
```

For the reload case send the existing `data.FileChangedMsg` shape with the same IDs. Assert collapsed IDs and visible row IDs, not internal call counts.

Update help/palette tests to require `>` "Collapse / expand selected branch" and absence of `c` / "Toggle closed issues".

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app ./internal/components -run 'Test(KeyGreater|CollapseSurvives|KeyEscStill|Help.*Collapse|Palette.*Collapse)' -v
```

Expected: `>` unbound, collapse map lost when `NewParadeWithData` replaces the parade, old `c` action remains.

- [ ] **Step 3: Implement app preservation and keyboard route**

In `rebuildParade`, capture and restore the collapse set just as selection and dimensions are preserved:

```go
oldCollapsed := maps.Clone(m.parade.Collapsed)
// build new parade
m.parade.Collapsed = oldCollapsed
m.parade.RebuildItems()
```

Use the actual exported helper name Task 3 lands. Bind `>` only while `PaneParade` is active, toggle the selected full ID, then call `syncSelection`. Remove global `case "c"`, `ActionToggleClosed`, palette command, `oldShowClosed`, and their tests. Do not change `E`, scope-first `esc`, filter, focus, or layout-preset behavior.

- [ ] **Step 4: Update help only**

Replace the parade binding with:

```go
{key: ">", desc: "Collapse / expand selected branch"}
```

Do not document mouse until Task 7, when it is executable.

- [ ] **Step 5: Run targeted tests and stale-toggle search**

Run:

```bash
go test ./internal/app ./internal/components -run 'Test(KeyGreater|CollapseSurvives|KeyEsc|KeyE|Help|Palette)' -v
rg -n 'ToggleClosed|ShowClosed|ActionToggleClosed|Toggle closed issues|case "c"' internal/app internal/components
```

Expected: tests PASS; search empty.

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/keys_test.go internal/app/app_test.go internal/app/palette_test.go internal/app/update_test.go internal/components/help.go internal/components/help_test.go internal/components/palette.go
git commit -m "feat: preserve per-branch collapse across cockpit rebuilds"
```

---

### Task 5: Detail status cleanup and existing bead-reference metadata

**Owner:** builder
**Depends on:** Task 4

**Files:**
- Modify: `internal/views/detail.go:17-178,202-471`
- Test: `internal/views/detail_test.go`

**Verification (anti-gameable):** rendered content for each existing dependency row class resolves by viewport row to the exact loaded issue; scrolling changes the viewport row but not the target. Missing and unloaded cross-rig refs return nil. Status is exactly `<glyph> Operator Review`, with no raw parenthetical.

- [ ] **Step 1: Write failing status and reference tests**

Keep the primary status assertion strict:

```go
row := statusRow(d.renderContent())
if strings.Contains(row, "(") || !strings.Contains(row, "◐ Operator Review") {
    t.Fatalf("status row = %q", row)
}
```

Create a detail fixture whose selected issue has:

- a loaded unresolved blocker
- a missing blocker
- a loaded resolved blocker
- a loaded `related` edge
- a loaded `parent-child` parent
- a loaded reverse dependent from `BlocksIDs`
- a cross-rig reference absent from `IssueMap`

After `SetIssue`, locate each plain rendered line by its unique ID and assert `ReferenceAt(line-d.Viewport.YOffset())` returns the matching loaded issue, while missing/cross-rig absent targets return nil. Set `Viewport.SetYOffset(...)` and repeat for a visible reference.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/views -run 'Test(SemanticStatusHardExample|DetailReferenceAt|DetailReferenceAtScrolled|DetailMissingReference)' -v
```

Expected: raw parenthetical remains and `ReferenceAt` is undefined.

- [ ] **Step 3: Implement render-time metadata**

Add:

```go
type Detail struct {
    // existing fields
    referenceLines map[int]string
}

func (d *Detail) ReferenceAt(viewportRow int) *data.Issue {
    if viewportRow < 0 || viewportRow >= d.Viewport.Height() { return nil }
    id := d.referenceLines[d.Viewport.YOffset()+viewportRow]
    return d.IssueMap[id]
}
```

Reset `referenceLines` at the start of every `renderContent`. Record the current `len(lines)` immediately before appending each existing DEPENDENCIES or CROSS-RIG reference row whose ID is known. Do not record the current issue's own ID, Progress, molecule titles, markdown text, section headings, or blank lines. Remove `("+raw+")` from the status row.

- [ ] **Step 4: Run targeted tests**

Run:

```bash
go test ./internal/views -run 'Test(SemanticStatus|DetailReferenceAt|SetIssuePreservesScroll|SetSizeUpdatesDimensions|EpicProgress|Molecule)' -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/views/detail.go internal/views/detail_test.go
git commit -m "feat: expose existing detail bead references for navigation"
```

---

### Task 6: Bubble Tea mouse selection, focus, and wheel routing

**Owner:** builder
**Depends on:** Task 5

**Files:**
- Modify: `internal/app/app.go:768-2005,3423-3458,3934-4166`
- Test: `internal/app/keys_test.go`
- Test: `internal/app/view_height_test.go`

**Verification (anti-gameable):** real Bubble Tea v2 `tea.MouseClickMsg` / `tea.MouseWheelMsg` values at nonzero scroll offsets select the expected full ID (click) or scroll the pane under the pointer (wheel) WITHOUT changing `activPane` / `detail.Focused`. The app's returned `tea.View` requests `MouseModeCellMotion`. Overlay/modal cases prove no underlying selection moves.

- [ ] **Step 1: Write failing View and click tests**

Add:

```go
func TestViewEnablesMouseCellMotion(t *testing.T) {
    m := setupModel(t)
    if got := m.View().MouseMode; got != tea.MouseModeCellMotion { t.Fatalf("%v", got) }
}
```

Build a sized model with enough rows to scroll. Dispatch actual v2 messages:

```go
tea.MouseClickMsg{X: leftX, Y: bodyTop + visibleRow, Button: tea.MouseLeft}
tea.MouseWheelMsg{X: leftX, Y: bodyTop + 1, Button: tea.MouseWheelDown}
```

If v2's alias construction requires conversion from `tea.Mouse`, use `tea.MouseClickMsg(tea.Mouse{...})`; compile against v2.0.9, do not invent fields.

Require:

```go
func TestMouseClickParadeSelectsScrolledRowAndFocusesParade(t *testing.T)
func TestMouseClickDetailReferenceNavigatesAndFocusesDetail(t *testing.T)
func TestMouseWheelScrollsPaneUnderPointerWithoutStealingFocus(t *testing.T)
func TestMouseIgnoresHeaderFooterPaddingAndScrollCue(t *testing.T)
func TestMouseDoesNotLeakThroughHelpOrForms(t *testing.T)
```

The right-reference fixture must use Task 5's real DEPENDENCIES row. Assert selected issue IDs and viewport offsets, not helper calls.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app -run 'Test(ViewEnablesMouse|MouseClick|MouseWheel|MouseIgnores|MouseDoesNotLeak)' -v
```

Expected: View mouse mode is None; mouse messages do not change selection/focus/scroll.

- [ ] **Step 3: Enable mouse and centralize body geometry**

Update `altView`:

```go
func altView(s string) tea.View {
    v := tea.NewView(s)
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    return v
}
```

Extract geometry constants/helpers used by both `layout` and mouse routing so header/body coordinates cannot drift:

```go
const headerHeight = 2
const footerHeight = 2

func (m Model) bodyBounds() (top, height, paradeWidth int)
```

Wide layout returns parade width `m.width` and no detail hit region.

- [ ] **Step 4: Route click and wheel messages**

In `Update`, after modal/form/input ownership checks and before focused-detail forwarding, handle `tea.MouseClickMsg` and `tea.MouseWheelMsg`.

- left click: set `activPane=PaneParade`, `detail.Focused=false`, call `parade.IssueAtViewportRow`, expand a collapsed epic if required, select by full ID, then `syncSelection`
- right click: set `activPane=PaneDetail`, `detail.Focused=true`; if `detail.ReferenceAt(bodyRow)` returns an issue, set detail issue and synchronize parade cursor only if `restoreParadeSelection` finds it
- left wheel: scroll Parade (`MoveUp` / `MoveDown` + `syncSelection`) without changing `activPane` / `detail.Focused`
- right wheel: scroll Detail (`Viewport.ScrollUp(1)` / `ScrollDown(1)`) without changing `activPane` / `detail.Focused`

Do not clear filters, scope, focus mode, or collapse state. Do not intercept mouse while help, palette, create/edit forms, prompts, dialogs, or right-panel overlays own the surface.

- [ ] **Step 5: Run targeted tests and screen-height regression**

Run:

```bash
go test ./internal/app -run 'Test(ViewEnablesMouse|MouseClick|MouseWheel|MouseIgnores|MouseDoesNotLeak|ScreenHeightMatchesTerminal|KeyTab|KeyEnter|KeyJK|KeyE|KeyEsc)' -v
```

Expected: PASS; screen remains exactly the requested terminal height.

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/keys_test.go internal/app/view_height_test.go
git commit -m "feat: focus and navigate cockpit panes with the mouse"
```

---

### Task 7: Documentation and built-binary cockpit acceptance

**Owner:** builder
**Depends on:** Task 6

**Files:**
- Modify: `README.md:12-38,76-115,178-195,199-215`
- Modify: `docs/keybindings.md:7-57,59-90`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `internal/components/help.go`, `internal/components/help_test.go`
- Modify or create approved fixtures under: `testdata/`
- Modify: `cmd/mg/main_test.go`
- Modify: `internal/tmux/status_test.go`

**Verification (anti-gameable):** a built `/tmp/mg-nfy` over a stored-review fixture prints the exact confirmed six-glyph count order; a real PTY smoke launches the TUI, sends `>` and mouse SGR input, and observes selection/focus/collapse on the actual surface. Stale product-language searches are empty outside historical `docs/plans/2026-09-16-*` files.

- [ ] **Step 1: Add failing process-level status acceptance**

Create or update a fixture with one `status:"review"` epic and mixed descendants. Add:

```go
func TestStatusModeOperatorReviewStoredState(t *testing.T) {
    // run go run ./cmd/mg --status --path testdata/operator-review-tree.jsonl
    // strip tmux markup; require exact confirmed six ordered pairs
    // require Operator Review count 1 and Working count 0 for the review epic
}
```

Keep a separate legacy fixture test only if the operator approved the temporary compatibility window. Name it `LegacyCompatibility` so it cannot be mistaken for primary storage.

- [ ] **Step 2: Run process tests to verify they fail**

Run:

```bash
go test ./cmd/mg ./internal/tmux -run 'TestStatusModeOperatorReviewStoredState|TestStatusLineOperatorReview' -v
```

Expected: missing fixture or stale ordered counts.

- [ ] **Step 3: Write docs to the shipped behavior**

README state list uses the exact confirmed glyphs and `Operator Review`. Replace Done-section / `c` language with tree-first and `>` per-branch collapse. Document:

```text
>     Collapse or expand the selected branch
click Select a bead and focus the pane under the pointer
wheel Move the pane under the pointer
E     Scope to selected epic subtree
esc   Clear scope first, then existing focus behavior
```

Document the explicit lead transition and ready-front rule exactly as approved, e.g. `br update <epic> --status review`; say `review` is absent from ready. Do not document a generic auto-enter rule unless that is the operator's answer. Preserve that keyboard operation is complete.

Architecture updates `SemanticState` to `OperatorReview`, describes one tree + two attention sections, and removes newest-first/global Done fold claims.

- [ ] **Step 4: Run targeted process and docs checks**

Run:

```bash
go test ./cmd/mg ./internal/tmux ./internal/components -run 'Test(StatusModeOperatorReview|StatusLineOperatorReview|Help)' -v
go build -o /tmp/mg-nfy ./cmd/mg
/tmp/mg-nfy --status --path testdata/operator-review-tree.jsonl
rg -n 'Awaiting Review|StateAwaitingReview|ExecAwaitingReview|♪|Toggle closed|Done section|newest-first|idStyleForAge' README.md docs/keybindings.md docs/ARCHITECTURE.md internal cmd
```

Expected: tests/build PASS; status line contains the exact approved ordered pairs; stale search empty. Historical `docs/plans/2026-09-16-*` is intentionally excluded.

- [ ] **Step 5: Smoke the actual TUI in a PTY**

Launch the built binary through a PTY against `testdata/operator-review-tree.jsonl` at a fixed size. Exercise:

1. select an epic; send `>`; capture screen and prove its descendants disappear while another root stays
2. send `>` again; descendants return in priority order with Done nested under the epic
3. send SGR left-click coordinates for a lower visible row; capture screen and prove the cursor/detail title moved to that exact bead
4. send SGR click/wheel in detail on a dependency reference; prove focus border changes and detail navigates/scrolls
5. send `E`, then `esc`; prove sibling scope still works and unwinds first

Expected: observable screen captures satisfy all five. Do not replace this with a unit-test-only claim.

- [ ] **Step 6: Commit**

```bash
git add README.md docs/keybindings.md docs/ARCHITECTURE.md internal/components/help.go internal/components/help_test.go testdata cmd/mg/main_test.go internal/tmux/status_test.go
git commit -m "docs: describe and prove the operator work cockpit"
```

---

## Lead verification after all tasks

Run once after every task verifier has passed:

```bash
go test ./...
go vet ./...
go build -o /tmp/mg-nfy ./cmd/mg
/tmp/mg-nfy --status --path testdata/operator-review-tree.jsonl
rg -n 'StateAwaitingReview|ExecAwaitingReview|SymExecAwaitingReview|Awaiting Review|♪|idStyleForAge|ShowClosed|ToggleClosed|ActionToggleClosed|UpdatedAt\.After' internal cmd README.md docs/keybindings.md docs/ARCHITECTURE.md
```

Required evidence:

- suite, vet, and build pass
- temporary Beads proof shows declared `review` survives `br update` and mg's exact `br list --json --limit 0 --all` source command without touching the real board
- status binary prints exact approved state/glyph order and one Operator Review for stored review
- TUI PTY smoke proves tree nesting, per-node collapse, left click, right-reference click, pane-local wheel, and `E`/`esc`
- stale production vocabulary/search is empty
- `mard-nfy.1` and `.2` remain deferred non-goals with no implementation task
- no PATH binary install, no merge/push, and no changes to `mard-nob` or `mard-r43`

## Open questions preserved from the spec

1. **Storage — settled.** Real beads status `review` in `.beads/policy.yaml`, written via `br update`. Forbidden: keep `in_progress` and only paint. Named shim `legacyConvergedEpicOperatorReview` for already-converged epics until migrated.
2. **What flips it — settled.** Explicit lead action (`br update <id> --status review`). Not auto-enter when children close.
3. **Right-pane clicks — settled.** Loaded DEPENDENCIES rows (blocking, resolved, non-blocking including parent, reverse `blocks`) plus a cross-rig ID only if that exact ID is locally loaded. Missing/unloaded visible, not clickable. No children widget. No molecule IDs.
4. **Glyphs — settled.** Ready `○` Working `●` Waiting/Blocked `⊘` Deferred `⏸` Operator Review `◐` Done `✓`.

Hover-scroll (Ata, same turn): wheel over left or right scrolls that pane and does not move keyboard/click focus. Task 6. Click still moves focus.