// Package actors is mg's read-only client for the actor society observation
// CLI (`actor project|list|status --json`). It exposes the societyRow
// projection only — no control verbs live here by design.
package actors

// ProjectRef identifies a project as the actor CLI reports it. Root is the
// filesystem root; ID is the project slug.
type ProjectRef struct {
	Root string `json:"root"`
	ID   string `json:"id"`
}

// Activity is the operator-facing activity capsule. Every field is optional;
// a null in the JSON stays a nil pointer so the UI can distinguish "absent"
// from "empty string".
type Activity struct {
	SessionTitle *string `json:"session_title"`
	CurrentStep  *string `json:"current_step"`
	Todo         *string `json:"todo"`
	Asked        *string `json:"asked"`
	Now          *string `json:"now"`
}

// SprintRef is the epic a persistent actor currently owns.
type SprintRef struct {
	Epic  string `json:"epic"`
	Title string `json:"title"`
}

// SocietyRow is one actor's row in the society projection.
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

// ProjectGroup is one project's actors, as returned by the village listing.
type ProjectGroup struct {
	Project ProjectRef   `json:"project"`
	Actors  []SocietyRow `json:"actors"`
}
