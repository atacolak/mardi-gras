# Semantic Execution State + Cockpit Foundations Technical Spec

**Design Brief:** `/tmp/2026-09-16-semantic-state-brief.md` ("Design authority: semantic execution state + cockpit foundations", Operator: Ata, 2026-09-16). The same text is the `Asked:` body of campaign epic `mard-r43` (verified via `br show mard-r43`), which is what makes it the settled authority for this wave.
**Status:** blocked on operator — 9 of 13 raw-status mapping rows are fully determined by the Brief; 4 (`draft`, `tombstone`, `pinned`, custom project statuses) cannot be assigned to one of Ata's six states without inventing product intent. See [Open questions](#open-questions). Everything else below is sound and plannable.
**Repo:** `/home/sf/workspace/mardi-gras` (branch `feat/operator-observability`, HEAD `fc2d3ff`)

**Campaign:** new epic `mard-r43`. The frozen prior campaign (`mard-nob`, `docs/plans/2026-09-15-operator-observability-spec.md` / `-plan.md`) is **not** widened by this spec, and its epic body is not edited.

## Intent (from the Brief — do not rewrite)

Quoted invariants that bind this spec:

- "mg must derive/render **semantic execution state**, not raw Beads status, and must not collapse unknown statuses into Lined Up."
- "Required semantic states (exact names): 1. Ready 2. Working 3. Waiting/Blocked 4. Deferred 5. Awaiting Review 6. Done"
- "Classification uses the **dependency / parent-child graph**, not raw status alone."
- "Hard example Ata named: an epic with all executable descendants settled, no live work, and operator acceptance pending must render **Awaiting Review** — NOT `Rolling (in_progress) 1/1 (100%)`. That is the current `mard-nob` defect on the live board."
- "Do **not** collapse `deferred` / `blocked` / `draft` / `pinned` / `tombstone` / custom project statuses into Lined Up."
- Cockpit foundations, this wave: "parent-child tree is actual hierarchy (parent-child dep edge, not dotted-ID paint)"; "render compact relative IDs under an epic rather than the full long id when the prefix is redundant"; "ability to scope/filter a view to one owned epic subtree (sibling cockpits)".
- "Ata will design click/collapse/color/layout later. Do **not** implement that redesign in this wave."
- "Dogfood leftover (in scope only if it blocks the above)."

Brief non-goals, carried verbatim: Gas Town / Gas City rewrite; actor control verbs; mutation-plane work; landing/merging `mard-nob`; installing over `/home/sf/.local/bin/mg` unless the operator asks.

## Mapping onto the current system

### Observed (verified live against this host and repo, 2026-09-16)

Defect surface:

- `internal/data/issue.go:45-53` defines `ParadeStatus` with exactly four groups: `ParadeRolling` (in_progress), `ParadeLinedUp` (open, not blocked), `ParadeStalled` (open, blocked), `ParadePastTheStand` (closed).
- `internal/data/issue.go:226-243` `Issue.ParadeGroup` switches on `i.Status` with only three arms (`StatusClosed`, `StatusInProgress`, `StatusOpen`) and a `default: return ParadeLinedUp`. `internal/data/issue.go:14-18` declares only three `Status` constants (`open`, `in_progress`, `closed`).
- Therefore every other Beads status reaches the `default` arm and lands in Lined Up — precisely the collapse the Brief forbids.
- `internal/views/detail.go:911-925` `paradeLabel` repeats the same three-arm switch and returns the literals `"Past the Stand"`, `"Stalled"`, `"Rolling"`, `"Lined Up"`.
- `internal/views/detail.go:223-225` renders the Status row as `<sym> <paradeLabel> (<raw status>)` — the `Rolling (in_progress)` half of Ata's hard example.
- `internal/views/detail.go:884-908` `epicProgress` counts children strictly by `ParentRelationshipID()` (the parent-child edge), and `detail.go:231-237` renders `issueProgress.Label()` = `"%d/%d (%d%%)"` — the `1/1 (100%)` half.

Authoritative status vocabulary (this is the fact that sizes the mapping table):

- `br` 0.5.11 (`br --version`). `br schema issue` gives `$defs.Status` as `anyOf [ enum[open, in_progress, blocked, deferred, draft, closed, tombstone, pinned], string ]`. The trailing bare `string` arm means the enum is **open**: custom statuses are legal.
- `br list --status bogus` errors with: "Built-in statuses: open, in_progress, blocked, deferred, draft, closed, tombstone, pinned. Custom statuses must be declared in .beads/policy.yaml (workflow.statuses) or exist on at least one issue."
- `br update --help`: "Terminal states (`closed`, `tombstone`) are refused — use the dedicated `br close` / `br delete` commands". `br delete --help`: "Delete an issue (creates tombstone)", `--hard` "Prune tombstones from JSONL immediately" — so un-pruned tombstones do reach mg's JSONL source.
- mg therefore recognises 3 of 8 built-in statuses and none of the custom space. `.beads/policy.yaml` does not exist in this repo, so no custom statuses are declared here today (inferred from absence).
- `br schema issue` gives `$defs.IssueType` as likewise open-ended: `enum[task, bug, feature, epic, chore, docs, question]` + bare `string`. mg's `IssueType` constants (`issue.go:23-32`) list `spike`/`story`/`milestone` (absent from br's enum) and omit `docs`/`question` (present in it — `mard-nob.6` is `issue_type: docs` on the live board). Outside Brief scope; recorded as a leftover, not planned.

