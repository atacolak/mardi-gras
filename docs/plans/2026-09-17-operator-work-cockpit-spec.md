# Operator Work Cockpit Technical Spec

**Design Brief:** `/home/sf/worlds/personal/designs/mardi-gras/operator-work-cockpit.md`
**Status:** four operator confirmations recorded 2026-09-17; Task 1 is executable. Hover-scroll (wheel-under-pointer, no focus steal) is a Task 6 contract.
**Repo:** `/home/sf/workspace/mardi-gras` (`feat/operator-observability`)

## Intent (from the Brief — do not rewrite)

Quoted invariants that bind this spec:

- "mg is the operator work cockpit, not a carnival parade and not a spreadsheet of raw beads statuses."
- "The left pane is a **tree of work**. An epic expands; its children live under it, including Done. Color is semantic state, not age. Sections exist only where a cut earns attention. Mouse click selects a row and moves pane focus. Keyboard still works."
- The six exact display names are **Ready**, **Working**, **Waiting/Blocked**, **Deferred**, **Operator Review**, and **Done**.
- "Operator Review" is a real operator gate, not `in_progress` painted differently. "Ata is not an agent and is not 24/7 progressing."
- "Done is not a separate ghetto at the bottom. Done rows belong under their epic."
- "Sort is graph + priority ... not newest-first."
- "Color of a bead is semantic state. Kill `idStyleForAge`."
- Sections exist only for **Waiting/Blocked** and **Deferred** (when deferred rows exist). Working, Ready, Operator Review, and Done live in the tree.
- Mouse clicks on a left-list bead or an existing bead reference in detail select the bead and move pane focus; mouse scroll and `j`/`k` then move the focused pane.
- `E` / `esc` remain the sibling-cockpit scope primitive.
- `DeriveState` remains the sole classifier.
- `draft`, `tombstone`, and unnamed custom statuses stay hidden. Pin is a badge, never a seventh state.
- Carnival glyphs are replaced, especially `♪`; the proposed six-glyph set remains subject to Ata's glance.

This wave continues `feat/operator-observability`. It does not start from `main`, install over `/home/sf/.local/bin/mg`, reopen `mard-nob`, or widen `mard-r43`.

## Mapping onto the current system

### Observed repository and board facts

