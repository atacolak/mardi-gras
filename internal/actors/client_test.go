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
	if r.Project.ID != "mardi-gras" {
		t.Errorf("project.id wrong: %+v", r.Project)
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

func TestFetchVillageNotOK(t *testing.T) {
	defer stubRun(`{"ok":false,"count":0,"projects":[]}`)()
	if _, err := FetchVillage(); err == nil {
		t.Fatal("want error on ok:false envelope")
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

func TestFetchStatusNotOK(t *testing.T) {
	defer stubRun(`{"ok":false}`)()
	if _, err := FetchStatus("zime"); err == nil {
		t.Fatal("want error on ok:false envelope")
	}
}

func TestAvailableMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir: no actor on PATH
	if Available() {
		t.Fatal("Available() = true with empty PATH, want false")
	}
}