Tool corroboration for the Awaiting Review rule — the Brief's condition is not novel, `br` already computes it:

- `br stats` on this board reports `Epics ready to close: 1`.
- `br epic status` reports `mard-nob … Progress: 7/7 children closed (100%) / Eligible for closure`, and `mard-r43 … 0/0 children closed (0%)` with no eligibility. `br epic --help` documents `close-eligible` as "Close epics that are eligible (all children closed)".
- `br ready --help`: "List ready issues (open, unblocked, not deferred)" — an exact match for the Brief's "Ready", including the defer clause.

Live `mard-nob` shape (`.beads/issues.jsonl` and `br show mard-nob`):

- `mard-nob` — `in_progress`, `issue_type: epic`, no dependencies, `Rollup: closed (7 closed)`.
- `mard-nob.1` through `.7` — all `closed`, each carrying exactly one `{"type":"parent-child","depends_on_id":"mard-nob"}` edge. `.6` is `issue_type: docs`; the rest are `task`.
- No issue on this board carries `defer_until`, and no issue has a status outside `{open, in_progress, closed}`.

**Correction to the Brief's transcription of the defect.** The Brief says the defect renders `Rolling (in_progress) 1/1 (100%)`. The `Rolling (in_progress)` label is real and reproducible. The counter is not `1/1`: all seven children carry parent-child edges, so `epicProgress` yields `7/7 (100%)` over the JSONL source, and over the live `br list` source it yields **no Progress row at all** (see the edge-fidelity finding below). This spec therefore pins the *label* invariant, which is what the Brief actually legislates, and does not encode `1/1`.

Existing hierarchy machinery to reuse rather than reinvent:

- `internal/data/issue.go:283-290` `ParentRelationshipID()` — first `parent-child` dep edge, `""` if none.
- `internal/data/issue.go:295-310` `ParentRelationshipDepth()` — cycle-safe and missing-parent-safe ancestor walk.
- `internal/data/issue.go:400-450` `OrderHierarchically()` — per-section depth-first child-under-parent ordering, cycle-safe, deliberately section-relative (a child whose parent is in another section renders at depth 0). This is why the packet's complaint "parade still groups by raw status first, so a child does not sit under its parent across buckets" is true: the ordering is correct, the *bucketing* is what splits families.
- `internal/data/issue.go:126-171` `EvaluateDependencies()` — the single canonical blocking evaluator; `IsBlocked` is true for unresolved **and** dangling blockers (`issue.go:164`).
- `internal/data/issue.go:317-320` `IsDeferred()` — `DeferUntil != nil && DeferUntil.After(now)`.

Every consumer of the grouping type (the full cutover surface):

- `internal/app/app.go:58` `groups map[data.ParadeStatus][]data.Issue`; recomputed at `app.go:288` (`NewWithGuard`), `app.go:1235` (`FileChangedMsg`), `app.go:1544` (`CLIHealthCheckMsg`), `app.go:3474` (`rebuildParade`, only when a filter/focus/exclude is active).
- `internal/views/parade.go:20-38` `paradeSection` + `sections()` (four titles, symbols, styles, colors, pre-rendered borders); `parade.go:65` `Parade.Groups`; `parade.go:124-173` `rebuildItems`; `parade.go:380` header count; `parade.go:369-376` `renderLegend` (four hard-coded legend labels). The three-arm raw-status switch is copied **three** further times inside `views`, and all three must cut over or the row glyph will disagree with the section it sits in: `parade.go:459-475` (inline `symStr` in `renderIssue`), `parade.go:719-734` `statusSymbol`, `parade.go:736-751` `statusColor`. `statusSymbol` is pinned by `TestStatusSymbol` (`render_test.go:36-76`) and `statusColor` by `TestStatusColor` (`render_test.go:78-118`); `detail.go:222-225` calls all three of `statusSymbol`, `paradeLabel`, `statusColor`, so the detail Status row inherits the cutover rather than needing its own switch.
- `internal/components/header.go:19` `Header.Groups` and `header.go:30-42` four counts rendered as `" %d●  %d♪  %d⊘  %d✓ "`.
- `internal/tmux/status.go:12-37` four `colour*` constants and four counts.
- `cmd/mg/main.go:125-126` the `--status` one-shot path: `data.GroupByParade(visible, …)` then `tmux.StatusLine(groups)`.
- `internal/data/loader.go:62-76` `GroupByParade`.
- ID rendering: `internal/views/parade.go:148` and `parade.go:164` set `RenderedID: idStyle.Render(iss.ID)` — always the full ID.

Filter/scope pipeline (where subtree scoping must plug in):

