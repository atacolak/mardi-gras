// Package digest is Mardi Gras's read-only consumer of the shared project
// summary owned by midi/worlds (omp-jkp9.11). It never invents a board
// summarizer: if the producer has not published a digest, there is nothing
// to render.
package digest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SchemaV1 is the consumer contract. Extra JSON fields are ignored. The
// producer (worlds / omp-jkp9.11) owns the real schema; this is the slice
// Mardi Gras will render.
const SchemaV1 = "project-digest.v1"

// StaleAfter marks a digest as stale for the freshness chip. The producer
// is the clock; we only report age.
const StaleAfter = 15 * time.Minute

// Digest is the shared project summary. Facts come from beads + ownership +
// compiled attention. Narrative is optional compression, never truth.
type Digest struct {
	Schema         string     `json:"schema"`
	Project        string     `json:"project"`
	Revision       string     `json:"revision"`
	GeneratedAt    time.Time  `json:"generated_at"`
	Sprint         *Sprint    `json:"sprint,omitempty"`
	Now            *Now       `json:"now,omitempty"`
	Working        []Working  `json:"working,omitempty"`
	Waiting        []Waiting  `json:"waiting,omitempty"`
	AwaitingReview bool       `json:"awaiting_review"`
	ForUser        int        `json:"for_user"`
	Recent         []Change   `json:"recent,omitempty"`
	Narrative      *Narrative `json:"narrative,omitempty"`
}

// Sprint is the current owned epic.
type Sprint struct {
	Epic  string `json:"epic"`
	Title string `json:"title"`
	Phase string `json:"phase"`
}

// Now is the executable frontier the lead can act on.
type Now struct {
	Count    int        `json:"count"`
	Frontier []Frontier `json:"frontier,omitempty"`
}

// Frontier is one ready/now bead.
type Frontier struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Working is an actor actually progressing work.
type Working struct {
	Actor string `json:"actor"`
	Work  string `json:"work"`
}

// Waiting is a hold with a reason label (running, blocked, peer, publish, …).
type Waiting struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
	ID     string `json:"id,omitempty"`
}

// Change is a small recent-settled item.
type Change struct {
	ID     string `json:"id"`
	Change string `json:"change"`
}

// Narrative is the optional @smol compression keyed by digest revision.
type Narrative struct {
	Text        string    `json:"text"`
	Revision    string    `json:"revision"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Parse unmarshals a producer payload. A missing/empty schema is accepted so
// an early producer can land without waiting on this consumer; a non-empty
// unknown schema is rejected so we do not silently render a different document.
func Parse(raw []byte) (*Digest, error) {
	var d Digest
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("project digest: %w", err)
	}
	if d.Schema != "" && d.Schema != SchemaV1 {
		return nil, fmt.Errorf("project digest: unknown schema %q", d.Schema)
	}
	if d.Schema == "" {
		d.Schema = SchemaV1
	}
	return &d, nil
}

// Age is time since GeneratedAt. Zero GeneratedAt means unknown.
func (d Digest) Age(now time.Time) time.Duration {
	if d.GeneratedAt.IsZero() {
		return 0
	}
	age := now.Sub(d.GeneratedAt)
	if age < 0 {
		return 0
	}
	return age
}

// Stale reports whether the digest is older than StaleAfter.
func (d Digest) Stale(now time.Time) bool {
	if d.GeneratedAt.IsZero() {
		return true
	}
	return d.Age(now) >= StaleAfter
}

// Freshness is a short operator chip: "3m ago", "stale 16m", or "unknown".
func (d Digest) Freshness(now time.Time) string {
	if d.GeneratedAt.IsZero() {
		return "unknown"
	}
	age := d.Age(now)
	label := formatAge(age)
	if d.Stale(now) {
		return "stale " + label
	}
	if age < 10*time.Second {
		return "just now"
	}
	return label + " ago"
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		sec := int(d.Seconds())
		if sec < 1 {
			sec = 1
		}
		return fmt.Sprintf("%ds", sec)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// Load reads the shared digest. It never inspects the Beads board.
//
// Order:
//  1. MG_PROJECT_DIGEST (explicit file)
//  2. <projectDir>/.omp/project-digest.json
//  3. `actor digest --json <projectID>` when that verb exists
//
// A miss is (nil, nil). A parse/IO failure is an error.
func Load(projectDir, projectID string) (*Digest, error) {
	if path := strings.TrimSpace(os.Getenv("MG_PROJECT_DIGEST")); path != "" {
		return loadFile(path)
	}
	if projectDir != "" {
		path := filepath.Join(projectDir, ".omp", "project-digest.json")
		if raw, err := os.ReadFile(path); err == nil {
			return Parse(raw)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("project digest file: %w", err)
		}
	}
	return loadActorCLI(projectID)
}

func loadFile(path string) (*Digest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("project digest file: %w", err)
	}
	return Parse(raw)
}

// loadActorCLI is a var so tests can stub the producer surface.
var loadActorCLI = func(projectID string) (*Digest, error) {
	if _, err := exec.LookPath("actor"); err != nil {
		return nil, nil
	}
	if projectID == "" {
		return nil, nil
	}
	out, err := exec.Command("actor", "digest", "--json", projectID).Output()
	if err != nil {
		// Unknown verb / gated producer: treat as unpublished, not a crash.
		return nil, nil
	}
	return Parse(out)
}
