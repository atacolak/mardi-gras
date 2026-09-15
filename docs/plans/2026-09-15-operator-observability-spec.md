# Operator Observability (Beads parade + actor society) Technical Spec

**Design Brief:** `/home/sf/worlds/personal/designs/oh-my-pi/actor-society-surface.md` (Status: approved). Campaign product intent for the mg side is carried by epic `mard-nob` ("Asked:" block, observed via `br list --json` on 2026-09-15) and the lead packet/recon (`/tmp/mard-nob-recon.md`, "Conservative product" decisions). These three sources agree; nothing below contradicts any of them.
**Status:** sound
**Repo:** `/home/sf/workspace/mardi-gras`

## Intent (from the Brief — do not rewrite)

Brief invariants that bind this spec:

- "V1 is read / observation only."
- "Beads remains work truth. HCOM remains transport."
- "Enrich it; do not create a parallel actor model, roster database, or state store."
- The shared projection (societyRow) answers: identity (name, role, project.id, kind), presence (lifecycle, readiness), ownership (current_sprint epic+title or null), activity capsule (session_title, current_step, todo, asked, now).

Campaign intent (epic `mard-nob` + lead packet):

- `mg` is the operator observability UI at this managed root (`/home/sf/workspace/mardi-gras`).
- The Beads parade stays the work-truth view.
- Persistent actors are presented via the existing `actor` CLI societyRow projection — never by scraping hcom/herdr.
- Read-only first: "Observability must not become a mutation control plane."
- Non-goals from the epic: replacing Gas Town/Gas City, owning Beads or actor state inside mg, village-wide mutation.

## Mapping onto the current system

Observed (verified 2026-09-15 against this host and repo):

- `cmd/mg/main.go:240` `resolveSource`: `--path` → SourceJSONL; `.beads/` dir + `bd` on PATH → SourceCLI; `.beads/issues.jsonl` → SourceJSONL fallback; else error exit.
- `internal/data/source.go:74` `FetchIssuesCLI` runs `bd list --json --limit 0 --all` and `parseIssuesCLIOutput` unmarshals a bare JSON **array**.
- This host: `br` 0.5.11 present, `bd` absent. `br list --json --limit 0 --all` returns a **wrapper** object `{"issues":[...],"total":...,"limit":...,"offset":...,"has_more":...}` (verified live). Wrapper issue fields match `data.Issue` (id, title, description, status, priority, issue_type, assignee, timestamps, …).
- `internal/data/loader.go:31-34` skips malformed JSONL lines (precedent for tolerant parsing).
- `internal/data/exec.go:43` `runWithTimeout` is a package var — the established test stub seam (`internal/data/mock_test.go:7`).
- `internal/data/watcher.go` `PollCLI` (5s) / `CLIHealthCheck` (15s) / `source.go:279` `FetchIssuesNow` all funnel into `FetchIssuesCLI`.
- `actor` CLI at `/home/sf/.local/bin/actor` (verified live):
  - `actor project <id-or-root> --json` → `{"ok":true,"project":{"root","id"},"count":N,"actors":[societyRow]}` (both id and filesystem root accepted).
  - `actor list --json` → `{"ok":true,"count":N,"projects":[{"project":{...},"actors":[societyRow]}]}`.
  - `actor status <name> --json` → `{"ok":true,"actor":societyRow}`.
  - societyRow fields (verified): `name, role, project{id}, kind, lifecycle, readiness, activity{session_title,current_step,todo,asked,now}, current_sprint{epic,title}|null`. `session_title`, `todo`, and `current_sprint` are observed null-able.
  - Observed latency: project ~1–3s, village list up to ~10s → background poll with timeout is mandatory.
- Zero `actor` types in this Go repo (grep). Gas Town pane (`internal/views/gastown.go`) is the progressive-hide pattern to copy: background `tea.Cmd` poll, never block Update, nil-safe before first fetch, feature hidden when its binary is absent (`AGENTS.md`: "Every feature must work or hide gracefully at each level").

Unknown / inferred:

- (inferred) `br` needs none of the `BD_*` env pinning in `bdChildEnv` — that helper already returns nil for non-`bd` binaries.
- (inferred) `actor` needs no custom child env; inherit parent.

## Architecture

Three independent slices, one campaign:

**(a) br-aware Beads source.** `data.Source` gains the selected CLI binary. `resolveSource` prefers `br` over `bd` when `.beads/` exists; JSONL fallback unchanged. `FetchIssuesCLI` dispatches parsing by binary: `br` → unwrap `.issues[]` from the wrapper; `bd` → existing array parse. All watchers/pollers thread the binary through. Footer label reports the real source (`br list` vs `bd list`).