- `internal/app/app.go:3465-3477` `rebuildParade()` runs `FilterIssuesWithHighlights(m.issues, m.filterInput.Value())`, then `ExcludeByType`/`ExcludeByLabel`, then `FocusFilter` (if `m.focusMode`), then `GroupByParade`. The `if` at `app.go:3474` decides whether to re-group at all, so a new scope must be added to that condition or the parade will keep unscoped groups.
- `internal/data/focus.go:13-60` `FocusFilter` is a *priority queue* (my in-progress work, top 5 unblocked, top 3 blocked). It performs no tree traversal and is **not** a subtree scope; it cannot serve the Brief's "scope to one owned epic subtree".
- Bound keys in `handleKey` include `q ? / tab esc f ctrl+g o p D M c 1 2 3 ! @ # $ b B a A s n N e r y t l C ctrl+k : j k J K space x X g G d w W R v h`. `E` is unbound (observed); `f`/`esc` is the established toggle-and-clear precedent.

UI vocabulary to extend:

- `internal/ui/symbols.go:13-16` `SymRolling`/`SymLinedUp`/`SymStalled`/`SymPassed` = `● ♪ ⊘ ✓`; `symbols.go:48` `SymDeferred = "⏸"`.
- `internal/ui/styles.go:25-34` `SectionRolling|LinedUp|Stalled|Passed` and `StatusRollingStr|LinedUpStr|StalledStr|PassedStr`, defined at `styles.go:190-210`.
- `internal/ui/theme.go:73-77` per-theme colors `StatusRolling|LinedUp|Stalled|Passed|Agent`; `ui.Orange` already exists as a theme primitive.
- `ui.SymWorking` (`symbols.go:72`) is already taken by the Gas Town pane, so the new state symbols need a distinct prefix.

### Contradiction check: the Brief's graph requirement vs the live source

**Observed:** mg fetches via `internal/data/source.go:84-99`, running `br list --json --limit 0 --all`. That output carries **no `dependencies` array at all** — only the scalars `dependency_count` / `dependent_count`. Verified field list for `mard-nob.7` from that exact command: `assignee, close_reason, closed_at, compaction_level, created_at, created_by, dependency_count, dependent_count, description, id, issue_type, original_size, priority, source_repo, source_repo_path, status, title, updated_at`.

Consequence on the operator's live path (`./mg` with no `--path`): `ParentRelationshipID()` returns `""` for every issue, so there is no hierarchy, nothing is ever blocked, `epicProgress` finds zero children, and **the Brief's hard example cannot fire** — `mard-nob` would render Working, not Awaiting Review. Graph-derived state over `br list --json` is inert without a fix.

This is the Brief's "Dogfood leftover (in scope only if it blocks the above)" clause firing: it blocks the above, so it is in scope. It is **not** a product question, because every candidate fix produces identical operator-visible state; they differ only in latency and freshness. Options observed:

- `br dep list <id> --json` returns exactly mg's `Dependency` shape (`{"issue_id","depends_on_id","type",…}`) but is per-issue — N subprocesses against a 15s budget.
- `br show <id> --json` includes `dependencies`, but in an **incompatible** shape (`{"id":…,"dependency_type":"parent-child"}` — neither `depends_on_id` nor `type`), and is also per-issue.
- `.beads/issues.jsonl` — the export `br where` reports, kept current by br's auto-flush (`--no-auto-flush` is the opt-out) — carries full `dependencies` arrays. Verified.
- Dotted-ID inference is **forbidden** by the Brief ("parent-child dep edge, not dotted-ID paint") and by the existing code comment at `issue.go:278-282`.

