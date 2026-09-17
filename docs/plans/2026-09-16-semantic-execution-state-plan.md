# Semantic Execution State + Cockpit Foundations Implementation Plan

**Technical Spec:** `docs/plans/2026-09-16-semantic-execution-state-spec.md`
**Design Brief:** `/tmp/2026-09-16-semantic-state-brief.md` (via spec; do not bypass)

> **For the project lead:** first `br where` in this repo. no board → `br init --prefix <xx>` here (never `~/.beads`). then one campaign parent bead for this plan, then task-by-task with isolated `builder` workers. Do not implement these tasks inline. Persist progress in this file's checkboxes **and** as one parent-child bead per Task (`br create --parent --body`). Title `tN: <ask>` (prefix `[done] ` after verifier pass). Description markdown, ≤800 characters, wrap at ~60 cols: `Asked:` paragraph, then `## Landed` with sha/tests/leftover (`not yet` while in flight). Do not paste this plan packet.

**Goal:** Replace four raw-status buckets with Ata's six graph-derived execution states, restore graph edges on `br list`, and add compact IDs plus epic-subtree scope.

**Architecture:** `internal/data` owns edge traversal and the sole semantic classifier. CLI issues receive edges from the co-located JSONL export before classification. All surfaces consume `StateOrder()`; draft/tombstone/pinned/custom return `ok=false`, with no seventh state.

**Tech stack:** Go 1.25, BubbleTea v2, lipgloss v2, existing JSONL and CLI seams; no dependencies.

---

## File structure

| Files | Responsibility |
|---|---|
| `internal/data/hierarchy.go`, tests | Edge hierarchy, epic ancestry, subtree, compact IDs |
| `internal/data/semantic.go`, tests | Six states, classifier, epic rule, grouping |
| `internal/data/source.go`, tests | Back-fill CLI dependency edges |
| `internal/ui/{symbols,theme,styles}.go`, tests | Distinct six-state `Exec*` vocabulary |
| `internal/views/{parade,detail}.go`, tests | Semantic render and relative IDs |
| `internal/components/{header,footer,help}.go`, tests | Six counts and scope affordance |
| `internal/tmux/status.go`, tests | Six status segments |
| `internal/app/app.go`, tests | Semantic groups and epic scope |
| `cmd/mg/main.go`, tests | Semantic `--status` and process acceptance |
| `testdata/mard-nob-awaiting-review.jsonl` | Frozen hard-example fixture |
| `docs/keybindings.md`, `README.md` | Scope docs |

## Shared contract before parallel work

All builders use `SemanticState` values in this exact integer order: Working=0, AwaitingReview=1, Ready=2, Deferred=3, WaitingBlocked=4, Done=5. UI lookup helpers intentionally accept that integer order to keep `internal/ui` a leaf package. `GroupBySemanticState` always returns a map containing all six keys and a separate unmapped slice. No builder may add a default mapping, seventh state, dotted-ID ancestry, or per-issue CLI fetch.

## Dependencies and parallel work

- Wave A, parallel: t1 hierarchy; t3 source edge merge; t4 UI vocabulary.
- t2 semantic derivation follows t1.
- t5 atomic render/type cutover follows t2+t4.
- t6 relative IDs and t7 scope logic run in parallel after t5.
- t8 scope UI/docs follows t7. t9 end-to-end follows all.

---

### Task 1: Edge-based hierarchy primitives

**Owner:** builder

**Files:** Create `internal/data/hierarchy.go`, `internal/data/hierarchy_test.go`

**Verification (anti-gameable):** cycle, missing-parent, reparented dotted-ID, and edge-only-child tests pass without inferring hierarchy from ID text.

- [x] **Step 1: Write failing tests**

Use:

```go
func parentEdge(child, parent string) []Dependency {
	return []Dependency{{IssueID: child, DependsOnID: parent, Type: "parent-child"}}
}
```

Assert: `ChildrenOf` ignores dotted paint; `Descendants` is transitive/cycle-safe; `EpicAncestor` returns nearest epic; `ScopeToSubtree` preserves input order and returns nil for unknown root; `RelativeDisplayID` maps matching `mard-nob.7` to `.7` but keeps reparented and edge-only IDs full.

- [x] **Step 2: Verify failure**

Run: `go test ./internal/data -run 'TestChildrenOf|TestDescendants|TestEpicAncestor|TestScopeToSubtree|TestRelativeDisplayID' -v`
Expected: undefined functions.

- [x] **Step 3: Implement**

