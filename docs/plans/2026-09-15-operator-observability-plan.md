# Operator Observability (Beads parade + actor society) Implementation Plan

**Technical Spec:** `/home/sf/workspace/mardi-gras/docs/plans/2026-09-15-operator-observability-spec.md`
**Design Brief:** `/home/sf/worlds/personal/designs/oh-my-pi/actor-society-surface.md` (via spec; do not bypass)

> **For the project lead:** first `br where` in this repo. no board → `br init --prefix <xx>` here (never `~/.beads`). then one campaign parent bead for this plan, then task-by-task with isolated `builder` workers. Do not implement these tasks inline. Persist progress in this file's checkboxes **and** as one parent-child bead per Task (`br create --parent --body`). Title `tN: <ask>` (prefix `[done] ` after verifier pass). Description markdown, ≤800 characters, wrap at ~60 cols: `Asked:` paragraph, then `## Landed` with sha/tests/leftover (`not yet` while in flight). Do not paste this plan packet. After pass, keep the child `in_progress` (mgr Rolling), assignee cleared — do not defer, do not `br close` until the parent parks (closed = Past the Stand). Lined Up is only for minted-not-yet-claimed children.

**Goal:** Make `mg` the read-only operator observability UI at this managed root: Beads parade over `br` (no `--path` needed on this host) plus an actor society pane driven by the `actor` CLI societyRow projection.
**Architecture:** (a) thread the CLI binary (`br`|`bd`) through the existing `data.Source`/`FetchIssuesCLI` seam, parsing br's `{issues:[...]}` wrapper; (b) new `internal/actors` read-only client over `actor project|list|status --json`; (c) new `internal/views/actors.go` pane on the gastown background-poll / hide-when-missing pattern.
**Tech stack:** Go, BubbleTea v2, lipgloss — exactly what the repo already uses. No new dependencies.

---

## File structure

| File | Action | Responsibility |
|---|---|---|
| `internal/data/source.go` | Modify | `Source.CLIBinary`, `CLIBd`/`CLIBr` consts, `Label()`, `FetchIssuesCLI(projectDir, binary)`, `brListEnvelope`, `parseBrListOutput`, `FetchIssuesNow(projectDir, binary)` |
| `internal/data/watcher.go` | Modify | `PollCLI(projectDir, binary)`, `CLIHealthCheck(projectDir, binary)` |
| `cmd/mg/main.go` | Modify | `brOnPath()`, `resolveSource` br-first precedence, startup fetch passes binary |
| `internal/app/app.go` | Modify (t1) | `Model.cliBinary` field, set in `NewWithGuard`, thread to poll call sites (lines ~380, 388, 1229, 1428, 1448) |
| `internal/data/source_test.go` | Modify | br wrapper parse + dispatch + label tests |
| `cmd/mg/main_test.go` | Modify | resolveSource precedence tests with fake-bin PATH |
| `internal/actors/types.go` | Create | `SocietyRow`, `Activity`, `SprintRef`, `ProjectRef`, `ProjectGroup` |
| `internal/actors/exec.go` | Create | stubbable `runWithTimeout` var + timeout tiers (mirrors `internal/data/exec.go`) |
| `internal/actors/client.go` | Create | `Available`, `FetchProject`, `FetchVillage`, `FetchStatus` + envelope decoding |
| `internal/actors/client_test.go` | Create | envelope/null/error/missing-binary tests |
| `internal/views/actors.go` | Create | `Actors` pane model (nil-safe, error state, scope toggle) |
| `internal/views/actors_test.go` | Create | render-contract tests |
| `internal/app/app.go` | Modify (t3) | `actorsAvail`, `showActors`, poll cmd + gate, `actorsMsg`/`actorsErrMsg`, keys `o`/`v` |
| `internal/components/help.go` | Modify (t3) | keybinding entries for the actors pane |
| `docs/keybindings.md`, `README.md` | Modify (t3) | keybinding docs (AGENTS.md same-change rule) |

**Dependency notes:** t1 and t2 are disjoint (t1: data/cmd/app-poll-sites; t2: new package) — run in parallel. t3 imports `internal/actors` (needs t2) and edits `internal/app/app.go` (t1 also touches it) — t3 starts only after t1 **and** t2 land. t4 verifies the whole.

---

### Task 1: br-aware Beads CLI source