- `internal/data/semantic.go:12-19` defines six states in the current order Working, Awaiting Review, Ready, Deferred, Waiting/Blocked, Done; `semantic.go:21-50` owns labels and `StateOrder`; `semantic.go:74-125` makes `DeriveState` the sole classifier.
- `semantic.go:99-106` currently derives Awaiting Review from an epic's settled descendants, even when the raw status is `in_progress`. `semantic.go:128-157` implements that legacy inference in `EpicAwaitingReview`.
- `internal/views/detail.go:219-227` renders the primary status as `<glyph> <semantic label> (<raw status>)`; `detail_test.go:45-53` pins `◐ Awaiting Review (in_progress)`. The parenthetical is the exact presentation the Brief removes.
- `internal/views/parade.go:20-46` builds six section definitions by iterating `StateOrder`; `parade.go:135-165` flattens all six into section-header / issue / footer rows. `parade.go:400-440` renders section counts, and `parade.go:369-385` renders a second six-state legend.
- `parade.go:147-160,230-260,421-429` owns the global Done-section fold (`ShowClosed`, `ToggleClosed`, key `c`). That model cannot put Done beneath its owning epic.
- `parade.go:177-194,695-700` calls `idStyleForAge`; it maps ID age through `ui.GradientHeat` over 30 days. No behavioral test requires the gradient.
- `internal/data/loader.go:47-59` sorts active before closed, priority ascending, then `UpdatedAt` descending. Both JSONL and CLI loaders call it (`loader.go:42`, `source.go:158,175`). The final tie-break is therefore newest-first today.
- `br` is 0.5.11. `br create --help` restricts initial `-s` to `open, deferred, in_progress, closed` (it includes `closed` and omits `blocked`). `br update --help` accepts an arbitrary non-terminal `--status`; terminal `closed` / `tombstone` require dedicated commands. `br list --status bogus` says custom statuses must be declared in `.beads/policy.yaml` under `workflow.statuses` or already exist on an issue. The installed binary also contains the `workflow.status_groups.ready` policy key. No `.beads/policy.yaml` exists in this repo.
- `internal/views/parade.go:49-86,351-397` already renders a flat slice of one-terminal-line `ParadeItem`s with `Cursor` and `ScrollOffset`. This is a suitable render cache for a tree: the hierarchy is built before flattening, and every visible item still maps one-to-one to a screen row.
- mg's CLI source uses `br list --json --limit 0 --all` (`internal/data/source.go:84-109`). On the current board that command includes deferred rows as well as in-progress and closed rows; `--status all` returns the same 22 active records. Because `review` does not exist yet, the plan must prove a declared custom status reaches this exact command rather than assuming it.
- `internal/data/mutate.go:9-15` can pass any `data.Status` to an update command, but hard-codes the legacy `bd` binary. This spec does not invent an mg quick-action key for review; the proposed explicit lead action is the documented `br update <id> --status review` workflow.
- Bubble Tea is v2.0.9 (`go.mod`). Mouse is logged only in `internal/app/debug.go:52-53`; no app update branch handles it. `altView` (`app.go:3934-3938`) creates a `tea.View` without setting `MouseMode`, so mouse reporting is disabled. Bubble Tea v2 enables it with `v.MouseMode = tea.MouseModeCellMotion` and emits `tea.MouseClickMsg` / `tea.MouseWheelMsg`.
- `app.go:3423-3458` fixes body geometry at a two-row header, two-row footer, and a 2:3 parade/detail split (minimum parade width 30). That is enough to map terminal mouse coordinates without introducing a zone library.
- `internal/views/detail.go:375-440` renders every existing reference to another bead. The `DEPENDENCIES` rows cover unresolved blockers, missing blockers, resolved blockers, every non-blocking edge (including `parent-child`, rendered as `child of`), and reverse `blocks` dependents. `CROSS-RIG` rows render external IDs. Each reference occupies its own rendered line, although the ID is inline with a verb and title.
- Detail does **not** render a children list: `epicProgress` (`detail.go:887-912`) returns counts only. Molecule rows (`detail.go:473-563`) render step titles, not bead IDs. This wave therefore adds no new right-pane widget language.
- `Detail.Viewport` owns scrolling (`detail.go:18-39`); its height is `Detail.Height-1` and the last row is a scroll cue (`detail.go:130-177`). A visible detail row maps to content line `Viewport.YOffset() + viewportRow`.
- `internal/ui/symbols.go:112-117` still proposes `● ◐ ♪ ⏸ ⊘ ✓`; `ui/exec_test.go:18-30` pins them. `theme.go:177-182` binds the six semantic colors through the existing dark/light rebake path.
- The live board reports both `mard-nob` and `mard-r43` as `in_progress` even though `br epic status` says 7/7 and 10/10 children closed and both are eligible for closure. There is no automatic transition into an operator gate today.
- `br schema issue` defines status as the eight built-ins plus a trailing bare string arm, so a declared custom `review` status is representable. The same schema exposes orthogonal `pinned: boolean`; mg's `data.Issue` (`issue.go:93-122`) does not currently deserialize it.

### Brief-to-code cutover table