```go
func ChildrenOf(parentID string, issues []Issue) []*Issue
func Descendants(rootID string, issueMap map[string]*Issue) []*Issue
func EpicAncestor(i *Issue, issueMap map[string]*Issue) *Issue
func ScopeToSubtree(issues []Issue, rootID string) []Issue
func RelativeDisplayID(i *Issue) string
```

Preserve input order; use visited sets; strip a dotted prefix only when it equals `ParentRelationshipID()`.

- [x] **Step 4: Verify pass**

Run the Step 2 command. Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/data/hierarchy.go internal/data/hierarchy_test.go
git commit -m "feat: add edge-based issue hierarchy primitives"
```

---

### Task 2: Semantic state derivation and hard example

**Owner:** builder
**Depends on:** t1

**Files:** Create `internal/data/semantic.go`, `semantic_test.go`; modify `issue.go`, `contract_test.go`, `loader_test.go`

**Verification (anti-gameable):** the required test constructs exactly seven closed parent-child children, asserts Awaiting Review, and explicitly rejects Working. `rg -n 'StateUnknown|default:.*StateReady' internal/data` is empty.

- [x] **Step 1: Write failing tests**

Required acceptance test:

```go
func TestDeriveStateMardNobHardExample(t *testing.T) {
	issues := []Issue{{ID: "mard-nob", Status: StatusInProgress, IssueType: TypeEpic}}
	for n := 1; n <= 7; n++ {
		id := fmt.Sprintf("mard-nob.%d", n)
		issues = append(issues, Issue{ID: id, Status: StatusClosed, IssueType: TypeTask, Dependencies: parentEdge(id, "mard-nob")})
	}
	m := BuildIssueMap(issues)
	got, ok := DeriveState(m["mard-nob"], m, DefaultBlockingTypes)
	if !ok { t.Fatal("hard-example state must be decidable") }
	if got == StateWorking { t.Fatal("hard-example epic rendered Working") }
	if got != StateAwaitingReview { t.Fatalf("got %v, want Awaiting Review", got) }
}
```

Add table-driven tests with concrete expected outputs for all remaining rows: `closed→Done`; unresolved/dangling blocker→WaitingBlocked`; raw `blocked→WaitingBlocked`; raw/future defer→Deferred`; past defer plus open→Ready`; `in_progress→Working`; `open→Ready`. A zero-descendant epic and open-descendant epic are not Awaiting Review; a draft descendant makes the epic undecidable. Draft, tombstone, pinned, and arbitrary custom each return `ok=false`, appear in no group, and never equal Ready. Extend `TestContractAllStatusValues` to `open,in_progress,blocked,deferred,draft,closed,tombstone,pinned`. Port the existing blocking/custom-blocking tests from `ParadeGroup` to `DeriveState`. Load `testdata/sample.jsonl` and assert ordered counts `3,0,12,0,3,3` plus no unmapped rows.

Use this table shape so precedence expectations are executable rather than prose:

```go
tests := []struct {
	name string
	issue Issue
	extra []Issue
	want SemanticState
}{
	{"closed", Issue{ID: "x", Status: StatusClosed}, nil, StateDone},
	{"raw blocked", Issue{ID: "x", Status: StatusBlocked}, nil, StateWaitingBlocked},
	{"raw deferred", Issue{ID: "x", Status: StatusDeferred}, nil, StateDeferred},
	{"working", Issue{ID: "x", Status: StatusInProgress}, nil, StateWorking},
	{"ready", Issue{ID: "x", Status: StatusOpen}, nil, StateReady},
}
```

Add future/past defer times and unresolved/dangling dependency rows to this table. Separately loop over `[]Status{StatusDraft, StatusTombstone, StatusPinned, Status("custom-review")}` and fail when `ok` is true or state is Ready.

- [x] **Step 2: Verify failure**

Run: `go test ./internal/data -run 'TestDeriveState|TestEpicAwaitingReview|TestGroupBySemanticState|TestContractAllStatusValues' -v`
Expected: undefined semantic API/statuses.

- [x] **Step 3: Implement**

Add built-in status constants. Create exact order:

```go
const (
	StateWorking SemanticState = iota
	StateAwaitingReview
	StateReady
	StateDeferred
	StateWaitingBlocked
	StateDone
)
func (s SemanticState) Label() string
func StateOrder() []SemanticState
func DeriveState(*Issue, map[string]*Issue, map[string]bool) (SemanticState, bool)
func EpicAwaitingReview(*Issue, map[string]*Issue) (bool, bool)
func GroupBySemanticState([]Issue, map[string]bool) (map[SemanticState][]Issue, []Issue)
```