**(b) `internal/actors` client.** New domain package (matches AGENTS.md domain-based boundaries; recon frontier t2). Read-only wrapper over `actor project|list|status --json`. Owns its own timeout/exec helper mirroring `internal/data/exec.go`'s pattern. Exposes `Available()` so the UI can hide the feature when the binary is missing. No control verbs exist in this package — restore/recycle/send/ensure are out of scope by design.

**(c) Actors view pane.** New `internal/views/actors.go` following the gastown pane pattern: app-owned background `tea.Cmd` poll with a single-flight gate (mirroring `gtPollInFlight`), nil-safe render before first fetch, error state that stops the spinner (mirroring `townStatusErr`), hidden entirely when `actors.Available()` is false. Default scope is the current project (`actor project <projectDir>`); village (`actor list`) is a secondary scope toggle inside the same pane — a second view, not a second model.

No hcom/herdr scraping. No new registry, scheduler, or Beads mirror. Gas Town pane untouched.

## Components and interfaces

### `internal/data` (modify)

```go
// source.go
const (
    CLIBd = "bd"
    CLIBr = "br"
)

type Source struct {
    Mode       SourceMode
    Path       string
    ProjectDir string
    Explicit   bool
    CLIBinary  string // CLIBd | CLIBr; empty unless Mode == SourceCLI
}

func (s Source) Label() string // "br list" | "bd list" | file basename

func FetchIssuesCLI(projectDir, binary string) ([]Issue, error)

// brListEnvelope mirrors `br list --json` wrapper output.
type brListEnvelope struct {
    Issues []Issue `json:"issues"`
}

func parseBrListOutput(out []byte, expectedPrefix string) ([]Issue, error)
```

`parseBrListOutput` reuses `validateIssuePrefixes` + `SortIssues` exactly as `parseIssuesCLIOutput` does. `watcher.go`: `PollCLI(projectDir, binary string)`, `CLIHealthCheck(projectDir, binary string)`; `source.go`: `FetchIssuesNow(projectDir, binary string)`.

### `cmd/mg/main.go` (modify)

- `brOnPath()` helper next to `bdOnPath()` (main.go:229).
- `resolveSource` precedence: `--path` → JSONL; `.beads/` + `br` → CLI(br); `.beads/` + `bd` → CLI(bd); `.beads/issues.jsonl` → JSONL; else error. (CLI-over-JSONL precedence preserved; `bd` used only when `br` is missing and `bd` present.)
- Startup fetch passes `source.CLIBinary`.

### `internal/app` (modify)

- `Model` gains `cliBinary string` (from `Source` in `NewWithGuard`) and threads it to `data.PollCLI` / `FetchIssuesNow` / `CLIHealthCheck` call sites (app.go:380, 388, 1229, 1428, 1448).

### `internal/actors` (new)

```go
// types.go
type ProjectRef struct {
    Root string `json:"root"`
    ID   string `json:"id"`
}
type Activity struct {
    SessionTitle *string `json:"session_title"`
    CurrentStep  *string `json:"current_step"`
    Todo         *string `json:"todo"`
    Asked        *string `json:"asked"`
    Now          *string `json:"now"`
}
type SprintRef struct {
    Epic  string `json:"epic"`
    Title string `json:"title"`
}
type SocietyRow struct {
    Name          string     `json:"name"`
    Role          string     `json:"role"`
    Project       ProjectRef `json:"project"`
    Kind          string     `json:"kind"` // canonical | sibling
    Lifecycle     string     `json:"lifecycle"`
    Readiness     string     `json:"readiness"`
    Activity      Activity   `json:"activity"`
    CurrentSprint *SprintRef `json:"current_sprint"`
}
type ProjectGroup struct {
    Project ProjectRef  `json:"project"`
    Actors  []SocietyRow `json:"actors"`
}

// client.go
func Available() bool                                  // exec.LookPath("actor")
func FetchProject(projectRef string) ([]SocietyRow, error) // actor project <ref> --json
func FetchVillage() ([]ProjectGroup, error)                // actor list --json
func FetchStatus(name string) (*SocietyRow, error)         // actor status <name> --json
```

Envelopes decoded with an `OK bool` field; `ok:false` or malformed JSON → error. Timeouts: project/status short (5s), village medium (15s), mirroring `internal/data/exec.go` tiers. Package exposes no mutation/control surface.

### `internal/views/actors.go` (new)

```go
type Actors struct { /* width, height, scrollOff, rows []actors.SocietyRow,
                        village []actors.ProjectGroup, scope, err, loadingFrame */ }
func NewActors(width, height int) Actors
// Setters for poll results; View() nil-safe before first fetch.
```