| Brief contract | Current seam | Required cutover | Files / symbols |
|---|---|---|---|
| Six exact names | `SemanticState`, `Label`, `StateOrder` | Rename Awaiting Review to Operator Review; order all count surfaces as Ready, Working, Waiting/Blocked, Deferred, Operator Review, Done | `internal/data/semantic.go`, `semantic_test.go`; `internal/ui/{symbols,theme,styles,exec_test}.go`; `internal/tmux/status.go`; render/status tests; README/architecture |
| Real Operator Review | `DeriveState`, `EpicAwaitingReview` | Map the chosen stored representation directly; retain only a named, temporary legacy read-compat rule for already-converged epics if the operator accepts it | `.beads/policy.yaml` (preferred option), `internal/data/{issue,semantic}.go`, tests |
| Tree first | `Parade.Items`, edge helpers | Build one parent-first forest across Working, Ready, Operator Review, and Done; nested Done stays under its actual parent | `internal/views/parade.go`, `parade_test.go`, `render_test.go`; `internal/data/hierarchy.go` if a sibling-order helper is extracted |
| Only two attention sections | `sections`, `rebuildItems`, `renderLegend` | Main tree has no state header; only Waiting/Blocked and non-empty Deferred get section boundaries; remove the six-state legend | `internal/views/parade.go`, render tests |
| Per-node collapse via `>` | `ShowClosed`, `ToggleClosed`, key `c` | Replace the global Done fold with a collapse set keyed by issue ID; `>` toggles the selected node; selecting a collapsed epic expands it | `internal/views/parade.go`, `internal/app/app.go`, `internal/components/{help,palette}.go`, docs/tests |
| Graph + priority sort | `SortIssues`, `OrderHierarchically` | Parent precedes descendants; siblings and roots sort by priority then stable ID; delete `UpdatedAt` ordering and raw active/closed partitioning | `internal/data/loader.go`, `contract_test.go`; parade tree tests |
| Color is state | `idStyleForAge`, `statusColor` | Delete `idStyleForAge`; render each row ID/glyph with `ExecColor(derivedState)`; Done naturally uses its muted semantic color | `internal/views/parade.go`, render tests |
| Pin is a badge | Beads `pinned` boolean; mg omits it | Add `Issue.Pinned bool`; render `PIN` as a compact badge without changing `DeriveState`; legacy raw `status:pinned` remains hidden | `internal/data/issue.go`, contract tests; `internal/views/parade.go`, render tests |
| Unmapped stay hidden | `GroupBySemanticState(...).unmapped` | Continue carrying draft, tombstone, legacy raw pinned, and unnamed custom rows in `Unmapped`; never place them in the tree or either attention section | `internal/data/semantic.go`, `focus.go`; existing and extended tests |
| Mouse selection + focus | `activPane`, `altView`, one-line item cache | Enable cell-motion mouse mode; click/wheel route by body geometry; left row hit selects; right loaded ref hit navigates; pointer pane becomes focused | `internal/app/app.go`, `keys_test.go`; `internal/views/{parade,detail}.go`, tests |
| Existing right refs only | dedicated DEPENDENCIES/CROSS-RIG lines | Record line-to-ID metadata while rendering; loaded local dependency/parent/dependent references are targets; missing and external IDs remain visible but cannot select absent local data | `internal/views/detail.go`, `detail_test.go`; app mouse integration |
| Keep sibling cockpit | `scopeRootID`, `E`, scope-first `esc` | Preserve behavior and docs; tree rebuild remains downstream of filtering/focus/scope | `internal/app/app.go`, existing scope tests |
| Header tally stays | generic `StateOrder` iteration | Keep six counts and tmux status; do not turn them into parade sections | `internal/components/header.go`, `internal/tmux/status.go`, their tests |

## Architecture

### 1. Semantic state remains one classifier

`DeriveState` remains the only path from a loaded issue plus graph to operator-facing state. No view may switch on raw status to choose a row color, glyph, section, or label.

The preferred storage mapping, pending operator confirmation, is:

```go
const StatusReview Status = "review"

const (
    StateReady SemanticState = iota
    StateWorking
    StateWaitingBlocked
    StateDeferred
    StateOperatorReview
    StateDone
)
```

Recommended precedence if `review` is accepted:

| Precedence | Raw / graph condition | Semantic result |
|---|---|---|
| 1 | `closed` | Done |
| 2 | `draft`, `tombstone`, legacy raw `pinned`, unnamed custom | hidden / `ok=false` |
| 3 | explicit stored `review` | Operator Review |
| 4 | unresolved graph blocker or raw `blocked` | Waiting/Blocked |
| 5 | named temporary read-compat rule: non-closed epic with descendants all closed and legacy raw status not `review` | Operator Review |
| 6 | raw/future defer | Deferred |
| 7 | `in_progress` | Working |
| 8 | `open` | Ready |

The explicit stored gate precedes blocking because the Brief says Operator Review "is not blocked". The legacy rule remains below blocking because it is only compatibility for old ledger rows, not authority to overwrite an explicit gate.