Follow spec precedence exactly. Pre-initialize all groups. Unmapped issues go only to the second return.

- [x] **Step 4: Verify pass**

Run Step 2 command. Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/data/semantic.go internal/data/semantic_test.go internal/data/issue.go internal/data/contract_test.go internal/data/loader_test.go
git commit -m "feat: derive semantic execution state from the issue graph"
```

---

### Task 3: Restore graph edges on CLI source

**Owner:** builder

**Files:** Modify `internal/data/source.go`, `source_test.go`

**Verification (anti-gameable):** mocked `br list` contains no dependencies; only temp `.beads/issues.jsonl` has the child edge; returned child reports the epic parent. No per-issue subprocess fan-out.

- [x] **Step 1: Write failing tests**

Create `TestFetchIssuesCLIBackfillsGraphEdges`: write a temp `.beads/issues.jsonl` containing epic plus child with a parent-child dependency; mock `br list` with the same issues but no `dependencies`; call `FetchIssuesCLI(tempDir, CLIBr)` and assert `BuildIssueMap(got)["child"].ParentRelationshipID() == "epic"`. Create `TestMergeGraphEdgesMissingExportLeavesInput` using `reflect.DeepEqual`, and `TestMergeGraphEdgesPreservesExistingDependencies` where the CLI issue already has a `blocks` edge that must remain unchanged.

- [x] **Step 2: Verify failure**

Run: `go test ./internal/data -run 'TestMergeGraphEdges|TestFetchIssuesCLIBackfillsGraphEdges' -v`
Expected: undefined function or missing edge.

- [x] **Step 3: Implement**

```go
func MergeGraphEdges(issues []Issue, projectDir string) []Issue {
	dir := ResolveBeadsDir(filepath.Join(projectDir, ".beads"))
	exported, _, err := LoadIssues(filepath.Join(dir, "issues.jsonl"))
	if err != nil { return issues }
	byID := BuildIssueMap(exported)
	for i := range issues {
		if len(issues[i].Dependencies) == 0 {
			if x := byID[issues[i].ID]; x != nil { issues[i].Dependencies = append([]Dependency(nil), x.Dependencies...) }
		}
	}
	return issues
}
```

Parse first in `FetchIssuesCLI`, then merge for both br and bd. Never run `sync`, `dep list`, or `show`.

- [x] **Step 4: Verify pass**

Run Step 2 command. Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/data/source.go internal/data/source_test.go
git commit -m "fix: restore dependency edges on CLI-loaded issues"
```

---

### Task 4: Six-state `Exec*` UI vocabulary

**Owner:** builder

**Files:** Modify `internal/ui/symbols.go`, `theme.go`, `styles.go`; create `exec_test.go`

**Verification (anti-gameable):** dark/light tests verify six glyphs/colors/indicators. Shared `Status*` and Gas Town `StateWorking`/`SymWorking` remain unchanged.

- [x] **Step 1: Write failing test**

Create `internal/ui/exec_test.go` and table expected glyphs `● ◐ ♪ ⏸ ⊘ ✓` for integer states 0..5 under both `ThemeDark` and `ThemeLight`; require each `ExecColor` non-nil, each `ExecIndicator` display width 1, and `ExecSectionStyle(state).GetBold()` true. Restore `ThemeDark` after the loop.

- [x] **Step 2: Verify failure**

Run: `go test ./internal/ui -run TestExecVocabulary -v`
Expected: undefined `Exec*` API.

- [x] **Step 3: Implement**

Add `SymExecWorking`, `SymExecAwaitingReview`, `SymExecReady`, `SymExecDeferred`, `SymExecWaiting`, `SymExecDone`. Add colors bound in `applyDerived()` to `BrightGreen`, `Orange`, `BrightGold`, `Dim`, `StatusStalled`, `Muted`. Add six `SectionExec*` and `Exec*Str` values plus:

```go
func ExecSymbol(int) string
func ExecColor(int) color.Color
func ExecSectionStyle(int) lipgloss.Style
func ExecIndicator(int) string
```

Keep legacy parade styles until t5.

- [x] **Step 4: Verify pass**

