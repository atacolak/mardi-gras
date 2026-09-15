package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/matt-wright86/mardi-gras/internal/actors"
	"github.com/matt-wright86/mardi-gras/internal/ui"
)

// ActorsScope selects which listing the pane shows: the current project's
// actors, or the village-wide grouping.
type ActorsScope int

const (
	ActorsScopeProject ActorsScope = iota
	ActorsScopeVillage
)

// Actors is the read-only actor society pane. It renders the `actor` CLI's
// societyRow projection in place of the detail pane. The app owns the poll
// (see the actors poll in internal/app) and pushes results in through the
// setters, so the pane is nil-safe before the first fetch lands and reports a
// failure without dropping the roster it already has — the same contract the
// Gas Town panel has.
type Actors struct {
	width, height int
	scrollOff     int // manual scroll window (not a viewport)
	scope         ActorsScope
	rows          []actors.SocietyRow
	village       []actors.ProjectGroup
	err           error
	loadingFrame  string // spinner glyph while the first fetch is in flight
	fetched       bool   // true once a poll has delivered data for the scope
}

// NewActors creates the actors pane at the given dimensions.
func NewActors(width, height int) Actors {
	return Actors{width: width, height: height}
}

// SetSize updates dimensions.
func (a *Actors) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// SetRows stores a project-scope poll result.
func (a *Actors) SetRows(rows []actors.SocietyRow) {
	a.rows, a.err, a.fetched = rows, nil, true
}

// SetVillage stores a village-scope poll result.
func (a *Actors) SetVillage(groups []actors.ProjectGroup) {
	a.village, a.err, a.fetched = groups, nil, true
}

// SetErr records a poll failure. Any roster already on screen stays; the
// failure is rendered alongside it.
func (a *Actors) SetErr(err error) {
	a.err = err
}

// ClearErr drops a recorded failure so a fresh attempt shows the loading line
// instead of a stale error while it runs.
func (a *Actors) ClearErr() {
	a.err = nil
}

// Loading reports whether the pane is still waiting on its first successful
// fetch for the active scope — the window the app animates its spinner for.
func (a Actors) Loading() bool {
	return !a.fetched && a.err == nil
}

// SetLoadingFrame sets the spinner glyph shown on the loading line while the
// first fetch is in flight. Pass "" when not loading.
func (a *Actors) SetLoadingFrame(frame string) {
	a.loadingFrame = frame
}

// SetScope selects the project-only or village-wide listing. The new scope's
// roster has not arrived yet, so the pane goes back to loading and drops its
// scroll window; the fetch the app schedules behind the toggle fills it.
func (a *Actors) SetScope(s ActorsScope) {
	if a.scope == s {
		return
	}
	a.scope = s
	a.fetched = false
	a.scrollOff = 0
}

// Scope returns the active listing scope.
func (a Actors) Scope() ActorsScope {
	return a.scope
}

// ToggleScope flips project ↔ village.
func (a *Actors) ToggleScope() {
	if a.scope == ActorsScopeVillage {
		a.SetScope(ActorsScopeProject)
		return
	}
	a.SetScope(ActorsScopeVillage)
}

// ScrollBy moves the manual scroll window by delta lines, clamped to the
// content actually rendered.
func (a *Actors) ScrollBy(delta int) {
	lines := len(strings.Split(a.content(), "\n"))
	a.scrollOff = min(max(a.scrollOff+delta, 0), max(lines-1, 0))
}

// View renders the pane, windowed by the manual scroll offset like the Gas
// Town panel.
func (a Actors) View() string {
	lines := strings.Split(a.content(), "\n")
	off := min(max(a.scrollOff, 0), max(len(lines)-1, 0))
	return ui.DetailBorder.Height(a.height).Render(strings.Join(lines[off:], "\n"))
}

// content renders the whole pane body as one string.
func (a Actors) content() string {
	return strings.Join(a.body(), "\n")
}