The compatibility rule must be named `legacyConvergedEpicOperatorReview` (or an equivalently explicit name approved during implementation), tested as compatibility rather than primary classification, and documented with a deletion condition: remove it after existing review epics have a real stored gate and the read window agreed by the operator has elapsed. A generic `EpicAwaitingReview` inference is not retained as the primary product rule.

`in_progress` remains Working under the Beads/lead invariant that only a living assigned worker owns that raw status. This wave does not join actor-runtime liveness into `internal/data`; doing so would change the classifier input contract and is not required to make Operator Review real.

### 2. Tree composition replaces six buckets

The parade receives the already-narrowed issue slice from `rebuildParade` and classifies each row once with `DeriveState`.

- Main forest: Ready, Working, Operator Review, Done.
- Attention cuts: Waiting/Blocked section; Deferred section only when non-empty.
- Hidden: `Unmapped` rows.

For each display set, ancestry comes only from `parent-child` edges. The main forest crosses semantic states, which is the essential cutover: an Operator Review epic can own Ready/Working/Done descendants, and Done is no longer displaced into a global section. If a child's parent is outside its display set (for example, a blocked child whose epic remains in the main tree), the child is a depth-zero row in its attention section and keeps its full ID.

Parents render before descendants. Roots and siblings sort by `Priority` ascending (P0 first), then ID ascending for deterministic output. `UpdatedAt` is not a default key. The loaded slice may still be reused, but tree construction must explicitly enforce this comparator rather than relying on incidental loader arrival order.

`Parade` replaces `ShowClosed bool` with `Collapsed map[string]bool`. Only rows with actual edge-children show a disclosure marker. The map survives refresh, filter rebuild, scope changes, and resize. A collapsed ancestor suppresses its descendants from `Items`; it does not change header counts, which still describe all narrowed issues. If the selected row becomes hidden by collapsing an ancestor, selection moves to that ancestor. `>` is currently unbound and becomes the keyboard-complete toggle for the selected row. Mouse selection of a collapsed epic expands it before the row becomes current.

### 3. Semantic color and badges

The proposed glyphs, pending Ata's glance, are:

| State | Proposed glyph | Rationale | Existing color binding |
|---|---:|---|---|
| Ready | `○` | open and available; replaces carnival `♪` | `BrightGold` |
| Working | `●` | active, filled work | `BrightGreen` |
| Waiting/Blocked | `⊘` | stopped by another bead | `StatusStalled` |
| Deferred | `⏸` | intentionally paused | `Dim` |
| Operator Review | `◐` | work complete, acceptance half remains | `Orange` |
| Done | `✓` | completed | `Muted` |

All six stay in the existing `Exec*` API and theme rebake path; dark and light tests must continue proving that indicators are rebuilt after `SetTheme`.

`idStyleForAge` is deleted. The row's derived state supplies its glyph and ID color. Age remains textual in detail, but recency no longer competes with semantic state in the tree. `Issue.Pinned` adds a compact `PIN` badge after the ID/title budget; it never changes the state, section, sort rank, or header count.

### 4. Mouse and focus routing

`altView` enables `tea.MouseModeCellMotion`. `Model.Update` handles click and wheel messages before the generic focused-detail forwarding path.

Body geometry is derived from the same constants as `layout`:

- screen rows 0-1: header (not a bead target)
- body starts at row 2
- body height: `height - 4`
- parade width: `m.parade.Width`; remaining body width is the right pane

Left pane:

- click an issue row: `itemIndex = ScrollOffset + (mouseY - bodyTop)`; headers, footers, and padding do not select; a valid issue becomes `Cursor` / `SelectedIssue`, detail synchronizes, and focus becomes `PaneParade`
- click a collapsed epic: expand it, then select it
- wheel over left: scroll Parade one selectable row per event. do NOT change `activPane` / `detail.Focused`

Right detail pane:

- click anywhere in the pane: focus becomes `PaneDetail`
- click a recorded, locally loaded bead-reference line: navigate detail to that bead; if its row is visible in the current parade, synchronize the parade cursor as well
- click a missing dependency or cross-rig ID absent from `IssueMap`: keep focus but do not fabricate or fetch a bead
- wheel over right: scroll the existing viewport one line per event. do NOT change `activPane` / `detail.Focused`

Wheel follows the pointer, not the focused pane. A wheel event never steals keyboard/click focus. Click still moves focus. Keyboard `j`/`k` still move the focused pane.

Mouse messages are ignored by underlying panes while a modal/form/help overlay owns input. Wide layout has no right pane; its entire body is parade geometry.

### 5. Right-pane target metadata

`Detail` records references while constructing `renderContent`; it never parses ANSI-rendered text after the fact.

```go
type Detail struct {
    // existing fields...
    referenceLines map[int]string // content line -> locally selectable issue ID
}

func (d *Detail) ReferenceAt(viewportRow int) *data.Issue
```

`ReferenceAt` adds `Viewport.YOffset()` internally. It returns a bead only when the row was recorded and `IssueMap[id]` contains it.

Existing reference inventory:

| Existing rendered row | Target eligibility |
|---|---|
| unresolved blocker (`waiting on`) | clickable when loaded |
| missing blocker (`missing`) | visible, not clickable |
| resolved blocker (`resolved`) | clickable when loaded |
| non-blocking edge (`related`, `duplicates`, `supersedes`, `discovered from`, `waits for`, `child of`, `replies to`, generic) | clickable when loaded |
| reverse dependent (`blocks`) | clickable when loaded |
| cross-rig external reference | visible; clickable only if that exact ID is also loaded locally |
| current issue's own `ID:` row | not a navigation target |
| epic Progress count | not a bead reference |
| molecule step title | not a bead reference |

No `CHILDREN` section, link syntax, hover affordance, or new widget language is added.

## Components and interfaces

### `internal/data/issue.go`

- Add the chosen stored review constant if approved (`StatusReview`).
- Add `Pinned bool `json:"pinned,omitempty"`` to `Issue`; do not map legacy raw `status:pinned` to a state.

### `internal/data/semantic.go`

- Rename `StateAwaitingReview` to `StateOperatorReview` and label to `Operator Review`.
- Reorder the integer contract to Ready, Working, Waiting/Blocked, Deferred, Operator Review, Done; update every `Exec*` lookup test in the same atomic task.
- Replace `EpicAwaitingReview` as primary classification with the direct stored-state arm plus the explicitly named legacy compatibility helper, if accepted.
- Keep `DeriveState` and `GroupBySemanticState` signatures unless the storage choice requires a field already present on `Issue`; do not add a second classifier.

### `internal/data/loader.go`

`SortIssues` becomes deterministic priority order only:

```go
func SortIssues(issues []Issue) {
    sort.SliceStable(issues, func(i, j int) bool {
        if issues[i].Priority != issues[j].Priority {
            return issues[i].Priority < issues[j].Priority
        }
        return issues[i].ID < issues[j].ID
    })
}
```

Tree construction still sorts each sibling list itself; this loader rule removes newest-first from every default source and supplies deterministic input to non-tree consumers.

### `internal/views/parade.go`

- `ParadeItem.Section` becomes optional attention-section metadata; every issue row carries its own derived `State`.
- `rebuildItems` emits the main forest, then only Waiting/Blocked and Deferred attention cuts. It does not iterate `StateOrder` to create six sections.
- Add `Collapsed map[string]bool`, row-selection/hit-test helpers, and node toggle helpers.
- Delete `ShowClosed`, `ToggleClosed`, `renderLegend`, and `idStyleForAge`.
- Render compact IDs whenever the actual parent is the visible row above; this now includes Done-under-epic across state boundaries.
- Render `PIN` from `Issue.Pinned` as a badge without changing state.

### `internal/views/detail.go`

- Primary status is exactly `<glyph> <semantic label>`; remove the raw-status parenthetical.
- Rename Awaiting Review references to Operator Review.
- Record existing dependency/cross-rig reference line metadata and expose `ReferenceAt`.

### `internal/app/app.go`