Renders one row per societyRow: name, kind, lifecycle, readiness, current_sprint (epic + title or "-"), asked/now (truncated to width). Detail expansion for the focused row shows the full capsule.

### `internal/app` wiring (modify, t3)

- `actorsAvail bool` = `actors.Available()` at startup; when false the pane and its keys are absent (progressive hide).
- `showActors bool` + toggle key `o` (verified unbound in `handleKey`; builder re-confirms against `components/help.go`).
- Scope toggle `v` inside the pane: project ↔ village.
- `actorsMsg` / `actorsErrMsg` routed like `townStatusMsg`; poll every 10s behind a single-flight gate; on-demand fetch when the pane opens (mirrors `activateGasTown`).
- `components/help.go`, `docs/keybindings.md`, README keybinding table updated in the same change (AGENTS.md rule).

## Data / control flow

Beads: `Init` → `startPoll` → `data.PollCLI(projectDir, cliBinary)` → `FetchIssuesCLI` → (`br list --json --limit 0 --all` | `bd list --json --limit 0 --all`) → parse (wrapper | array) → `FileChangedMsg` → parade rebuild. JSONL path unchanged.

Actors: pane opened (or avail at startup) → `pollActors` `tea.Cmd` → `actors.FetchProject(projectDir)` (or `FetchVillage`) → `actorsMsg{rows}` → view setters → render. Errors → `actorsErrMsg` → pane error state + toast, poll rescheduled. Single-flight gate drops re-polls while one is in flight.

## Error handling

- `br`/`actor` missing → feature hidden, not crashed: source resolution falls through (br→bd→jsonl); actors pane absent.
- Poll failures (exit non-zero, timeout) → toast + pane/footer error state, polling continues; never block Update, never crash (mirrors `townStatusErr` handling, app.go `townStatusMsg`).
- Malformed JSON from `br`/`actor` → returned as error and surfaced the same way; the last good data stays on screen. Precedent: malformed JSONL lines are skipped, not fatal (loader.go:31-34).
- `ok:false` envelopes from `actor` → treated as poll failure.
- `br` wrapper without an `issues` key → decode yields empty slice; treated as empty parade, not an error (wrapper shape verified live).

## Testing (behavioral contracts; exact tests live in the plan)

- br wrapper JSON parses into the same `[]Issue` the bd array path produces (fixture-based, via `mockRun` stub of `runWithTimeout`).
- `resolveSource` picks br when both br and bd exist; bd only when br absent; JSONL when neither; `--path` always wins. (cmd/mg `main_test.go` with a fake-bin PATH dir.)
- `Source.Label()` reports `br list` for a br CLI source.
- `internal/actors`: project/list/status envelopes parse including null `session_title`/`todo`/`current_sprint`; `ok:false` → error; garbage → error; missing binary → `Available()==false`.
- Actors view renders rows from a fixture societyRow set and renders safely with nil data and with an error set.
- End-to-end (verifier/lead re-runnable): in `/home/sf/workspace/mardi-gras`, `go build ./cmd/mg` then run `./mg` with no `--path` → parade shows the `mard-nob` epic via `br list`, actors pane shows the `zime` societyRow.

## Non-goals

- Gas Town / Gas City removal or rewrite.
- Actor control verbs (restore/recycle/ensure/send) anywhere in mg.
- Scraping hcom/herdr; mail via mg.
- Expanding `internal/data/mutate.go` or building a mutation control plane. (Consequence, not requirement: mutation keys shell out to `bd`, which is absent on this host — they will error. Read-only-first accepts this; not fixed in this campaign.)
- Any second actor registry, state store, or Beads mirror.
- Village-wide mutation UI.

## Implementation approach chosen (and rejected internals)

Chosen: thread the CLI binary through the existing `Source`/`FetchIssuesCLI` seam (smallest diff, reuses prefix validation, sorting, health tracking, and the `mockRun` test seam). Rejected: a new `BeadsClient` interface abstraction — one caller, two binaries; YAGNI. Rejected: JSONL-only on this host — violates the packet's "prefer `br`" and goes stale between syncs.

Chosen for actors: new `internal/actors` package + one pane modeled on the gastown progressive-hide pattern. Rejected: extending `internal/gastown.Driver` — actors are not an orchestrator backend and the Driver seam would force ErrUnsupported sprawl. Rejected: reading hcom state files — Brief forbids a parallel model; the `actor` CLI is the canonical projection.

## Open questions

None affecting product. (Internal, settled in the plan: toggle key `o`, scope key `v`, 10s actor poll interval, 5s/15s actor timeouts.)
