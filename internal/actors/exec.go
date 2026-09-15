package actors

import (
	"context"
	"os/exec"
	"time"
)

// Timeout tiers mirror internal/data/exec.go. Village listing is the slow one
// (observed up to ~10s on a live village); project/status are fast.
const (
	timeoutShort  = 5 * time.Second
	timeoutMedium = 15 * time.Second
)

// runWithTimeout executes the actor CLI with a context timeout and returns
// stdout. Declared as a var so tests can stub it (same seam as
// internal/data's mockRun).
var runWithTimeout = func(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, "actor", args...).Output()
}
