package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/matt-wright86/mardi-gras/internal/data"
)

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindBeadsFileInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".beads"))
	mustWrite(t, filepath.Join(dir, ".beads", "issues.jsonl"), []byte("[]"))

	got := findBeadsFile(dir)
	want := filepath.Join(dir, ".beads", "issues.jsonl")
	if got != want {
		t.Errorf("findBeadsFile(%q) = %q, want %q", dir, got, want)
	}
}

func TestFindBeadsFileWalksUp(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".beads"))
	mustWrite(t, filepath.Join(root, ".beads", "issues.jsonl"), []byte("[]"))

	child := filepath.Join(root, "a", "b")
	mustMkdir(t, child)

	got := findBeadsFile(child)
	want := filepath.Join(root, ".beads", "issues.jsonl")
	if got != want {
		t.Errorf("findBeadsFile(%q) = %q, want %q", child, got, want)
	}
}

func TestFindBeadsFileNotFound(t *testing.T) {
	dir := t.TempDir()

	got := findBeadsFile(dir)
	if got != "" {
		t.Errorf("findBeadsFile(%q) = %q, want empty string", dir, got)
	}
}

func TestParseBlockingTypesFlag(t *testing.T) {
	got := parseBlockingTypes("blocks,depends")
	want := map[string]bool{"blocks": true, "depends": true}

	if len(got) != len(want) {
		t.Fatalf("parseBlockingTypes returned %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseBlockingTypes missing key %q or wrong value", k)
		}
	}
}

func TestParseBlockingTypesEnvFallback(t *testing.T) {
	t.Setenv("MG_BLOCK_TYPES", "blocks,depends")

	got := parseBlockingTypes("")
	want := map[string]bool{"blocks": true, "depends": true}

	if len(got) != len(want) {
		t.Fatalf("parseBlockingTypes returned %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseBlockingTypes missing key %q or wrong value", k)
		}
	}
}

func TestParseBlockingTypesDefault(t *testing.T) {
	t.Setenv("MG_BLOCK_TYPES", "")

	got := parseBlockingTypes("")
	want := data.DefaultBlockingTypes

	if len(got) != len(want) {
		t.Fatalf("parseBlockingTypes returned %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseBlockingTypes missing key %q or wrong value", k)
		}
	}
}

func TestParseBlockingTypesTrimsWhitespace(t *testing.T) {
	got := parseBlockingTypes(" blocks , depends ")
	want := map[string]bool{"blocks": true, "depends": true}

	if len(got) != len(want) {
		t.Fatalf("parseBlockingTypes returned %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseBlockingTypes key %q: got %v, want %v", k, got[k], v)
		}
	}
}

func TestParseBlockingTypesEmptyCommas(t *testing.T) {
	got := parseBlockingTypes(",,")
	want := data.DefaultBlockingTypes

	if len(got) != len(want) {
		t.Fatalf("parseBlockingTypes(%q) returned %d entries, want %d (DefaultBlockingTypes)", ",,", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseBlockingTypes(%q) missing key %q or wrong value", ",,", k)
		}
	}
}

// ---------------------------------------------------------------------------
// resolveSource tests
// ---------------------------------------------------------------------------

func TestResolveSourceExplicitPath(t *testing.T) {
	src := resolveSource(t.TempDir(), "/some/path/.beads/issues.jsonl")
	if src.Mode != data.SourceJSONL {
		t.Fatalf("expected SourceJSONL, got %d", src.Mode)
	}
	if src.Path != "/some/path/.beads/issues.jsonl" {
		t.Fatalf("expected explicit path, got %q", src.Path)
	}
	if !src.Explicit {
		t.Fatal("expected Explicit to be true with --path flag")
	}
	if src.ProjectDir != "/some/path" {
		t.Fatalf("expected ProjectDir /some/path, got %q", src.ProjectDir)
	}
}

func TestResolveSourceExplicitPathTrailingSlash(t *testing.T) {
	src := resolveSource(t.TempDir(), "/some/path/.beads/issues.jsonl/")
	if src.Path != "/some/path/.beads/issues.jsonl" {
		t.Fatalf("expected trailing slash cleaned, got %q", src.Path)
	}
	if src.ProjectDir != "/some/path" {
		t.Fatalf("expected ProjectDir /some/path, got %q", src.ProjectDir)
	}
}

func TestResolveSourceExplicitRelativePath(t *testing.T) {
	src := resolveSource(t.TempDir(), "./project/.beads/issues.jsonl")
	if !filepath.IsAbs(src.Path) {
		t.Fatalf("expected absolute path from relative --path, got %q", src.Path)
	}
	if !filepath.IsAbs(src.ProjectDir) {
		t.Fatalf("expected absolute ProjectDir from relative --path, got %q", src.ProjectDir)
	}
}

func TestResolveSourceCLIPreferredOverJSONL(t *testing.T) {
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("bd not on PATH, skipping CLI preference test")
	}

	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".beads"))
	mustWrite(t, filepath.Join(dir, ".beads", "issues.jsonl"), []byte("[]"))

	src := resolveSource(dir, "")
	if src.Mode != data.SourceCLI {
		t.Fatalf("expected SourceCLI when bd is on PATH, got %d", src.Mode)
	}
	if src.ProjectDir != dir {
		t.Fatalf("expected ProjectDir %q, got %q", dir, src.ProjectDir)
	}
}

func TestResolveSourceJSONLLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".beads"))
	mustWrite(t, filepath.Join(dir, ".beads", "issues.jsonl"), []byte("[]"))

	// Remove bd from PATH so CLI is not available
	t.Setenv("PATH", dir)

	src := resolveSource(dir, "")
	if src.Mode != data.SourceJSONL {
		t.Fatalf("expected SourceJSONL as legacy fallback, got %d", src.Mode)
	}
	if src.Path == "" {
		t.Fatal("expected non-empty Path")
	}
	if src.Explicit {
		t.Fatal("expected Explicit to be false for auto-detected JSONL")
	}
}

func TestResolveSourceJSONLWalksUp(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".beads"))
	mustWrite(t, filepath.Join(root, ".beads", "issues.jsonl"), []byte("[]"))

	child := filepath.Join(root, "a", "b")
	mustMkdir(t, child)

	// Remove bd from PATH so we test the JSONL walkup path
	t.Setenv("PATH", root)

	src := resolveSource(child, "")
	if src.Mode != data.SourceJSONL {
		t.Fatalf("expected SourceJSONL, got %d", src.Mode)
	}
	want := filepath.Join(root, ".beads", "issues.jsonl")
	if src.Path != want {
		t.Fatalf("expected path %q, got %q", want, src.Path)
	}
}

func TestResolveSourceCLIFallback(t *testing.T) {
	// Only test if bd is on PATH
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("bd not on PATH, skipping CLI fallback test")
	}

	dir := t.TempDir()
	// Create .beads/ dir but no issues.jsonl
	mustMkdir(t, filepath.Join(dir, ".beads"))

	src := resolveSource(dir, "")
	if src.Mode != data.SourceCLI {
		t.Fatalf("expected SourceCLI, got %d", src.Mode)
	}
	if src.ProjectDir != dir {
		t.Fatalf("expected ProjectDir %q, got %q", dir, src.ProjectDir)
	}
}

func TestResolveSourceNoBdNoCLI(t *testing.T) {
	dir := t.TempDir()
	// Create .beads/ dir but no issues.jsonl
	mustMkdir(t, filepath.Join(dir, ".beads"))

	// Override PATH to exclude bd
	t.Setenv("PATH", dir) // temp dir won't have bd

	src := resolveSource(dir, "")
	// Without bd on PATH and no JSONL, should return empty source
	if src.Mode != data.SourceJSONL {
		t.Fatalf("expected default SourceJSONL mode, got %d", src.Mode)
	}
	if src.Path != "" {
		t.Fatalf("expected empty Path, got %q", src.Path)
	}
}

func TestResolveSourceNoBeadsDir(t *testing.T) {
	dir := t.TempDir()
	// No .beads/ at all

	src := resolveSource(dir, "")
	if src.Path != "" {
		t.Fatalf("expected empty Path with no .beads dir, got %q", src.Path)
	}
}

// writeFakeBin drops an executable stub named `name` into dir. resolveSource
// only calls exec.LookPath, so the body never runs.
func writeFakeBin(t *testing.T, dir, name string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// setupBeadsDir returns a project root containing an empty .beads/ directory.
func setupBeadsDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".beads"))
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

// ---------------------------------------------------------------------------
// findBeadsDir tests
// ---------------------------------------------------------------------------

func TestFindBeadsDirInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".beads"))

	got := findBeadsDir(dir)
	if got != dir {
		t.Errorf("findBeadsDir(%q) = %q, want %q", dir, got, dir)
	}
}

func TestFindBeadsDirWalksUp(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".beads"))

	child := filepath.Join(root, "a", "b")
	mustMkdir(t, child)

	got := findBeadsDir(child)
	if got != root {
		t.Errorf("findBeadsDir(%q) = %q, want %q", child, got, root)
	}
}

func TestFindBeadsDirNotFound(t *testing.T) {
	dir := t.TempDir()
	got := findBeadsDir(dir)
	if got != "" {
		t.Errorf("findBeadsDir(%q) = %q, want empty string", dir, got)
	}
}

func TestNoSourceMessageMentionsBothCLIs(t *testing.T) {
	msg := noSourceMessage()
	if !strings.Contains(msg, "br") {
		t.Errorf("no-source message should mention br, got %q", msg)
	}
	if !strings.Contains(msg, "bd") {
		t.Errorf("no-source message should mention bd, got %q", msg)
	}
	if !strings.Contains(msg, "issues.jsonl") {
		t.Errorf("no-source message should still name issues.jsonl, got %q", msg)
	}
}

func TestLoadFailureLineUsesSourceLabel(t *testing.T) {
	src := data.Source{Mode: data.SourceCLI, CLIBinary: data.CLIBr}
	got := loadFailureLine(src, errors.New("boom"))
	if !strings.Contains(got, "br list") {
		t.Errorf("load-failure line should name br list, got %q", got)
	}
	if strings.Contains(got, "bd list") {
		t.Errorf("br source should not mention bd list, got %q", got)
	}
}