// body returns the pane one element per rendered line, so the scroll window
// and its bounds are derived from exactly the lines the renderer draws.
func (a Actors) body() []string {
	muted := lipgloss.NewStyle().Foreground(ui.Muted)
	alert := lipgloss.NewStyle().Foreground(ui.StatusStalled)

	scope, other := "project", "village"
	if a.scope == ActorsScopeVillage {
		scope, other = "village", "project"
	}
	lines := []string{
		ui.DetailTitle.Render("ACTORS") + muted.Render(" · "+scope),
		muted.Render(fmt.Sprintf("v: %s · j/k: scroll · o: close", other)),
	}

	switch {
	case a.err != nil && !a.fetched:
		lines = append(lines, alert.Render("actor CLI: "+a.err.Error()))
	case !a.fetched:
		lines = append(lines, muted.Render(strings.TrimSpace(a.loadingFrame)+" Loading actors…"))
	case a.scope == ActorsScopeVillage:
		if len(a.village) == 0 {
			lines = append(lines, muted.Render("No actors in the village."))
		}
		for _, g := range a.village {
			// DetailSection carries a leading margin, so the blank separator
			// between project groups comes from the style itself.
			lines = append(lines, ui.DetailSection.Render(projectLabel(g.Project)))
			for _, r := range g.Actors {
				lines = append(lines, renderSocietyRow(r, a.width))
			}
		}
	default:
		if len(a.rows) == 0 {
			lines = append(lines, muted.Render("No actors for this project."))
		}
		for _, r := range a.rows {
			lines = append(lines, renderSocietyRow(r, a.width))
		}
	}

	if a.err != nil && a.fetched {
		lines = append(lines, "", alert.Render("last poll failed: "+a.err.Error()))
	}
	return lines
}

func projectLabel(p actors.ProjectRef) string {
	if p.ID != "" {
		return p.ID
	}
	if p.Root != "" {
		return p.Root
	}
	return "-"
}

// renderSocietyRow renders one actor as a single line:
//
//	zime  canonical  active/ready  mard-nob — operator UI  now: Running hub
//
// Every activity field and the sprint are null-able in the CLI's projection, so
// each is rendered only when present; a missing sprint shows a dash rather than
// collapsing the column.
//
// The pane lives in the detail slot, which is barely a third of the terminal,
// so the line is fitted by priority rather than clipped: the sprint title is
// decoration and goes first, then the line is truncated from the tail. Asked is
// rendered after now so that the live step — the thing an operator scans for —
// survives a truncation that the stale request does not.
func renderSocietyRow(r actors.SocietyRow, width int) string {
	line := societyLine(r, true)
	if width > 0 && ansi.StringWidth(line) > width-2 {
		line = societyLine(r, false)
	}
	if width > 0 {
		line = ansi.Truncate(line, max(width-2, 8), "…")
	}
	return line
}

// societyLine joins the row's columns; withSprintTitle drops the sprint title
// (keeping the epic) when the pane is too narrow for the full row.
func societyLine(r actors.SocietyRow, withSprintTitle bool) string {
	sprint := "-"
	if r.CurrentSprint != nil {
		sprint = r.CurrentSprint.Epic
		if sprint == "" {
			sprint = "-"
		}
		if withSprintTitle && r.CurrentSprint.Title != "" {
			sprint += " — " + r.CurrentSprint.Title
		}
	}

	fields := []string{r.Name}
	if r.Kind != "" {
		fields = append(fields, r.Kind)
	}
	switch {
	case r.Lifecycle != "" && r.Readiness != "":
		fields = append(fields, r.Lifecycle+"/"+r.Readiness)
	case r.Lifecycle != "":
		fields = append(fields, r.Lifecycle)
	case r.Readiness != "":
		fields = append(fields, r.Readiness)
	}
	fields = append(fields, sprint)
	for _, a := range []struct {
		label string
		value *string
	}{{"now", r.Activity.Now}, {"asked", r.Activity.Asked}} {
		if s := activityText(a.label, a.value); s != "" {
			fields = append(fields, s)
		}
	}
	return strings.Join(fields, "  ")
}

func activityText(label string, v *string) string {
	if v == nil || *v == "" {
		return ""
	}
	return label + ": " + *v
}