**Owner:** builder
**Files:**
- Modify: `internal/data/source.go`
- Modify: `internal/data/watcher.go`
- Modify: `cmd/mg/main.go`
- Modify: `internal/app/app.go` (poll call sites only)
- Test: `internal/data/source_test.go`, `cmd/mg/main_test.go`

**Verification (anti-gameable):** `go test ./internal/data -run 'TestParseBrListOutput|TestFetchIssuesCLI|TestSourceLabel' -v` and `go test ./cmd/mg -run TestResolveSource -v` pass; `go build ./...` clean. Verifier re-runs: `cd /home/sf/workspace/mardi-gras && go run ./cmd/mg --status` prints a parade status line containing `mard-nob` without any `--path` flag.

- [ ] **Step 1: Write the failing tests**

Add to `internal/data/source_test.go`:

```go
func TestParseBrListOutput(t *testing.T) {
	out := []byte(`{"issues":[{"id":"mard-nob","title":"operator UI","status":"in_progress","priority":1,"issue_type":"epic","assignee":"zime","updated_at":"2026-09-15T06:59:04Z"},{"id":"mard-abc","title":"closed one","status":"closed","priority":2,"issue_type":"task","updated_at":"2026-09-15T06:00:00Z"}],"total":2,"limit":0,"offset":0,"has_more":false}`)
	issues, err := parseBrListOutput(out, "mard")
	if err != nil {
		t.Fatalf("parseBrListOutput: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
	// SortIssues: active first
	if issues[0].ID != "mard-nob" {
		t.Errorf("first issue = %q, want mard-nob (active sorts first)", issues[0].ID)
	}
}

func TestParseBrListOutputMalformed(t *testing.T) {
	if _, err := parseBrListOutput([]byte(`not json`), ""); err == nil {
		t.Fatal("want error for malformed br output")
	}
}

func TestFetchIssuesCLIBrWrapper(t *testing.T) {
	defer mockRun(`{"issues":[{"id":"mard-nob","title":"t","status":"open","priority":1,"issue_type":"epic","updated_at":"2026-09-15T06:59:04Z"}],"total":1}`)()
	issues, err := FetchIssuesCLI("", CLIBr)
	if err != nil {
		t.Fatalf("FetchIssuesCLI br: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != "mard-nob" {
		t.Fatalf("got %+v", issues)
	}
}

func TestFetchIssuesCLIBdArray(t *testing.T) {
	defer mockRun(`[{"id":"mard-nob","title":"t","status":"open","priority":1,"issue_type":"epic","updated_at":"2026-09-15T06:59:04Z"}]`)()
	issues, err := FetchIssuesCLI("", CLIBd)
	if err != nil {
		t.Fatalf("FetchIssuesCLI bd: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != "mard-nob" {
		t.Fatalf("got %+v", issues)
	}
}

func TestSourceLabelBr(t *testing.T) {
	s := Source{Mode: SourceCLI, CLIBinary: CLIBr, ProjectDir: "/x"}
	if s.Label() != "br list" {
		t.Errorf("Label() = %q, want %q", s.Label(), "br list")
	}
}
```

Add to `cmd/mg/main_test.go` (uses a temp bin dir on PATH with fake executables — empty files with the exec bit suffice, since `resolveSource` only calls `exec.LookPath`):

```go
func writeFakeBin(t *testing.T, dir, name string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setupBeadsDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".beads"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestResolveSourcePrefersBr(t *testing.T) {
	bin := t.TempDir()
	writeFakeBin(t, bin, "br")
	writeFakeBin(t, bin, "bd")
	t.Setenv("PATH", bin)
	src := resolveSource(setupBeadsDir(t), "")
	if src.Mode != data.SourceCLI || src.CLIBinary != data.CLIBr {
		t.Fatalf("got %+v, want CLI/br", src)
	}
}

func TestResolveSourceBdOnlyWhenBrMissing(t *testing.T) {
	bin := t.TempDir()
	writeFakeBin(t, bin, "bd")
	t.Setenv("PATH", bin)
	src := resolveSource(setupBeadsDir(t), "")
	if src.Mode != data.SourceCLI || src.CLIBinary != data.CLIBd {
		t.Fatalf("got %+v, want CLI/bd", src)
	}
}

func TestResolveSourceJSONLWhenNoCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := setupBeadsDir(t)
	if err := os.WriteFile(filepath.Join(root, ".beads", "issues.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := resolveSource(root, "")
	if src.Mode != data.SourceJSONL {
		t.Fatalf("got %+v, want JSONL fallback", src)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/data -run 'TestParseBrListOutput|TestFetchIssuesCLI|TestSourceLabel' -v 2>&1 | head -20; go test ./cmd/mg -run TestResolveSource -v 2>&1 | head -20`