func TestLoadFailureLineBdSource(t *testing.T) {
	src := data.Source{Mode: data.SourceCLI, CLIBinary: data.CLIBd}
	got := loadFailureLine(src, errors.New("boom"))
	if !strings.Contains(got, "bd list") {
		t.Errorf("bd source should name bd list, got %q", got)
	}
}

func TestLoadFailureHintDoltOnlyForBd(t *testing.T) {
	bd := loadFailureHint(data.Source{Mode: data.SourceCLI, CLIBinary: data.CLIBd})
	if !strings.Contains(bd, "Dolt") || !strings.Contains(bd, "bd") {
		t.Errorf("bd hint should mention Dolt and bd, got %q", bd)
	}
	br := loadFailureHint(data.Source{Mode: data.SourceCLI, CLIBinary: data.CLIBr})
	if strings.Contains(br, "Dolt") || strings.Contains(br, "dolt") {
		t.Errorf("br source must not get the Dolt/bd hint, got %q", br)
	}
}

// mardNobStatusTimeout bounds the `go run` compile-and-run step. A cold build
// cache has to link the whole BubbleTea tree before the status line prints, so
// the budget is generous; without it a wedged build hangs the test forever.
const mardNobStatusTimeout = 120 * time.Second

// statusWaitDelay bounds how long the process tests keep waiting on the child's
// output pipes after that child exits or the deadline above fires. `go run`
// compiles and runs the binary as a grandchild that inherits stdout: killing
// the `go run` parent at the deadline leaves the binary holding the write end
// of the pipe, so CombinedOutput can block indefinitely and the 120s deadline
// never actually bounds the test. Five seconds is far below the deadline, which
// keeps a wedged run failing with its own captured output instead of hanging
// the suite.
const statusWaitDelay = 5 * time.Second

// boundedCommand is the one place the process tests apply their deadline and
// pipe policy, so a hang cannot outlive the child by more than statusWaitDelay.
func boundedCommand(ctx context.Context, dir, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.WaitDelay = statusWaitDelay
	return cmd
}

// statusCommand builds the mg --status invocation over a fixture.
func statusCommand(ctx context.Context, fixturePath string) *exec.Cmd {
	return boundedCommand(ctx, filepath.Join("..", ".."),
		"go", "run", "./cmd/mg", "--status", "--path", fixturePath)
}

// tmuxMarkup matches the #[fg=colourN] directives tmux.StatusLine emits.
var tmuxMarkup = regexp.MustCompile(`#\[[^\]]*\]`)

// TestStatusCommandWaitDelayBoundsThePipe is the regression for statusWaitDelay.
// A child that exits while a grandchild inherits its stdout leaves the read end
// of the output pipe open with nobody to close it; without WaitDelay the helper
// above returns only when that grandchild finally exits, so the deadline is
// decorative. `sleep` deliberately outlives this test's threshold.
func TestStatusCommandWaitDelayBoundsThePipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the probe needs a POSIX shell")
	}
	ctx, cancel := context.WithTimeout(context.Background(), mardNobStatusTimeout)
	defer cancel()

	cmd := boundedCommand(ctx, t.TempDir(), "sh", "-c", "sleep 30 & exit 0")

	start := time.Now()
	_, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if elapsed > statusWaitDelay+10*time.Second {
		t.Fatalf("CombinedOutput waited %s on a pipe the child had already abandoned — WaitDelay is not bounding it", elapsed)
	}
	// The probe's child exits 0 while `sleep` still holds the pipe, which is
	// precisely the case WaitDelay resolves: Wait gives up on the open pipe and
	// reports ErrWaitDelay instead of blocking. Any other error is a real one.
	if err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("probe command failed: %v", err)
	}
}

// TestStatusModeOperatorReviewStoredState is the built-binary acceptance test
// for an explicitly stored review gate: the epic must count as Operator Review,
// never Working. It runs the real binary over a real fixture, so loading,
// hierarchy, derivation, grouping, and tmux rendering are all covered.
//
// The six counts are asserted in order: the widget reads left to right in
// StateOrder, and tmux's #[fg=...] markup sits between the tokens in the raw
// stream, so the sequence is only contiguous once that markup is removed.
func TestStatusModeOperatorReviewStoredState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), mardNobStatusTimeout)
	defer cancel()

	out, err := statusCommand(ctx, "testdata/operator-review-tree.jsonl").CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("go run exceeded %s: %s", mardNobStatusTimeout, out)
		}
		t.Fatalf("%v: %s", err, out)
	}

	plain := tmuxMarkup.ReplaceAllString(string(out), "")
	if want := "2○ 0● 1⊘ 1⏸ 1◐ 1✓"; !strings.Contains(plain, want) {
		t.Errorf("status line does not carry the ordered counts %q: %q", want, plain)
	}
	if strings.Contains(plain, "1●") {
		t.Fatalf("stored review epic rendered Working: %s", out)
	}
}