- Preserve `E`, scope-first `esc`, the five-axis narrowing pipeline, resize-safe rebuild, and selection restoration.
- Preserve collapse state instead of `ShowClosed` in `rebuildParade`.
- Bind `>` to toggle the selected node; remove `c`'s global Done fold and its command-palette action.
- Enable and route Bubble Tea v2 click/wheel messages as described above.
- Do not clear filter/focus/scope to make a clicked reference visible.

### `internal/ui`, header, tmux, docs

- Rename and reorder the complete `Exec*` family atomically; apply the proposed glyph table only after glyph confirmation.
- Header and tmux keep six counts in the new `StateOrder`.
- Help, keybinding docs, README, architecture, fixture names, and built-binary status assertions use `Operator Review`, `>`, tree language, and the confirmed glyphs; delete claims about `c` folding a Done section.

## Data / control flow

1. Source load parses `pinned`, proves the declared `review` status is returned by mg's exact `br list --json --limit 0 --all` source path, merges graph edges, and sorts by priority + ID without `UpdatedAt`.
2. `rebuildParade` applies fuzzy filter → type/label exclusions → focus mode → epic scope exactly as today.
3. `GroupBySemanticState` classifies once for header/tmux counts and carries hidden statuses separately.
4. Parade tree construction consumes the narrowed issues, the derived-state map, edge hierarchy, and persistent collapse set. It emits the main forest plus two attention sections.
5. Keyboard or mouse selection updates parade selection and detail through the existing sync path. Right-reference navigation uses detail's full `IssueMap` and synchronizes the parade only when the target row is visible.
6. Poll/resize rebuilds preserve collapse keys and selection by real full issue ID; relative IDs remain presentation only.

## Error handling

- A malformed parent-child cycle emits every issue at most once as a root, preserving the existing `OrderHierarchically` safety contract.
- A child whose parent is hidden, filtered, in another attention cut, or absent renders at depth zero with its full ID.
- A collapsed-node ID that disappears on refresh is pruned lazily; stale keys have no visible effect.
- Collapsing an ancestor of the selected row moves selection to that ancestor and re-synchronizes detail.
- Mouse clicks outside the body, on section borders, padding, the detail scroll cue, or an unloaded reference only change focus when appropriate; they never panic or invent a bead.
- Mouse input is ignored while modal/form/help input owns the screen.
- Hidden statuses remain present in `Parade.Unmapped`, absent from tree/sections/counts, and never fall through to Ready.
- If the preferred `review` status is not declared, mg may read it as a custom string but builders must not silently treat every custom string as Operator Review. Only the exact approved representation is mapped.
- Missing/unreadable JSONL edge export keeps the existing edge-less fallback behavior; no new board write is introduced.

## Testing (behavioral contracts; exact tests live in the plan)

- A stored review bead derives Operator Review directly and never Working; the legacy converged-epic fixture exercises the named compatibility path only.
- State labels/order and all six glyphs are exact under dark and light themes; `♪` is absent from execution-state surfaces.
- An epic with mixed Working, Ready, and Done descendants renders as one parent-first tree; Done is beneath its real parent and no Done section exists.
- Only Waiting/Blocked and Deferred render section headers; an empty Deferred set renders no Deferred section; no six-state legend remains.
- Collapse hides only the selected node's descendants, survives reload/resize/scope rebuild, and preserves unrelated branches. `>` toggles it; `E`/`esc` behavior remains unchanged.
- Two equal-priority issues whose timestamps are reversed sort by stable ID, proving recency is not consulted; P0 precedes P1 among siblings.
- Row ID/glyph color equals `ExecColor(DeriveState(...))` regardless of age; no `idStyleForAge` symbol remains.
- `Pinned: true` renders a `PIN` badge while retaining the same state and header count as the unpinned equivalent.
- A left click selects the exact visible row after scroll offset and focuses Parade. A right click on each loaded dependency-row class navigates to its target and focuses Detail. Missing/external references do not fabricate selection.
- Built `mg --status --path <operator-review fixture>` reports one Operator Review and zero Working for the stored-review epic; if the temporary compatibility shim is approved, a separate legacy fixture proves the same presentation through that named compatibility path.
- Status detail reads `◐ Operator Review` (or the confirmed glyph), never `(in_progress)`.