Run Step 2 command. Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/ui/symbols.go internal/ui/theme.go internal/ui/styles.go internal/ui/exec_test.go
git commit -m "feat: add six-state execution UI vocabulary"
```

---

### Task 5: Atomic semantic cutover and deletion of four-bucket API

**Owner:** builder
**Depends on:** t2, t4

**Files:** Modify `internal/data/issue.go`, `loader.go`, `loader_test.go`; `internal/ui/styles.go`; `internal/views/parade.go`, `parade_test.go`, `render_test.go`, `detail.go`, `detail_test.go`; `internal/components/header.go`, `header_test.go`; `internal/tmux/status.go`, `status_test.go`; `internal/app/app.go`; `cmd/mg/main.go`

**Verification (anti-gameable):** targeted render tests pass; stale-name search finds no `ParadeStatus`, `ParadeGroup`, `GroupByParade`, or parade-only Section/Str identifiers. Existing general-purpose `SymRolling/SymLinedUp/SymStalled/SymPassed` remain because non-parade callsites use them.

- [x] **Step 1: Rewrite tests to fail on old vocabulary**

Create a parade fixture containing: ordinary in-progress task; in-progress epic with one closed child edge; ordinary open task; future-deferred open task; raw blocked task; and a closed task. Assert the rendered view contains all six exact labels. Replace `TestParadeLabel` with a semantic detail test and assert the hard-example detail contains `◐ Awaiting Review (in_progress)`. Update header/tmux sample-fixture assertions to the ordered pairs `3● 0◐ 12♪ 0⏸ 3⊘ 3✓`. Port `TestStatusSymbol` and `TestStatusColor` to cover all six derived states; do not delete them. Keep indentation tests, changing expected Ready glyph to `SymExecReady`.

- [x] **Step 2: Verify failure**

Run: `go test ./internal/views ./internal/components ./internal/tmux -run 'TestParadeViewSections|TestSemanticStatus|TestHeader|TestStatusLine' -v`
Expected: old labels/types fail.
- [x] **Step 3: Cut over all consumers**

Delete `ParadeStatus`, its constants, `Issue.ParadeGroup`, and `GroupByParade`. Use:

```go
type paradeSection struct {
	Title string
	Symbol string
	Style lipgloss.Style
	Color color.Color
	State data.SemanticState
	BorderVertical string
}

func sections() []paradeSection {
	var out []paradeSection
	for _, state := range data.StateOrder() {
		c := ui.ExecColor(int(state))
		out = append(out, paradeSection{Title: state.Label(), Symbol: ui.ExecSymbol(int(state)), Style: ui.ExecSectionStyle(int(state)), Color: c, State: state, BorderVertical: lipgloss.NewStyle().Foreground(c).Render(ui.BoxVertical)})
	}
	return out
}
```

`Parade.Groups` becomes semantic and adds `Unmapped []data.Issue`. Row symbols come from section state, not raw status. Detail derives once and renders `<symbol> <label> (<raw>)`; defensive unmapped detail renders only raw status muted. Header and tmux iterate `StateOrder()`. Tmux colors: `colour42,208,220,240,196,244`. App stores semantic groups plus unmapped at all four grouping sites. `main --status` uses `GroupBySemanticState`.

Delete only `SectionRolling/LinedUp/Stalled/Passed` and `Status*Str`. Do **not** delete old general glyphs or shared palette colors: detail dependency glyphs, doctor, problems, gastown, codex transcript, and header progress still use them.

- [x] **Step 4: Verify pass and clean cutover**

Run:

```bash
go test ./internal/data ./internal/views ./internal/components ./internal/tmux ./internal/app ./cmd/mg -run 'Test(GroupBySemanticState|ParadeViewSections|SemanticStatus|Header|StatusLine|RebuildParade)' -v
rg -n 'ParadeStatus|ParadeRolling|ParadeLinedUp|ParadeStalled|ParadePastTheStand|ParadeGroup|GroupByParade|SectionRolling|SectionLinedUp|SectionStalled|SectionPassed|StatusRollingStr|StatusLinedUpStr|StatusStalledStr|StatusPassedStr' internal cmd
```

Expected: tests PASS; search empty.

- [x] **Step 5: Commit**

```bash
git add internal/data/issue.go internal/data/loader.go internal/data/loader_test.go internal/ui/styles.go internal/views internal/components/header.go internal/components/header_test.go internal/tmux internal/app/app.go cmd/mg/main.go
git commit -m "feat: cut every parade surface to semantic execution state"
```

---

### Task 6: Compact relative IDs at nested depth

**Owner:** builder
**Depends on:** t1, t5

**Files:** Modify `internal/views/parade.go`, `parade_test.go`, `render_test.go`

**Verification (anti-gameable):** a render test proves matching nested `mard-nob.7` becomes `.7`, reparented/edge-only children stay full, and depth-zero stays full; existing edge-indent tests pass.

- [x] **Step 1: Write failing render test**

Build epic plus matching, reparented, and edge-only children. Inspect `ansi.Strip(item.RenderedID)` and assert:

```go
map[string]string{
	"mard-nob": "mard-nob",
	"mard-nob.7": ".7",
	"legacy.2": "legacy.2",
	"edge-child": "edge-child",
}
```

- [x] **Step 2: Verify failure**

Run: `go test ./internal/views -run 'TestParadeRelativeDisplayID|TestParadeIndentUsesParentRelationships|TestRenderIssueHierarchicalIndent' -v`
Expected: matching child still full.

- [x] **Step 3: Implement**

In item construction:

```go
d := depth[iss.ID]
display := iss.ID
if d > 0 { display = data.RelativeDisplayID(iss) }
// RenderedID: idStyle.Render(display), Depth: d
```

Apply to open and Done paths or extract their identical append helper.

- [x] **Step 4: Verify pass**

Run Step 2 command. Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/views/parade.go internal/views/parade_test.go internal/views/render_test.go
git commit -m "feat: compact IDs for nested epic rows"
```

