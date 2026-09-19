package digest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseRejectsUnknownSchema(t *testing.T) {
	_, err := Parse([]byte(`{"schema":"project-digest.v0","project":"x"}`))
	if err == nil {
		t.Fatal("unknown schema must fail")
	}
}

func TestParseAcceptsProducerFields(t *testing.T) {
	raw := []byte(`{
		"schema":"project-digest.v1",
		"project":"mardi-gras",
		"revision":"r1",
		"generated_at":"2026-09-19T05:00:00Z",
		"sprint":{"epic":"mard-wte","title":"dogfood","phase":"executing"},
		"now":{"count":2,"frontier":[{"id":"mard-wte.13","title":"badges"}]},
		"working":[{"actor":"zime","work":"mard-wte"}],
		"waiting":[{"reason":"publish","count":1}],
		"awaiting_review":false,
		"for_user":1,
		"recent":[{"id":"mard-wte.1","change":"closed"}],
		"narrative":{"text":"Dogfood is still landing.","revision":"r1","generated_at":"2026-09-19T05:00:01Z"},
		"extra_producer_field":true
	}`)
	d, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Project != "mardi-gras" || d.Now.Count != 2 || d.Narrative.Text == "" {
		t.Fatalf("parse = %+v", d)
	}
}

func TestFreshness(t *testing.T) {
	now := time.Date(2026, 9, 19, 5, 20, 0, 0, time.UTC)
	fresh := Digest{GeneratedAt: now.Add(-3 * time.Minute)}
	if got := fresh.Freshness(now); got != "3m ago" {
		t.Fatalf("fresh = %q", got)
	}
	stale := Digest{GeneratedAt: now.Add(-20 * time.Minute)}
	if got := stale.Freshness(now); got != "stale 20m" {
		t.Fatalf("stale = %q", got)
	}
	if !stale.Stale(now) || fresh.Stale(now) {
		t.Fatal("stale threshold")
	}
	if got := (Digest{}).Freshness(now); got != "unknown" {
		t.Fatalf("zero clock = %q", got)
	}
}

func TestLoadReadsExplicitFileAndNeverTheBoard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "digest.json")
	body := `{"schema":"project-digest.v1","project":"mardi-gras","revision":"r2","generated_at":"2026-09-19T05:00:00Z","now":{"count":1}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MG_PROJECT_DIGEST", path)
	d, err := Load("/no/such/beads", "mardi-gras")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || d.Now == nil || d.Now.Count != 1 {
		t.Fatalf("load = %+v", d)
	}
}

func TestLoadMissIsNil(t *testing.T) {
	t.Setenv("MG_PROJECT_DIGEST", "")
	orig := loadActorCLI
	loadActorCLI = func(string) (*Digest, error) { return nil, nil }
	t.Cleanup(func() { loadActorCLI = orig })
	d, err := Load(t.TempDir(), "mardi-gras")
	if err != nil || d != nil {
		t.Fatalf("unpublished producer must be a miss, got %+v / %v", d, err)
	}
}
