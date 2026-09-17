package ui

import (
	"image/color"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// execStates pins the shared six-state execution vocabulary to its integer
// order: Working=0, AwaitingReview=1, Ready=2, Deferred=3, WaitingBlocked=4,
// Done=5. That order is the data package's SemanticState; ui is a leaf package
// and does not import it, so every Exec* lookup takes the bare int.
//
// palette is read through a func so the test never freezes a palette var at
// package-init time (the dark values), which a later SetTheme(ThemeLight)
// would otherwise turn into a false failure.
var execStates = []struct {
	name    string
	state   int
	glyph   string
	palette func() color.Color
}{
	{"working", 0, "●", func() color.Color { return BrightGreen }},
	{"awaiting review", 1, "◐", func() color.Color { return Orange }},
	{"ready", 2, "♪", func() color.Color { return BrightGold }},
	{"deferred", 3, "⏸", func() color.Color { return Dim }},
	{"waiting/blocked", 4, "⊘", func() color.Color { return StatusStalled }},
	{"done", 5, "✓", func() color.Color { return Muted }},
}

// TestExecVocabulary checks the six-state vocabulary under both palettes:
// exact glyph, the palette primitive each state binds to, a non-nil color, a
// single-cell indicator, and a bold section header. ThemeDark is restored so
// the rest of the package sees the default palette.
func TestExecVocabulary(t *testing.T) {
	for _, theme := range []struct {
		name  string
		theme Theme
	}{{"dark", ThemeDark}, {"light", ThemeLight}} {
		t.Run(theme.name, func(t *testing.T) {
			SetTheme(theme.theme)
			t.Cleanup(func() { SetTheme(ThemeDark) })

			for _, tc := range execStates {
				t.Run(tc.name, func(t *testing.T) {
					if got := ExecSymbol(tc.state); got != tc.glyph {
						t.Errorf("ExecSymbol(%d) = %q, want %q", tc.state, got, tc.glyph)
					}
					if got, want := ExecColor(tc.state), tc.palette(); got == nil {
						t.Errorf("ExecColor(%d) = nil, want %v", tc.state, want)
					} else if got != want {
						t.Errorf("ExecColor(%d) = %v, want the %v primitive", tc.state, got, want)
					}
					if got := ExecIndicator(tc.state); ansi.StringWidth(got) != 1 {
						t.Errorf("ExecIndicator(%d) = %q, display width %d, want 1",
							tc.state, got, ansi.StringWidth(got))
					}
					if !ExecSectionStyle(tc.state).GetBold() {
						t.Errorf("ExecSectionStyle(%d) is not bold", tc.state)
					}
				})
			}
		})
	}
}

// TestExecIndicatorRebakesOnThemeSwitch pins the reason CLAUDE.md forbids
// capturing pre-rendered strings outside internal/ui: the Exec*Str indicators
// are rendered once in rebuildStyles and must be re-rendered by SetTheme. A
// package-init capture from the dark palette would leave the dark bytes in
// place after SetTheme(ThemeLight) — the frozen-palette bug — and the
// display-width-only assertion in TestExecVocabulary cannot see it. Every
// execution-state color binds to a primitive with a distinct light variant, so
// the light indicator must differ from the dark one for all six states.
func TestExecIndicatorRebakesOnThemeSwitch(t *testing.T) {
	t.Cleanup(func() { SetTheme(ThemeDark) })

	SetTheme(ThemeDark)
	dark := make([]string, len(execStates))
	for _, tc := range execStates {
		dark[tc.state] = ExecIndicator(tc.state)
	}

	SetTheme(ThemeLight)
	for _, tc := range execStates {
		light := ExecIndicator(tc.state)
		if light == dark[tc.state] {
			t.Errorf("ExecIndicator(%d) = %q under both palettes — the pre-rendered string was not rebaked",
				tc.state, light)
		}
		if got, want := ExecColor(tc.state), tc.palette(); got != want {
			t.Errorf("ExecColor(%d) = %v after SetTheme(ThemeLight), want the light %v primitive", tc.state, got, want)
		}
	}
}

// TestExecVocabularyOutOfRange pins the documented fallback. An integer
// outside 0-5 is a broken SemanticState contract, not a seventh state: the
// lookups must not panic and must not render as ready work.
func TestExecVocabularyOutOfRange(t *testing.T) {
	t.Cleanup(func() { SetTheme(ThemeDark) })
	SetTheme(ThemeDark)

	for _, state := range []int{-1, 6, 42} {
		if got := ExecSymbol(state); got != SymMissing {
			t.Errorf("ExecSymbol(%d) = %q, want the missing marker %q", state, got, SymMissing)
		}
		if got := ExecColor(state); got != Muted {
			t.Errorf("ExecColor(%d) = %v, want Muted %v", state, got, Muted)
		}
		if got := ExecIndicator(state); ansi.StringWidth(got) != 1 {
			t.Errorf("ExecIndicator(%d) = %q, display width %d, want 1",
				state, got, ansi.StringWidth(got))
		}
		if !ExecSectionStyle(state).GetBold() {
			t.Errorf("ExecSectionStyle(%d) is not bold", state)
		}
	}
}