---

### Task 7: Epic-subtree scope key and filter

**Owner:** builder
**Depends on:** t1, t5

**Files:** Modify `internal/app/app.go`, `keys_test.go`

**Verification (anti-gameable):** real BubbleTea `E` key message scopes child selection to epic+descendants and removes an unrelated root; `esc` restores it; no-epic selection is unchanged.

- [x] **Step 1: Write failing key tests**

Create `setupEpicScopeModel`: issues are epic `in_progress`, child `open` with a parent-child edge, and unrelated `open`; size the model and select child. `TestKeyEScopesToEpicSubtree` dispatches `tea.KeyPressMsg{Code:'E',Text:"E"}`, asserts `scopeRootID=="epic"`, requires epic+child visible and unrelated absent. `TestKeyEscClearsEpicScope` presses E then Escape and requires unrelated restored. `TestKeyEScopeRequiresEpicAncestor` starts on an ordinary task and asserts root/visible IDs unchanged.

- [x] **Step 2: Verify failure**

Run: `go test ./internal/app -run 'TestKeyEScopesToEpicSubtree|TestKeyEscClearsEpicScope|TestKeyEScopeRequiresEpicAncestor' -v`
Expected: missing scope field/behavior.

- [x] **Step 3: Implement**

Add `scopeRootID string`. In `rebuildParade`, after fuzzy/exclude/focus and before grouping:

```go
if m.scopeRootID != "" {
	filteredIssues = data.ScopeToSubtree(filteredIssues, m.scopeRootID)
}
```

Include `scopeRootID != ""` in the regroup condition. `E` uses `EpicAncestor(selected, BuildIssueMap(m.issues))`, sets root, and rebuilds. `esc` clears scope first, then existing focus/detail behavior. Never extend `FocusFilter` or infer epic from dots.

- [x] **Step 4: Verify pass**

Run Step 2 command. Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/keys_test.go
git commit -m "feat: scope the parade to an epic subtree"
```

---

### Task 8: Persistent scope chip, help, and docs

**Owner:** builder
**Depends on:** t7

**Files:** Modify `internal/components/footer.go`, `footer_test.go`, `help.go`; `internal/app/app.go`; `docs/keybindings.md`; `README.md`

**Verification (anti-gameable):** footer test requires `SCOPE mard-r43` only when active. Manual targeted smoke scopes sample `mg-007` and observes chip disappear on Escape.

- [x] **Step 1: Write failing footer tests**

Add `TestFooterRendersEpicScopeChip`: set `ScopeRootID="mard-r43"`, strip ANSI from `View()`, require `SCOPE mard-r43`. Add `TestFooterOmitsEmptyEpicScopeChip`: default footer must not contain `SCOPE`.

```go
f := NewFooter(120, false, false, false)
f.ScopeRootID = "mard-r43"
if !strings.Contains(ansi.Strip(f.View()), "SCOPE mard-r43") { t.Fatal("missing scope chip") }
```

- [x] **Step 2: Verify failure**

Run: `go test ./internal/components -run 'TestFooter.*Scope' -v`
Expected: missing field.

- [x] **Step 3: Implement UI plumbing**

Add `ScopeRootID string` and prepend:

```go
ui.FooterKey.Render(ui.FleurDeLis + " SCOPE " + f.ScopeRootID)
```

beside the focus badge. In app footer construction set `footer.ScopeRootID = m.scopeRootID`.

- [x] **Step 4: Update help/docs**

Document:

```text
E    Scope to selected epic subtree
esc  Clear epic scope first, then focus mode, then return to parade
```

README mentions compact nested IDs and live epic scope only; no click/collapse/color/layout promises.

- [x] **Step 5: Verify pass**

Run: `go test ./internal/components ./internal/app -run 'TestFooter.*Scope|TestKeyE' -v`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add internal/components/footer.go internal/components/footer_test.go internal/components/help.go internal/app/app.go docs/keybindings.md README.md
git commit -m "docs: expose and explain epic subtree scope"
```

