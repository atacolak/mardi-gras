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
