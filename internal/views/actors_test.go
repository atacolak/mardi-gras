package views

import (
	"errors"
	"strings"
	"testing"

	"github.com/matt-wright86/mardi-gras/internal/actors"
)

func actorFixture() []actors.SocietyRow {
	now := "SPRINT: mard-nob operator Beads/actor observability UI"
	asked := "reko -> zime: operator UI"
	return []actors.SocietyRow{{
		Name:          "zime",
		Role:          "project-lead",
		Kind:          "canonical",
		Lifecycle:     "active",
		Readiness:     "ready",
		Activity:      actors.Activity{Now: &now, Asked: &asked},
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
	a.SetErr(errors.New("actor project: exit status 1"))
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

func TestActorsViewVillageGrouping(t *testing.T) {
	a := NewActors(80, 24)
	a.SetScope(ActorsScopeVillage)
	a.SetVillage([]actors.ProjectGroup{
		{Project: actors.ProjectRef{ID: "mard", Root: "/home/sf/workspace/mardi-gras"}, Actors: actorFixture()},
		{Project: actors.ProjectRef{ID: "oh-my-pi", Root: "/home/sf/workspace/oh-my-pi"}, Actors: []actors.SocietyRow{
			{Name: "tori", Kind: "sibling", Lifecycle: "idle", Readiness: "ready"},
		}},
	})
	v := a.View()
	for _, want := range []string{"mard", "zime", "oh-my-pi", "tori"} {
		if !strings.Contains(v, want) {
			t.Errorf("village View() missing %q:\n%s", want, v)
		}
	}
}

// The pane lives in the detail slot (a third of the terminal), so at narrow
// widths the decorative sprint title must give way before the activity the
// operator opened the pane for.
func TestActorsViewNarrowWidthKeepsActivity(t *testing.T) {
	a := NewActors(60, 24)
	a.SetRows(actorFixture())
	v := a.View()
	if !strings.Contains(v, "now: SPRINT") {
		t.Errorf("activity dropped at narrow width:\n%s", v)
	}
}

// A failed poll must keep the last good roster on screen and report the
// failure alongside it, rather than blanking the pane.
func TestActorsViewKeepsLastGoodRowsWhenPollFails(t *testing.T) {
	a := NewActors(80, 24)
	a.SetRows(actorFixture())
	a.SetErr(errors.New("actor project: timeout"))
	v := a.View()
	if !strings.Contains(v, "zime") {
		t.Errorf("last good roster dropped on poll failure:\n%s", v)
	}
	if !strings.Contains(v, "timeout") {
		t.Errorf("poll failure not reported:\n%s", v)
	}
}

// A fresh attempt returns the pane to its loading line instead of leaving a
// stale error up while the fetch runs.
func TestActorsClearErrResumesLoading(t *testing.T) {
	a := NewActors(80, 24)
	a.SetErr(errors.New("actor project: timeout"))
	a.ClearErr()
	if !a.Loading() {
		t.Error("expected the pane to be loading again after the error was cleared")
	}
	if v := a.View(); !strings.Contains(v, "Loading actors") {
		t.Errorf("want the loading line back, got:\n%s", v)
	}
}