---

### Task 9: Built-binary hard-example acceptance

**Owner:** builder
**Depends on:** t3, t5, t6, t7, t8

**Files:** Create `testdata/mard-nob-awaiting-review.jsonl`; modify `cmd/mg/main_test.go`, `internal/tmux/status_test.go`

**Verification (anti-gameable):** built binary output includes ordered `0● 1◐ 0♪ 0⏸ 0⊘ 7✓`; Awaiting Review is 1 and Working is 0. This exercises loader, hierarchy, derivation, grouping, and tmux render in one process.

- [x] **Step 1: Write failing process test**

Create a process-level test in `cmd/mg/main_test.go` using this exact body:

```go
func TestStatusModeMardNobAwaitingReview(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/mg", "--status", "--path", "testdata/mard-nob-awaiting-review.jsonl")
	cmd.Dir = filepath.Join("..", "..")
	out, err := cmd.CombinedOutput()
	if err != nil { t.Fatalf("%v: %s", err, out) }
	for _, want := range []string{"0●", "1◐", "0♪", "0⏸", "0⊘", "7✓"} {
		if !strings.Contains(string(out), want) { t.Errorf("missing %q: %s", want, out) }
	}
	if strings.Contains(string(out), "1●") { t.Fatalf("hard-example epic rendered Working: %s", out) }
}
```

- [x] **Step 2: Verify failure**

Run: `go test ./cmd/mg -run TestStatusModeMardNobAwaitingReview -v`
Expected: fixture cannot be opened.

- [x] **Step 3: Create fixture**

Create eight JSONL rows: `mard-nob` is `in_progress` epic; `.1` through `.7` are closed; every child has `{"issue_id":"mard-nob.N","depends_on_id":"mard-nob","type":"parent-child"}`. Use real timestamps and `.6` type `docs`. Do not edit sample.jsonl or `.beads`.

- [x] **Step 4: Add fixture-to-tmux regression**

Load fixture, call `GroupBySemanticState` and `StatusLine`, assert the same six pairs and no unmapped issues.

- [x] **Step 5: Run targeted acceptance**

```bash
go test ./internal/tmux -run 'TestStatusLine.*MardNob|TestStatusLineFormat' -v
go test ./cmd/mg -run TestStatusModeMardNobAwaitingReview -v
go build -o /tmp/mg-r43 ./cmd/mg
/tmp/mg-r43 --status --path testdata/mard-nob-awaiting-review.jsonl
```

Expected: tests PASS; line contains `0● 1◐ 0♪ 0⏸ 0⊘ 7✓`, never `1●`.

- [x] **Step 6: Commit**

```bash
git add testdata/mard-nob-awaiting-review.jsonl cmd/mg/main_test.go internal/tmux/status_test.go
git commit -m "test: pin awaiting review through the built status command"
```

---

## Lead verification after all tasks

```bash
go test ./...
go vet ./...
go build -o /tmp/mg-r43 ./cmd/mg
/tmp/mg-r43 --status --path testdata/mard-nob-awaiting-review.jsonl
rg -n 'ParadeStatus|ParadeRolling|ParadeLinedUp|ParadeStalled|ParadePastTheStand|ParadeGroup|GroupByParade' internal cmd
```

Required evidence: suite/vet/build pass; binary shows Awaiting Review 1, Working 0, Done 7; stale search empty; unmapped negative tests pass; manual `E`/Escape scope and compact-ID smoke pass; no `.beads`, `mard-nob`, or installed-binary changes.

## Open product questions preserved from the spec

1. What does `draft` render as?
2. What does `tombstone` render as?
3. What does `pinned` render as, given br models it as a status?
4. What does an unrecognised custom status render as, given the six names are the authorised vocabulary?

Nothing in Tasks 1-9 answers these. The compile-visible `ok=false` hole is the intended boundary.