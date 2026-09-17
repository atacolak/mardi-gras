package ui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// execStates pins the shared six-state execution vocabulary to its integer
// order: Ready=0, Working=1, WaitingBlocked=2, Deferred=3, OperatorReview=4,
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
	{"ready", 0, "●", func() color.Color { return SwatchGold }},
	{"working", 1, "◐", func() color.Color { return SwatchGreen }},
	{"waiting/blocked", 2, "⊘", func() color.Color { return SwatchRose }},
	{"deferred", 3, "⏸", func() color.Color { return SwatchLavender }},
	{"operator attention", 4, "○", func() color.Color { return SwatchCyan }},
	{"done", 5, "✓", func() color.Color { return SwatchSteel }},
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

// TestSwatchPaletteHexes pins Ata's labeled six-swatch palette. The dark values
// are the Brief's, as printed on the screenshot; the light values must differ
// (see TestExecIndicatorRebakesOnThemeSwitch) and are the darker same-hue
// variants the spec chose.
func TestSwatchPaletteHexes(t *testing.T) {
	t.Cleanup(func() { SetTheme(ThemeDark) })
	cases := []struct {
		name        string
		get         func() color.Color
		dark, light string
	}{
		{"rose", func() color.Color { return SwatchRose }, "#E06C75", "#8C3A44"},
		{"gold", func() color.Color { return SwatchGold }, "#E5B567", "#8C6A1A"},
		{"cyan", func() color.Color { return SwatchCyan }, "#56B6C2", "#2E6E73"},
		{"green", func() color.Color { return SwatchGreen }, "#7FB069", "#3F6B32"},
		{"lavender", func() color.Color { return SwatchLavender }, "#A78BBA", "#6B4C82"},
		{"steel", func() color.Color { return SwatchSteel }, "#6F8FAF", "#3E5C7E"},
	}
	for _, tc := range cases {
		SetTheme(ThemeDark)
		if got := tc.get(); got != lipgloss.Color(tc.dark) {
			t.Errorf("dark %s = %v, want %s", tc.name, got, tc.dark)
		}
		SetTheme(ThemeLight)
		if got := tc.get(); got != lipgloss.Color(tc.light) {
			t.Errorf("light %s = %v, want %s", tc.name, got, tc.light)
		}
	}
}

// TestExecColorsUseEverySwatch is ask 10's "use EVERY swatch, do not leftover a
// color": each of the six states must be painted with a swatch, no two states
// may share one, and no swatch may go unused.
func TestExecColorsUseEverySwatch(t *testing.T) {
	t.Cleanup(func() { SetTheme(ThemeDark) })
	SetTheme(ThemeDark)

	swatches := map[string]color.Color{
		"rose":     SwatchRose,
		"gold":     SwatchGold,
		"cyan":     SwatchCyan,
		"green":    SwatchGreen,
		"lavender": SwatchLavender,
		"steel":    SwatchSteel,
	}
	used := make(map[string]int, len(swatches))
	for state := 0; state < 6; state++ {
		got := ExecColor(state)
		matched := ""
		for name, swatch := range swatches {
			if got == swatch {
				matched = name
			}
		}
		if matched == "" {
			t.Errorf("ExecColor(%d) = %v, which is not one of Ata's six swatches", state, got)
			continue
		}
		used[matched]++
	}
	for name := range swatches {
		switch used[name] {
		case 1:
		case 0:
			t.Errorf("swatch %s is bound to no state; the Brief forbids a leftover color", name)
		default:
			t.Errorf("swatch %s is bound to %d states; each swatch owns exactly one", name, used[name])
		}
	}
}

// TestExecIndicatorHighlight pins ask 11's "glyph only" highlight: same color,
// same one-cell width, visibly different bytes, and rebaked on theme switch.
func TestExecIndicatorHighlight(t *testing.T) {
	t.Cleanup(func() { SetTheme(ThemeDark) })
	for _, tc := range execStates {
		plain, hi := ExecIndicator(tc.state), ExecIndicatorHighlight(tc.state)
		if plain == hi {
			t.Errorf("ExecIndicatorHighlight(%d) == ExecIndicator(%d); the highlight is invisible", tc.state, tc.state)
		}
		if ansi.StringWidth(hi) != 1 {
			t.Errorf("ExecIndicatorHighlight(%d) width = %d, want 1", tc.state, ansi.StringWidth(hi))
		}
	}
	SetTheme(ThemeDark)
	dark := ExecIndicatorHighlight(0)
	SetTheme(ThemeLight)
	if ExecIndicatorHighlight(0) == dark {
		t.Error("ExecIndicatorHighlight was not rebaked by SetTheme")
	}
}