Chosen internal: merge edges from the co-located JSONL export onto CLI-sourced issues. See [Implementation approach chosen](#implementation-approach-chosen-and-rejected-internals).

### Contradiction check: everything else

No other product-shaped gap. The six states, the epic rule, hierarchy-by-edge, relative IDs, and subtree scoping are all satisfiable by extending existing seams. Nothing in the Brief requires weakening a current behavior, and the only current behavior the Brief overrides is the four-bucket vocabulary it explicitly replaces.

## Architecture

One derived-state model in `internal/data`, computed from the issue graph, consumed by every render surface through a single ordered vocabulary.

**1. Derivation (`internal/data/semantic.go`, new).** `SemanticState` replaces `ParadeStatus`. `DeriveState(issue, issueMap, blockingTypes) (SemanticState, bool)` is the one classifier; the boolean reports whether the raw status is one this wave is authorised to classify. `GroupBySemanticState` replaces `GroupByParade`. `Issue.ParadeGroup` and `ParadeStatus` are deleted so no caller can retain four-bucket semantics.

**2. Hierarchy primitives (`internal/data/hierarchy.go`, new).** `Descendants`, `ChildrenOf`, `EpicAncestor`, `ScopeToSubtree`, `RelativeDisplayID` — the transitive parent-child-edge walkers that the epic rule, the subtree scope, and the compact IDs all need. Cycle-safety and missing-parent-safety follow `ParentRelationshipDepth`/`OrderHierarchically` exactly.

**3. Edge fidelity (`internal/data/source.go`, modify).** CLI-sourced issues get their `Dependencies` back-filled from the co-located JSONL export before they reach any consumer, so the graph rules are live on the operator's default path and not only under `--path`.

**4. Vocabulary (`internal/ui`, modify).** Six symbols, six section styles, six pre-rendered indicators, two new theme colors — built from primitives that already exist in the theme, because the Brief defers color and layout redesign to Ata's later wave.

**5. Render cutover (`views/parade.go`, `components/header.go`, `tmux/status.go`, `views/detail.go`, `cmd/mg/main.go`, `app/app.go`).** Every surface iterates `data.StateOrder()` instead of hard-coding four constants — the per-surface constant lists are how header and tmux can drift from the parade today.

**6. Cockpit foundations (`views/parade.go`, `app/app.go`).** Compact relative IDs on nested rows; an epic-subtree scope on key `E`, cleared by `esc`, threaded through `rebuildParade()`.

Section order is derived by rule, not taste: the four existing sections keep their current relative order, and each new section is inserted immediately after the section it derives from (Awaiting Review derives from `in_progress`, so it follows Working; Deferred derives from `open`, so it follows Ready). Result: **Working, Awaiting Review, Ready, Deferred, Waiting/Blocked, Done**. The Brief assigns layout to Ata's later wave, so this is a placement rule, not an information-architecture proposal.

## Components and interfaces

### `internal/data/semantic.go` (new)

```go
// SemanticState is the operator-facing execution state mg derives from the
// issue graph. It is not a Beads status; see DeriveState.
type SemanticState int

// Order is render order: the four pre-existing parade sections keep their
// relative order, and each new state follows the state it derives from.
const (
    StateWorking SemanticState = iota
    StateAwaitingReview
    StateReady
    StateDeferred
    StateWaitingBlocked
    StateDone
)

// Label returns Ata's exact state name.
func (s SemanticState) Label() string // "Working" | "Awaiting Review" | "Ready" | "Deferred" | "Waiting/Blocked" | "Done"

// StateOrder is the single source of render order for every surface.
func StateOrder() []SemanticState

// DeriveState classifies one issue from the dependency / parent-child graph.
// ok is false when the raw status is outside the set this wave is authorised
// to map (draft, tombstone, pinned, and any custom status): such an issue is
// never silently bucketed, and in particular never becomes StateReady.
func DeriveState(i *Issue, issueMap map[string]*Issue, blockingTypes map[string]bool) (state SemanticState, ok bool)

// GroupBySemanticState buckets issues by derived state. Issues DeriveState
// cannot classify are excluded from every bucket and returned separately.
func GroupBySemanticState(issues []Issue, blockingTypes map[string]bool) (groups map[SemanticState][]Issue, unmapped []Issue)

// EpicAwaitingReview reports the Brief's settled-descendants rule. decidable
// is false when a descendant carries a status DeriveState cannot classify, so
// the epic's own state cannot be honestly computed either.
func EpicAwaitingReview(i *Issue, issueMap map[string]*Issue) (awaiting, decidable bool)
```

New `Status` constants recognised alongside the existing three (recognising a status is not the same as deciding its state):

```go
const (
    StatusBlocked   Status = "blocked"
    StatusDeferred  Status = "deferred"
    StatusDraft     Status = "draft"
    StatusTombstone Status = "tombstone"
    StatusPinned    Status = "pinned"
)
```

### Mapping table (the contract DeriveState implements)

Precedence is top to bottom; the first matching row wins.

| # | Raw status | Graph / field condition | Semantic state | Authority |
|---|---|---|---|---|
| 1 | `closed` | — | **Done** | Brief state 6; br terminal state |
| 2 | `draft` | — | **unmapped** (`ok=false`) | Brief forbids collapsing it; which of the six is [Open question 1](#open-questions) |
| 3 | `tombstone` | — | **unmapped** (`ok=false`) | [Open question 2](#open-questions) |
| 4 | `pinned` | — | **unmapped** (`ok=false`) | [Open question 3](#open-questions) |
| 5 | any status not named in another row | — | **unmapped** (`ok=false`) | Custom statuses; [Open question 4](#open-questions) |
| 6 | any mapped status | `EvaluateDependencies(...).IsBlocked` — unresolved or dangling blocker | **Waiting/Blocked** | Brief state 3. Blocked-wins is the existing documented rule at `issue.go:225` ("Stalled wins over Rolling") |
| 7 | `blocked` | — | **Waiting/Blocked** | Name identity; Brief forbids collapsing it |
| 8 | not `closed` | `IssueType == epic` AND at least one loaded descendant AND every executable descendant `closed` | **Awaiting Review** | Brief's hard example; corroborated by `br epic status` "Eligible for closure" |
| 9 | `deferred` | — | **Deferred** | Name identity; Brief forbids collapsing it |
| 10 | any mapped status | `DeferUntil` in the future | **Deferred** | Brief state 4; `br ready` = "open, unblocked, **not deferred**" |
| 11 | `in_progress` | — | **Working** | Brief state 2 |
| 12 | `open` | — | **Ready** | Brief state 1; `br ready` = "open, unblocked, not deferred" |
| 13 | any mapped status | epic whose descendants include an unmapped status | **unmapped** (`ok=false`) | Undecidable until Open questions 1-4 are answered |

Notes that are contract, not commentary:

- Rows 2-5 are checked before every bucket, so an unmapped status can never leak into Ready. This is the Brief's "do not collapse" invariant, expressed as an early return rather than a `default:` arm.
- Row 6 before row 8: an epic that is itself blocked by an external dependency reports Waiting/Blocked, not Awaiting Review. `mard-nob` has no dependencies, so the hard example is unaffected. This keeps blocked-wins uniform instead of special-casing epics.
- Rows 6-7 before rows 9-10: an issue that is both blocked and deferred reports Waiting/Blocked. This is non-lossy — `parade.go:589-595` already renders the `⏸` deferred badge on the row regardless of bucket, so deferral stays visible.
- Row 8's "no live work" clause from the Brief is implied by "every executable descendant closed", and is asserted separately in tests rather than encoded twice.
- Row 8 applies to any epic that is not itself Done, not only `in_progress` ones. The Brief conditions the rule on the descendants and on "operator acceptance pending", never on the epic's own raw status, and the point of the wave is to stop deriving from raw status alone. An `open` epic with all children closed is equally awaiting acceptance.
- Row 8 requires at least one loaded descendant. `br epic status` shows `mard-r43` at `0/0` and withholds eligibility, so an empty epic is not "settled" — this follows the tool rather than inventing.
- "Executable descendant" means a descendant whose raw status is in the mapped set. A descendant with an unmapped status makes the epic undecidable (row 13), because whether it counts as settled is exactly Open questions 1-4.

### `internal/data/hierarchy.go` (new)

```go
// ChildrenOf returns the issues naming parentID through a parent-child edge.
func ChildrenOf(parentID string, issues []Issue) []*Issue

// Descendants returns every transitive parent-child descendant of rootID,
// excluding the root. Cycle-safe and missing-parent-safe, like
// ParentRelationshipDepth.
func Descendants(rootID string, issueMap map[string]*Issue) []*Issue

// EpicAncestor returns i when i is an epic, else its nearest ancestor of type
// epic, else nil. Walks parent-child edges only.
func EpicAncestor(i *Issue, issueMap map[string]*Issue) *Issue

// ScopeToSubtree returns rootID plus its descendants, preserving input order.
// An unknown rootID yields nil, so a stale scope empties the view instead of
// silently showing everything.
func ScopeToSubtree(issues []Issue, rootID string) []Issue

// RelativeDisplayID compacts an ID whose dotted prefix is redundant because
// its parent-child parent is that prefix: "mard-nob.7" under "mard-nob" gives
// ".7". Returns the full ID when the prefix is not redundant — a reparented
// issue keeps its old dotted ID, and an edge-only child has no dotted prefix.
// It needs no issueMap: the comparison is between this issue's own dotted
// prefix and its own parent-child edge.
func RelativeDisplayID(i *Issue) string
```

### `internal/data/source.go` (modify)

```go
// MergeGraphEdges back-fills Dependencies on issues from a CLI source, which
// `br list --json` omits entirely (it returns only dependency_count).
func MergeGraphEdges(issues []Issue, projectDir string) []Issue
```

Reads `filepath.Join(ResolveBeadsDir(filepath.Join(projectDir, ".beads")), "issues.jsonl")` with the existing tolerant `LoadIssues`, and copies `Dependencies` onto matching IDs. A missing or unreadable export is not an error: issues pass through edge-less, exactly as today. Called from `FetchIssuesCLI` (`source.go:86-95`) so every poll path — `watcher.go` `PollCLI`, `CLIHealthCheck`, `source.go:317` `FetchIssuesNow`, and the `cmd/mg/main.go:99` startup fetch — inherits it without further threading.

### `internal/ui` (modify)

A new six-name `Exec*` vocabulary is **added**; the existing palette primitives are not renamed. Two observed facts force this. First, `ui.StateWorking` (`theme.go:115`) and `ui.SymWorking` (`symbols.go:72`) are already taken by Gas Town agent states, so a `State*` prefix would collide on the exact word "Working". Second, `StatusStalled` and `StatusPassed` are used well beyond the parade as a general alert/muted palette — `views/doctor.go:96,128`, `views/problems.go:119,161,163,217`, `views/actors.go:134`, `components/footer.go:180`, and `ui/gradient.go:203-204` — so renaming them would churn files this feature does not touch, without making any of those call sites more correct.

`symbols.go` — six glyphs, reusing existing ones so no aesthetic redesign lands:

```go
SymExecWorking        = "●" // same glyph as SymRolling
SymExecAwaitingReview = "◐" // half-filled: work complete, acceptance pending
SymExecReady          = "♪" // same glyph as SymLinedUp
SymExecDeferred       = "⏸" // same glyph as SymDeferred
SymExecWaiting        = "⊘" // same glyph as SymStalled
SymExecDone           = "✓" // same glyph as SymPassed
```

`theme.go` — six color vars (`ExecWorking`, `ExecAwaitingReview`, `ExecReady`, `ExecDeferred`, `ExecWaiting`, `ExecDone`) assigned in `applyDerived()` (`theme.go:136-166`), which is the existing home for "colors defined in terms of other palette entries the same way in both themes". Every one binds to a primitive that already exists in both palettes, so neither `applyDarkPalette` nor `applyLightPalette` changes: `BrightGreen`, `Orange`, `BrightGold`, `Dim`, `StatusStalled`, `Muted` respectively.

`styles.go` — six `SectionExec*` styles and six `Exec*Str` pre-rendered indicators. The four parade-only `Section{Rolling,LinedUp,Stalled,Passed}` styles and four `Status*Str` indicators they replace are deleted, since `parade.go` is their only consumer (verified: `styles.go:190-210` defines them, `parade.go` alone reads them).

**The four legacy `Sym*` glyph constants must survive.** `SymRolling`/`SymLinedUp`/`SymStalled`/`SymPassed` are read by nine call sites that have nothing to do with execution state: `detail.go:381,418,951` (dependency-direction glyphs), `doctor.go:127`, `problems.go:161`, `gastown.go:1041`, `codex_transcript.go:152,156,219`, and `header.go:174` (the progress-bar `%` label). Only their parade/tmux/status-row uses are cut over to `SymExec*`; deleting the constants is out of scope and would break six unrelated files. `symbols.go:32` `SymResolved` already documents this aliasing habit.

### `internal/views/parade.go` (modify)

`paradeSection.Status` becomes `State data.SemanticState`; `sections()` is built by iterating `data.StateOrder()` rather than hard-coding rows. `Parade.Groups` is keyed by `data.SemanticState`. All three remaining copies of the raw-status switch become derived-state lookups: the inline `symStr` switch in `renderIssue` (`parade.go:459-475`), `statusSymbol` (`parade.go:719-734`), and `statusColor` (`parade.go:736-751`). `statusSymbol`/`statusColor` keep their exported-to-the-package signatures but take the derived state, which is what makes `detail.go:222-225` correct for free. `RenderedID` uses `data.RelativeDisplayID` when `item.Depth > 0` — the parent is then on screen in the same section, so the relative form is readable — and the full ID otherwise.

`Parade` gains `Unmapped []data.Issue` so unclassifiable rows are carried rather than dropped on the floor. Rendering them is [Open questions 1-4](#open-questions) and is not implemented in this wave.

### `internal/app/app.go` (modify)

`Model.groups` becomes `map[data.SemanticState][]data.Issue`. New field `scopeRootID string`. `rebuildParade()` applies `data.ScopeToSubtree(filteredIssues, m.scopeRootID)` after the fuzzy/exclude/focus passes, and `m.scopeRootID != ""` joins the re-group condition at `app.go:3474`. Key `E` sets `scopeRootID` from `data.EpicAncestor(selected)`; `esc` clears it alongside the existing filter/focus clears; the footer shows a scope chip.

## Data / control flow

Load: `FetchIssuesCLI` runs `br list --json --limit 0 --all`, parses the envelope, then **`MergeGraphEdges` (new)**, then `validateIssuePrefixes` and `SortIssues`. The `--path` JSONL path already carries edges and is untouched.

Group: `m.issues` through fuzzy filter + highlights, exclude type/label, focus filter, **`ScopeToSubtree` (new)**, then `GroupBySemanticState` returning `(groups, unmapped)`.

Render: `sections()` iterates `StateOrder()`; each section's rows are ordered by the existing `OrderHierarchically`, so children now sit under parents *within a semantically coherent bucket*. The cross-bucket family split disappears for families that share a state and remains only where members genuinely differ in state, which is what the buckets are for. Per row: derived state selects symbol and style; `RelativeDisplayID` applies when depth > 0.

Detail: selected issue, `DeriveState`, `semanticLabel(state)`, Status row `<sym> <label> (<raw status>)`. Keeping the raw status in parentheses is existing behavior and preserves "derive, don't hide": `mard-nob` reads `◐ Awaiting Review (in_progress)`.

Headless: `cmd/mg/main.go --status` runs `GroupBySemanticState` then `tmux.StatusLine` with six counts.

## Error handling

- Missing or unreadable `.beads/issues.jsonl` during `MergeGraphEdges`: issues pass through unmodified, no error surfaced. Precedent: `loader.go:31-35` skips malformed JSONL lines rather than aborting, and `AGENTS.md`'s "every feature must work or hide gracefully at each level".
- A stale JSONL export yields stale edges for at most one br auto-flush cycle. Recorded as a risk, not handled: mg is a stateless presentation layer and must not write to the board.
- Dangling blockers stay Waiting/Blocked rather than becoming a new error state — `EvaluateDependencies` already treats `DepMissing` as blocking (`issue.go:164`), and `TestParadeGroup_DanglingDep` (`loader_test.go:210-228`) pins it.
- Parent-child cycles: `Descendants` and `ScopeToSubtree` visit each ID once, mirroring `OrderHierarchically`'s guarantee that "an issue reachable only through a cycle is emitted as a root so it can never disappear".
- A missing parent means no compaction and no indent, never an indent under an unrelated row (existing invariant, `issue.go:420-423`).
- `scopeRootID` pointing at an issue that has since vanished yields an empty parade with the scope chip still lit, so the operator can see *why* it is empty and press `esc`. Silently widening back to everything would misreport the board.
- Unmapped statuses are excluded from every bucket and never rendered as Ready. Ata's answer decides what they do render as.

## Testing (behavioral contracts; exact tests live in the plan)

**The hard example is an acceptance test, at two levels.**

1. `internal/data`: an epic that is `in_progress`, `issue_type: epic`, with seven `closed` children each carrying a `parent-child` edge — the exact `mard-nob` shape — derives `StateAwaitingReview`, and explicitly **not** `StateWorking`.
2. End-to-end through the built binary: `mg --status --path testdata/mard-nob-awaiting-review.jsonl` prints an Awaiting Review count of 1 and a Working count of 0. This exercises loader, derivation, grouping, and tmux render in one process, so it cannot be satisfied by editing a unit test's expectation.

Other contracts:

- Every one of the eight built-in br statuses parses into `data.Issue` (extends `TestContractAllStatusValues`, `contract_test.go:564-581`, which currently covers three).
- `draft`, `tombstone`, `pinned`, and an arbitrary custom status each return `ok=false` from `DeriveState` and appear in no bucket — in particular not `StateReady`. This is the Brief's "do not collapse" invariant as a negative test, and it holds without deciding the open questions.
- Blocked-wins: an `in_progress` issue with an unresolved blocker is Waiting/Blocked; with a dangling blocker likewise (ports `TestParadeGroup_InProgressBlocked` and `TestParadeGroup_DanglingDep`).
- An epic with zero loaded descendants is not Awaiting Review; an epic with one open descendant is not Awaiting Review; an epic with a `draft` descendant is undecidable.
- Deferred: `defer_until` in the future is Deferred, in the past is Ready; raw status `deferred` is Deferred.
- `ScopeToSubtree` returns root plus transitive descendants only, is cycle-safe, and returns nil for an unknown root.
- `RelativeDisplayID`: `mard-nob.7` with parent `mard-nob` gives `.7`; a reparented issue keeping an unrelated dotted ID gives the full ID; an edge-only child with no dotted prefix gives the full ID.
- `MergeGraphEdges`: a CLI payload with no `dependencies` plus an export that has them yields issues with edges; a missing export yields the input unchanged.
- Parade renders all six section titles when all six states are populated, and `⏸`/`◐` rows are not in the Ready section.
- Header and tmux render six counts; existing count fixtures are re-derived from `testdata/sample.jsonl` rather than copied forward.
- Existing indent contracts still hold: `TestParadeIndentUsesParentRelationships` (`parade_test.go:242-264`) and the depth-2 indent case (`render_test.go:505-535`).
- Fixture facts that size the above (observed): `testdata/` holds only `sample.jsonl` and `screenshot.jsonl`. `sample.jsonl` is 14 `open` / 4 `in_progress` / 3 `closed`, carries **zero** `parent-child` edges, has no `defer_until` on any issue, and its single `epic` has no children. So the epic rule cannot fire there — re-derived counts (`loader_test.go:35-59`, `tmux/status_test.go:26-51`) change only because four buckets become six, never because an epic flips state. The hard example therefore requires a new fixture file rather than an edit to `sample.jsonl`, which keeps the existing count contracts readable.

## Non-goals

- Gas Town / Gas City rewrite; actor control verbs; mutation-plane work; landing or merging `mard-nob`; installing over `/home/sf/.local/bin/mg`. (Brief.)
- Click, collapse, color, and layout redesign — explicitly Ata's later wave. Section order here is a placement rule, not a design.
- Reopening, widening, or editing `mard-nob`, its epic body, or its frozen spec/plan.
- Rewriting the actor CLI or changing mg's village timeout. (Brief's dogfood section; it does not block this wave.)
- Syncing mg's `IssueType` constants with br's enum (`docs`/`question` missing, `spike`/`story`/`milestone` absent from br). Real and observed, but outside Brief scope — leftover, not a requirement.
- Writing to the board to obtain edges. mg stays read-only; the JSONL export is read, never produced.
- Rendering for unmapped statuses. Gated on Ata's answer.

## Implementation approach chosen (and rejected internals)

**Edge fidelity — chosen:** read the co-located `.beads/issues.jsonl` export and merge `Dependencies` onto CLI-sourced issues. One file read, reuses `LoadIssues` and `ResolveBeadsDir`, and the export is the same file mg already falls back to when no CLI is on PATH. *Rejected:* N calls to `br dep list <id> --json` — correct shape, but one subprocess per issue against a 15s timeout while mg polls every 5s. *Rejected:* N calls to `br show <id> --json` — same fan-out plus an incompatible dependency shape (`dependency_type`/`id` instead of `type`/`depends_on_id`), so it would need a second parser. *Rejected:* inferring hierarchy from dotted IDs — forbidden by the Brief and by `issue.go:278-282`. *Rejected:* invoking `br sync` to freshen the export — that is a write, and mg is a presentation layer.

**Derivation shape — chosen:** one `DeriveState` returning `(state, ok)`. The boolean is what lets this wave ship the determined rows while making the four undetermined statuses a compile-visible, test-pinned hole instead of a silent default. *Rejected:* a `StateUnknown` seventh constant — that is a seventh operator-visible state, which "Required semantic states (exact names)" does not authorise. *Rejected:* keeping a `default:` arm — that is the exact defect.

**Type replacement — chosen:** delete `ParadeStatus`, `Issue.ParadeGroup`, and `GroupByParade` rather than deprecating them, so the compiler enumerates every surface that must cut over and no caller can retain four-bucket semantics. *Rejected:* keeping `ParadeStatus` as an alias — leaves the collapsing `default` arm reachable.

**Ordering source — chosen:** `StateOrder()` as the single iteration source for parade, header, and tmux. Each of those three surfaces currently hard-codes its own list of four constants, which is how they drift. *Rejected:* per-surface constant lists (status quo).

**UI naming — chosen:** add a six-name `Exec*` family (`SymExec*` glyphs, `Exec*` colors, `SectionExec*` styles, `Exec*Str` indicators) and delete only the parade-only styles it replaces. The new colors are aliases of primitives that already exist in both palettes, so the Brief's deferral of color to Ata is honored and `applyDarkPalette`/`applyLightPalette` are untouched. *Rejected:* renaming the four `Status*` color fields into the state vocabulary — it reads cleaner in the parade but churns `doctor.go`, `problems.go`, `actors.go`, `footer.go`, and `gradient.go`, which use those colors as a general alert palette and have nothing to do with execution state. *Rejected:* a `State*` prefix — `ui.StateWorking` and `ui.SymWorking` already mean Gas Town agent states. *Rejected:* leaving `SymRolling` in place as the parade's "Working" glyph — the packet asks for a label cutover, and a glyph constant named for the old vocabulary is how the next wave reintroduces this bug. *Rejected:* new palette entries — the Brief defers color to Ata.

**Relative IDs — chosen:** compact only when the row's dotted prefix equals its actual parent-child parent, and only at `Depth > 0`, where that parent is on screen in the same section. Literal to "when the prefix is redundant", and it keeps the reparented-ID case (`issue.go:278-282`) rendering in full. *Rejected:* stripping the project prefix from every ID — loses identity in an unscoped view and is not what the Brief asked for. *Rejected:* compacting by string prefix without checking the edge — that is dotted-ID paint.

**Subtree scope — chosen:** a scope pass in `rebuildParade()` keyed by root ID, on the free key `E`, cleared by `esc`, following the `f`/`esc` focus-mode precedent. *Rejected:* extending `FocusFilter` — it is a priority queue with a fixed top-5/top-3 truncation, so it would silently drop subtree members. *Rejected:* a `--scope` CLI flag only — the Brief asks for the ability to scope a view, and sibling cockpits switch scope live.

## Open questions

All four affect operator-visible product and belong to Ata, not to the lead and not to the planner. They are why **Status** above is `blocked on operator` rather than `sound`. Each is isolated to one arm of `DeriveState` and to rendering for the rows that arm produces; nothing else in this spec depends on the answers, and no issue on this board carries any of these statuses today.

1. **`draft`** — which of the six states does a draft render as? It must not be Ready (the Brief forbids collapsing it, and a draft is not ready work). None of the six means "not yet authored". Deferred is the nearest fit but asserts "intentionally scheduled for later", which is a different claim. Options: Deferred, a badge over some other state, or hidden by default.
2. **`tombstone`** — a deleted record (`br delete` creates one; `--hard` prunes it from the JSONL, so un-pruned tombstones do reach mg). Done would assert the work was completed, which is false. Options: Done, a distinct rendering, or excluded from the parade entirely.
3. **`pinned`** — br models pinned as a *status*, mutually exclusive with `in_progress`, yet it reads like an orthogonal decoration. Because it occupies the status field there is no other status to derive from. Options: one of the six (which?), or a badge whose underlying state is derived from something else — and if so, from what?
4. **Custom statuses** (`.beads/policy.yaml` `workflow.statuses`, or any status present on an issue) — the enum is open (`br schema issue` `$defs.Status` ends in a bare `string`). What does an unrecognised status render as, given that the Brief forbids collapsing it into Ready and authorises exactly six state names? A seventh "Unknown" bucket would exceed the authorised vocabulary.

Settled internals, recorded here so they are not mistaken for open questions: section order (the placement rule in [Architecture](#architecture)); blocked-wins precedence over deferred and over the epic rule; the epic rule applying to any non-Done epic; the at-least-one-descendant requirement; scope key `E` and clear on `esc`; the `◐` glyph and the `Orange`/`Dim` bindings; the JSONL-export edge merge.