Expected: FAIL to compile — `undefined: parseBrListOutput`, `undefined: CLIBr`, `too many arguments in call to FetchIssuesCLI`.

- [ ] **Step 3: Implement the br-aware source**

In `internal/data/source.go`, add the constants, the `CLIBinary` field, the envelope type, and the parser; change `FetchIssuesCLI` and `FetchIssuesNow` to take the binary; update `Label()`:

```go
const (
	CLIBd = "bd"
	CLIBr = "br"
)

// Source: add field
	CLIBinary  string // CLIBd | CLIBr; empty unless Mode == SourceCLI

func (s Source) Label() string {
	if s.Mode == SourceCLI {
		if s.CLIBinary == CLIBr {
			return "br list"
		}
		return "bd list"
	}
	if s.Path != "" {
		return filepath.Base(s.Path)
	}
	return "issues.jsonl"
}

// FetchIssuesCLI runs `<binary> list --json --limit 0 --all` and parses the result.
// br returns a wrapper object; bd returns a bare array.
func FetchIssuesCLI(projectDir, binary string) ([]Issue, error) {
	out, err := runWithTimeout(timeoutMedium, binary, bdListArgs()...)
	if err != nil {
		return nil, wrapExitError(binary+" list --json", err)
	}
	if binary == CLIBr {
		return parseBrListOutput(out, LoadIssuePrefix(projectDir))
	}
	return parseIssuesCLIOutput(out, LoadIssuePrefix(projectDir))
}

// brListEnvelope mirrors the `br list --json` wrapper shape.
type brListEnvelope struct {
	Issues []Issue `json:"issues"`
}

func parseBrListOutput(out []byte, expectedPrefix string) ([]Issue, error) {
	var env brListEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("br list parse: %w", err)
	}
	if err := validateIssuePrefixes(env.Issues, expectedPrefix); err != nil {
		return nil, err
	}
	SortIssues(env.Issues)
	return env.Issues, nil
}
```

In `FetchIssuesNow` (same file): signature becomes `func FetchIssuesNow(projectDir, binary string) tea.Cmd` and its inner call becomes `FetchIssuesCLI(projectDir, binary)`.

In `internal/data/watcher.go`: `PollCLI(projectDir, binary string)` and `CLIHealthCheck(projectDir, binary string)`, each passing `binary` through to `FetchIssuesCLI`.

In `cmd/mg/main.go`: add `brOnPath` and rewrite the CLI branch of `resolveSource`:

```go
// brOnPath returns true if the br command is available.
func brOnPath() bool {
	_, err := exec.LookPath("br")
	return err == nil
}

// inside resolveSource, replacing the bdOnPath block:
	// Prefer CLI when br or bd is available; br wins (bd is the legacy binary).
	if projectDir := findBeadsDir(cwd); projectDir != "" {
		if brOnPath() {
			return data.Source{Mode: data.SourceCLI, ProjectDir: projectDir, CLIBinary: data.CLIBr}
		}
		if bdOnPath() {
			return data.Source{Mode: data.SourceCLI, ProjectDir: projectDir, CLIBinary: data.CLIBd}
		}
	}
```

Also update the doc comment above `resolveSource` to state the new precedence, and change the startup fetch (main.go:101) to `data.FetchIssuesCLI(source.ProjectDir, source.CLIBinary)`.