## Non-goals

- No Gas Town / Gas City rewrite or actor control verbs.
- mg does not own or implement `br` / beads_rust; `.beads/policy.yaml` only declares the project workflow if the operator accepts that storage.
- No PATH `mg` installation; dogfood remains `./mg` on `feat/operator-observability` unless Ata asks.
- Newest-first is not retained as the default and no recency toggle is added.
- No seventh display state and no collapsing draft, tombstone, legacy raw pinned, or unnamed custom statuses into Ready.
- No new right-pane children widget, link language, or molecule-ID presentation.
- `mard-nfy.1` closed-epic sibling resume is deferred +30d: a later confirmation-gated control may restore a retired sibling into its herdr workspace **and** window; it has no task in this plan.
- `mard-nfy.2` subagent presentation is deferred +30d: OMP task/eval workers are not persistent actors, so talk midi before considering br work or another registry; it has no task in this plan.

## Implementation approach chosen (and rejected internals)

**Tree construction — chosen:** reuse edge hierarchy and flatten one cross-state forest into `Parade.Items`, extracting only Waiting/Blocked and Deferred into attention cuts. This keeps the existing one-line viewport, cursor, selection, and width-budget machinery. **Rejected:** six renamed buckets — violates tree-first and Done-under-epic. **Rejected:** a new tree widget/package — unnecessary abstraction around an existing flat render cache. **Rejected:** dotted-ID ancestry — already forbidden and wrong after reparenting.

**Collapse state — chosen:** a per-ID collapse set on `Parade`, preserved by `rebuildParade`. **Rejected:** global `ShowClosed` / `c` — cannot express nested collapse and recreates a Done ghetto. **Rejected:** storing collapse in Beads — UI state belongs to the running lens, not the ledger.

**Sort — chosen:** graph structure first, priority then stable ID among siblings; remove `UpdatedAt` from `SortIssues`. **Rejected:** preserving recency as an implicit tie-break — directly violates the Brief. **Rejected:** adding a sort-mode preference — unrequested.

**Mouse hit testing — chosen:** deterministic geometry plus row metadata. The left cache is already one row per `ParadeItem`; detail records line-to-ID while rendering. **Rejected:** ANSI-text regex parsing — fragile under styling and truncation. **Rejected:** a zone/dependency library — unnecessary. **Rejected:** a new right-pane link widget — forbidden by the Brief.

**Operator Review storage — recommended but not closed:** declare exact custom status `review` in `.beads/policy.yaml`, preserving the built-in workflow statuses used by this board, keep `review` out of `workflow.status_groups.ready`, and transition explicitly with `br update <id> --status review`. The live board proves Beads does not auto-enter it when children close. A narrowly named compatibility rule may read old converged epics until migration. **Rejected:** permanent settled-descendant paint over `in_progress` — the Brief explicitly calls that a category error. **Rejected:** a label convention without operator approval — weaker than the preferred real status and not selected by the Brief.

**Glyphs — settled:** `○ ● ⊘ ⏸ ◐ ✓` in exact state order. Removes `♪`. No carnival theme.

## Open questions — settled 2026-09-17 (Ata: "yeah sounds good")

1. **Storage.** Real beads status `review` in `.beads/policy.yaml`, written via `br update`. Forbidden: keep `in_progress` and only paint. Named temporary shim `legacyConvergedEpicOperatorReview` is allowed for already-converged epics until migrated.
2. **What flips it.** Explicit lead action (`br update <id> --status review`). Not auto-enter when children close.
3. **Right-pane clicks.** Loaded DEPENDENCIES rows (blocking, resolved, non-blocking including parent, reverse `blocks`) plus a cross-rig ID only if that exact ID is locally loaded. Missing/unloaded visible, not clickable. No children widget. No molecule IDs.
4. **Glyphs.** Ready `○` Working `●` Waiting/Blocked `⊘` Deferred `⏸` Operator Review `◐` Done `✓`.

**Hover-scroll (same turn, Task 6):** wheel over a pane scrolls that pane without moving focus. Click still moves focus.