In `internal/app/app.go`: add `cliBinary string` to `Model`; in `NewWithGuard`, where `sourceMode` is set from `source.Mode`, also set `cliBinary: source.CLIBinary`; update the five call sites — `data.PollCLI(m.projectDir)` → `data.PollCLI(m.projectDir, m.cliBinary)` (line ~380), `data.FetchIssuesNow(m.projectDir)` → `data.FetchIssuesNow(m.projectDir, m.cliBinary)` (line ~388), and `data.CLIHealthCheck(m.projectDir)` → `data.CLIHealthCheck(m.projectDir, m.cliBinary)` (lines ~1229, 1428, 1448).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/data -run 'TestParseBrListOutput|TestFetchIssuesCLI|TestSourceLabel' -v && go test ./cmd/mg -run TestResolveSource -v && go build ./...`
Expected: all PASS, build clean.

- [ ] **Step 5: Smoke the real source headlessly**

Run: `go run ./cmd/mg --status`
Expected: a one-line tmux status string whose parade counts include the `mard-nob` epic (e.g. a Rolling count ≥ 1), no `--path` given, no error about `bd`.

- [ ] **Step 6: Commit**

```bash
git add internal/data/source.go internal/data/watcher.go internal/data/source_test.go cmd/mg/main.go cmd/mg/main_test.go internal/app/app.go
git commit -m "feat: prefer br list --json wrapper for the Beads CLI source"
```

---

### Task 2: `internal/actors` read-only client

**Owner:** builder
**Files:**
- Create: `internal/actors/types.go`
- Create: `internal/actors/exec.go`
- Create: `internal/actors/client.go`
- Test: `internal/actors/client_test.go`

**Verification (anti-gameable):** `go test ./internal/actors -v` passes, including a missing-binary test that manipulates PATH (cannot be faked by hardcoding); `go vet ./internal/actors` clean. Verifier re-runs: `rg -n 'restore|recycle|send|ensure' internal/actors --type go` returns nothing (no control verbs).

- [ ] **Step 1: Write the failing tests**

Create `internal/actors/client_test.go`:

```go
package actors

import (
	"testing"
	"time"
)

// stubRun replaces runWithTimeout with a stub returning fixed output.
func stubRun(out string) func() {
	orig := runWithTimeout
	runWithTimeout = func(_ time.Duration, _ ...string) ([]byte, error) {
		return []byte(out), nil
	}
	return func() { runWithTimeout = orig }
}

const projectFixture = `{"ok":true,"project":{"root":"/home/sf/workspace/mardi-gras","id":"mardi-gras"},"count":1,"actors":[{"name":"zime","role":"project-lead","project":{"id":"mardi-gras"},"kind":"canonical","lifecycle":"active","readiness":"ready","activity":{"session_title":null,"current_step":"SPRINT: mard-nob","todo":null,"asked":"reko -> zime: operator UI","now":"SPRINT: mard-nob"},"current_sprint":{"epic":"mard-nob","title":"operator Beads/actor observability UI"}}]}`

func TestFetchProjectParsesSocietyRow(t *testing.T) {
	defer stubRun(projectFixture)()
	rows, err := FetchProject("mardi-gras")
	if err != nil {
		t.Fatalf("FetchProject: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.Name != "zime" || r.Kind != "canonical" || r.Lifecycle != "active" || r.Readiness != "ready" {
		t.Errorf("identity/presence wrong: %+v", r)
	}
	if r.CurrentSprint == nil || r.CurrentSprint.Epic != "mard-nob" {
		t.Errorf("current_sprint wrong: %+v", r.CurrentSprint)
	}
	if r.Activity.SessionTitle != nil || r.Activity.Todo != nil {
		t.Errorf("null capsule fields must stay nil: %+v", r.Activity)
	}
	if r.Activity.Now == nil || *r.Activity.Now != "SPRINT: mard-nob" {
		t.Errorf("activity.now wrong: %+v", r.Activity.Now)
	}
}

func TestFetchProjectNullSprint(t *testing.T) {
	defer stubRun(`{"ok":true,"project":{"root":"/x","id":"x"},"count":1,"actors":[{"name":"sumo","role":"project-lead","project":{"id":"x"},"kind":"canonical","lifecycle":"stopped","readiness":"needs_operator","activity":{"session_title":"s","current_step":null,"todo":null,"asked":"a","now":"n"},"current_sprint":null}]}`)()
	rows, err := FetchProject("x")
	if err != nil {
		t.Fatalf("FetchProject: %v", err)
	}
	if rows[0].CurrentSprint != nil {
		t.Errorf("want nil sprint, got %+v", rows[0].CurrentSprint)
	}
}

func TestFetchProjectNotOK(t *testing.T) {
	defer stubRun(`{"ok":false,"count":0,"actors":[]}`)()
	if _, err := FetchProject("x"); err == nil {
		t.Fatal("want error on ok:false envelope")
	}
}

func TestFetchProjectMalformed(t *testing.T) {
	defer stubRun(`garbage`)()
	if _, err := FetchProject("x"); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func TestFetchVillageParsesGroups(t *testing.T) {
	defer stubRun(`{"ok":true,"count":1,"projects":[{"project":{"root":"/x","id":"x"},"actors":[{"name":"sumo","role":"project-lead","project":{"id":"x"},"kind":"canonical","lifecycle":"stopped","readiness":"needs_operator","activity":{"session_title":null,"current_step":null,"todo":null,"asked":null,"now":null},"current_sprint":null}]}]}`)()
	groups, err := FetchVillage()
	if err != nil {
		t.Fatalf("FetchVillage: %v", err)
	}
	if len(groups) != 1 || groups[0].Project.ID != "x" || len(groups[0].Actors) != 1 {
		t.Fatalf("got %+v", groups)
	}
}

func TestFetchStatusParsesSingleRow(t *testing.T) {
	defer stubRun(`{"ok":true,"actor":{"name":"zime","role":"project-lead","project":{"id":"mardi-gras"},"kind":"canonical","lifecycle":"active","readiness":"ready","activity":{"session_title":null,"current_step":null,"todo":null,"asked":"a","now":"n"},"current_sprint":{"epic":"mard-nob","title":"t"}}}`)()
	row, err := FetchStatus("zime")
	if err != nil {
		t.Fatalf("FetchStatus: %v", err)
	}
	if row.Name != "zime" || row.CurrentSprint == nil {
		t.Fatalf("got %+v", row)
	}
}

func TestAvailableMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir: no actor on PATH
	if Available() {
		t.Fatal("Available() = true with empty PATH, want false")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/actors -v 2>&1 | head -10`
Expected: FAIL — `package github.com/matt-wright86/mardi-gras/internal/actors: no Go files` (or undefined symbols once files exist as stubs).

- [ ] **Step 3: Implement the package**

`internal/actors/types.go`:

```go
// Package actors is mg's read-only client for the actor society observation
// CLI (`actor project|list|status --json`). It exposes the societyRow
// projection only — no control verbs (restore/recycle/send) exist here.
package actors

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
	Project ProjectRef   `json:"project"`
	Actors  []SocietyRow `json:"actors"`
}
```

`internal/actors/exec.go`:

```go
package actors

import (
	"context"
	"os/exec"
	"time"
)

// Timeout tiers mirror internal/data/exec.go. Village list is the slow one
// (observed up to ~10s on a live village); project/status are fast.
const (
	timeoutShort  = 5 * time.Second
	timeoutMedium = 15 * time.Second
)

// runWithTimeout executes the actor CLI with a context timeout and returns
// stdout. Declared as a var so tests can stub it (same seam as
// internal/data's mockRun).
var runWithTimeout = func(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, "actor", args...).Output()
}
```

`internal/actors/client.go`:

```go
package actors

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Available reports whether the actor CLI is on PATH. When false, callers
// hide the actors feature entirely (progressive hide, same as gastown).
func Available() bool {
	_, err := exec.LookPath("actor")
	return err == nil
}

type projectEnvelope struct {
	OK      bool         `json:"ok"`
	Project ProjectRef   `json:"project"`
	Count   int          `json:"count"`
	Actors  []SocietyRow `json:"actors"`
}

// FetchProject runs `actor project <projectRef> --json`. projectRef may be a
// project id or a filesystem root (both accepted by the CLI).
func FetchProject(projectRef string) ([]SocietyRow, error) {
	out, err := runWithTimeout(timeoutShort, "project", projectRef, "--json")
	if err != nil {
		return nil, fmt.Errorf("actor project: %w", err)
	}
	var env projectEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("actor project parse: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("actor project: envelope not ok")
	}
	return env.Actors, nil
}

type listEnvelope struct {
	OK       bool           `json:"ok"`
	Count    int            `json:"count"`
	Projects []ProjectGroup `json:"projects"`
}

// FetchVillage runs `actor list --json` (village-wide, grouped by project).
func FetchVillage() ([]ProjectGroup, error) {
	out, err := runWithTimeout(timeoutMedium, "list", "--json")
	if err != nil {
		return nil, fmt.Errorf("actor list: %w", err)
	}
	var env listEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("actor list parse: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("actor list: envelope not ok")
	}
	return env.Projects, nil
}

type statusEnvelope struct {
	OK    bool       `json:"ok"`
	Actor SocietyRow `json:"actor"`
}

// FetchStatus runs `actor status <name> --json`.
func FetchStatus(name string) (*SocietyRow, error) {
	out, err := runWithTimeout(timeoutShort, "status", name, "--json")
	if err != nil {
		return nil, fmt.Errorf("actor status: %w", err)
	}
	var env statusEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("actor status parse: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("actor status: envelope not ok")
	}
	return &env.Actor, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/actors -v && go vet ./internal/actors`
Expected: all 8 tests PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add internal/actors/
git commit -m "feat: read-only actor society client (actor project|list|status --json)"
```

---

### Task 3: Actors pane + app wiring

**Owner:** builder
**Depends on:** t1 landed, t2 landed (imports `internal/actors`; shares `internal/app/app.go` with t1)
**Files:**
- Create: `internal/views/actors.go`
- Modify: `internal/app/app.go`
- Modify: `internal/components/help.go`
- Modify: `docs/keybindings.md`, `README.md` (keybinding tables)
- Test: `internal/views/actors_test.go`

**Verification (anti-gameable):** `go test ./internal/views -run TestActors -v` passes; `go build ./...` clean; verifier runs `./mg` in `/home/sf/workspace/mardi-gras`, presses `o`, and sees a row for `zime` with kind/lifecycle/readiness, sprint `mard-nob`, and asked/now text; pressing `v` switches to the village grouping; running with an empty PATH (`PATH=/usr/bin ./mg`) shows no actors pane and no `o` key in help.

- [ ] **Step 1: Write the failing view tests**

Create `internal/views/actors_test.go`:

```go
package views

import (
	"strings"
	"testing"

	"github.com/matt-wright86/mardi-gras/internal/actors"
)

func actorFixture() []actors.SocietyRow {
	now := "SPRINT: mard-nob operator Beads/actor observability UI"
	asked := "reko -> zime: operator UI"
	return []actors.SocietyRow{{
		Name:       "zime",
		Role:       "project-lead",
		Kind:       "canonical",
		Lifecycle:  "active",
		Readiness:  "ready",
		Activity:   actors.Activity{Now: &now, Asked: &asked},
		CurrentSprint: &actors.SprintRef{Epic: "mard-nob", Title: "operator Beads/actor observability UI"},
	}}
}

func TestActorsViewRendersSocietyRow(t *testing.T) {
	a := NewActors(80, 24)
	a.SetRows(actorFixture())
	v := a.View()
	for _, want := range []string{"zime", "canonical", "active", "ready", "mard-nob"} {
		if !strings.Contains(v, want) {
			t.Errorf("View() missing %q:\n%s", want, v)
		}
	}
}

func TestActorsViewNilSafeBeforeFirstFetch(t *testing.T) {
	a := NewActors(80, 24)
	v := a.View() // must not panic, must show a loading/empty state
	if !strings.Contains(strings.ToLower(v), "loading") && !strings.Contains(strings.ToLower(v), "no actors") {
		t.Errorf("want loading/empty state, got:\n%s", v)
	}
}

func TestActorsViewErrorState(t *testing.T) {
	a := NewActors(80, 24)
	a.SetErr(errForTest("actor project: exit status 1"))
	v := a.View()
	if !strings.Contains(v, "exit status 1") {
		t.Errorf("want error text in view, got:\n%s", v)
	}
}

func TestActorsViewNullSprintRendersDash(t *testing.T) {
	a := NewActors(80, 24)
	rows := actorFixture()
	rows[0].CurrentSprint = nil
	a.SetRows(rows)
	if v := a.View(); !strings.Contains(v, "-") {
		t.Errorf("want dash for null sprint, got:\n%s", v)
	}
}
```

(`errForTest` = `errors.New`; name it whatever the file's imports make natural — the contract is the rendered string.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/views -run TestActors -v 2>&1 | head -10`
Expected: FAIL to compile — `undefined: NewActors`.

- [ ] **Step 3: Implement `internal/views/actors.go`**

Follow the `internal/views/gastown.go` pane conventions (value-receiver `View`, pointer setters, manual `scrollOff`, styles from `internal/ui`). Minimum shape:

```go
// Package views — Actors renders the read-only actor society pane.
package views

import (
	"strings"

	"github.com/matt-wright86/mardi-gras/internal/actors"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// ActorsScope selects project-only vs village-wide listing.
type ActorsScope int

const (
	ActorsScopeProject ActorsScope = iota
	ActorsScopeVillage
)

type Actors struct {
	width, height int
	scrollOff     int
	rows          []actors.SocietyRow
	village       []actors.ProjectGroup
	scope         ActorsScope
	err           error
	loadingFrame  string
	fetched       bool // false until first successful poll
}

func NewActors(width, height int) Actors {
	return Actors{width: width, height: height}
}

func (a *Actors) SetRows(rows []actors.SocietyRow)   { a.rows, a.err, a.fetched = rows, nil, true }
func (a *Actors) SetVillage(g []actors.ProjectGroup) { a.village, a.err, a.fetched = g, nil, true }
func (a *Actors) SetErr(err error)                   { a.err = err }
func (a *Actors) SetScope(s ActorsScope)             { a.scope = s }
func (a *Actors) SetLoadingFrame(f string)           { a.loadingFrame = f }
func (a *Actors) SetSize(w, h int)                   { a.width, a.height = w, h }

// View renders the pane. Nil-safe before the first fetch; shows the last
// good data plus the error when a poll fails (mirrors the gastown panel).
func (a Actors) View() string {
	var b strings.Builder
	b.WriteString(ui.DetailTitle.Render("ACTORS"))
	b.WriteString("\n")
	if a.err != nil && !a.fetched {
		b.WriteString("actor CLI error: " + a.err.Error())
		return b.String()
	}
	if !a.fetched {
		b.WriteString(strings.TrimSpace(a.loadingFrame) + " Loading actors…")
		return b.String()
	}
	if a.scope == ActorsScopeVillage {
		for _, g := range a.village {
			b.WriteString(ui.DetailSection.Render(g.Project.ID) + "\n")
			for _, r := range g.Actors {
				b.WriteString(renderSocietyRow(r, a.width) + "\n")
			}
		}
		return b.String()
	}
	if len(a.rows) == 0 {
		b.WriteString("No actors for this project.")
		return b.String()
	}
	for _, r := range a.rows {
		b.WriteString(renderSocietyRow(r, a.width) + "\n")
	}
	if a.err != nil {
		b.WriteString("\n" + ui.ErrorStyle.Render("last poll failed: "+a.err.Error()))
	}
	return b.String()
}

// renderSocietyRow: "zime  canonical  active/ready  mard-nob — operator…  now: …"
func renderSocietyRow(r actors.SocietyRow, width int) string {
	sprint := "-"
	if r.CurrentSprint != nil {
		sprint = r.CurrentSprint.Epic
	}
	now := ""
	if r.Activity.Now != nil {
		now = "  now: " + *r.Activity.Now
	}
	line := r.Name + "  " + r.Kind + "  " + r.Lifecycle + "/" + r.Readiness + "  " + sprint + now
	if width > 0 && len(line) > width-2 {
		line = line[:width-3] + "…"
	}
	return line
}
```

(Exact style names — `ui.DetailTitle`, `ui.DetailSection`, `ui.ErrorStyle` — must be checked against `internal/ui/styles.go`; use the closest existing styles, do not invent new ones.)

- [ ] **Step 4: Run view tests to verify they pass**

Run: `go test ./internal/views -run TestActors -v`
Expected: 4 tests PASS.

- [ ] **Step 5: Wire the pane into the app**

In `internal/app/app.go`, mirroring the gastown wiring (`activateGasTown`, `townStatusMsg`, `gtPollInFlight`):

1. Fields on `Model`: `actors views.Actors`, `actorsAvail bool`, `showActors bool`, `actorsPollInFlight bool`.
2. In `NewWithGuard`: `actorsAvail: actors.Available()`, `actors: views.NewActors(w, h)` (use the same initial dimensions the other sub-models get).
3. Messages and poll command:

```go
type actorsMsg struct {
	rows    []actors.SocietyRow
	village []actors.ProjectGroup
}
type actorsErrMsg struct{ err error }

const actorsPollInterval = 10 * time.Second

func (m Model) pollActors() tea.Cmd {
	return tea.Tick(actorsPollInterval, func(time.Time) tea.Msg {
		if m.actorsScopeIsVillage() {
			g, err := actors.FetchVillage()
			if err != nil {
				return actorsErrMsg{err}
			}
			return actorsMsg{village: g}
		}
		rows, err := actors.FetchProject(m.projectDir)
		if err != nil {
			return actorsErrMsg{err}
		}
		return actorsMsg{rows: rows}
	})
}
```

Gate re-polls behind `actorsPollInFlight` exactly like `gatedPollAgentState` gates on `gtPollInFlight`; fetch immediately (bypassing the tick) when the pane opens, like `activateGasTown` does.

4. Key handling in `handleKey`: `case "o":` toggles `showActors` (only when `m.actorsAvail`; no-op otherwise) and triggers the immediate fetch on open. When the pane is focused, `case "v":` flips project ↔ village scope and refetches. (`o` and `v` are unbound in `handleKey` as of this plan; re-confirm against `components/help.go` before committing.)
5. Layout: when `showActors` is true, the actors pane replaces the detail pane (same slot the gastown panel uses; if both are on, gastown wins — record that choice in a comment).
6. Route `actorsMsg`/`actorsErrMsg` in `Update`: setters on `m.actors`, toast on error (mirroring the `townStatusMsg` error path), reschedule the gated poll.
7. Footer: when `actorsAvail` is false, no actors key hint appears (progressive hide).

- [ ] **Step 6: Build and update keybinding docs**

Run: `go build ./...`
Expected: clean.

Update `internal/components/help.go` (new pane section or row for `o` toggle + `v` scope), `docs/keybindings.md`, and the `README.md` keybinding table — same change, per AGENTS.md.

- [ ] **Step 7: Commit**

```bash
git add internal/views/actors.go internal/views/actors_test.go internal/app/app.go internal/components/help.go docs/keybindings.md README.md
git commit -m "feat: read-only actors pane over the actor CLI societyRow projection"
```

---

### Task 4: End-to-end acceptance

**Owner:** builder
**Depends on:** t1, t2, t3 landed
**Files:** none (verification only; fixes, if any, land as new commits on the same branch)

**Verification (anti-gameable):** the commands below are re-run by the verifier/lead from a clean checkout of the branch; outputs must match.

- [ ] **Step 1: Full test suite and build**

Run: `go test ./... && go vet ./... && go build -o mg ./cmd/mg`
Expected: all packages PASS, vet clean, binary produced.

- [ ] **Step 2: Headless source check (no --path)**

Run: `cd /home/sf/workspace/mardi-gras && ./mg --status`
Expected: one status line whose counts reflect the live `mard-nob` ledger via `br list` (Rolling ≥ 1 while the epic is in progress); no `bd` error, no `--path` flag.

- [ ] **Step 3: TUI acceptance (operator scenario)**

Run: `cd /home/sf/workspace/mardi-gras && ./mg`
Expected (verifier observes):
1. Parade shows the `mard-nob` epic; footer source label reads `br list`.
2. Press `o`: actors pane opens and, within one poll, shows a `zime` row with `canonical`, `active`/`ready` (or current lifecycle/readiness), sprint `mard-nob`, and asked/now text.
3. Press `v` in the pane: village grouping appears (multiple projects).
4. `PATH=/usr/bin ./mg`: no actors pane, no `o` in help, parade still works (br lives outside the stripped PATH only if the verifier's stripped PATH excludes it — if br is also stripped, the JSONL fallback or the no-source error is the correct behavior; record which).

- [ ] **Step 4: Commit (only if fixes were needed)**

```bash
git add -p
git commit -m "fix: acceptance findings for operator observability"
```

---

## Self-review notes

- Spec coverage: (a) br source → t1; (b) actors client → t2; (c) pane + wiring → t3; acceptance → t4. Error-handling contracts are tested in t1 (malformed br), t2 (ok:false, malformed, missing binary), t3 (error state, nil-safe).
- No placeholders: every code step shows the code; every command names the runner and expected signal.
- Type consistency: `SocietyRow`/`Activity`/`SprintRef`/`ProjectGroup` identical across t2 and t3; `FetchIssuesCLI(projectDir, binary)` identical across t1 call sites.
- Invention scan: no Gas Town changes, no control verbs, no mutation work, no second model. `o`/`v` keys and the 10s interval are internal choices recorded in the spec.
