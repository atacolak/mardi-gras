package data

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// blockingEdge declares an issue→blocker `blocks` dependency, the type
// DefaultBlockingTypes counts.
func blockingEdge(issue, blocker string) []Dependency {
	return []Dependency{{IssueID: issue, DependsOnID: blocker, Type: "blocks"}}
}

// TestDeriveStateMardNobHardExample is the acceptance test for the defect the
// Brief names: an in_progress epic whose every child is closed has no live work
// left, so it is awaiting operator acceptance — not Working.
func TestDeriveStateMardNobHardExample(t *testing.T) {
	issues := []Issue{{ID: "mard-nob", Status: StatusInProgress, IssueType: TypeEpic}}
	for n := 1; n <= 7; n++ {
		id := fmt.Sprintf("mard-nob.%d", n)
		issues = append(issues, Issue{ID: id, Status: StatusClosed, IssueType: TypeTask, Dependencies: parentEdge(id, "mard-nob")})
	}
	m := BuildIssueMap(issues)
	got, ok := DeriveState(m["mard-nob"], m, DefaultBlockingTypes)
	if !ok {
		t.Fatal("hard-example state must be decidable")
	}
	if got == StateWorking {
		t.Fatal("hard-example epic rendered Working")
	}
	if got != StateAwaitingReview {
		t.Fatalf("got %v, want Awaiting Review", got)
	}
}

// TestDeriveStatePrecedence walks the spec's mapping table one row at a time.
func TestDeriveStatePrecedence(t *testing.T) {
	future := time.Now().Add(72 * time.Hour)
	past := time.Now().Add(-72 * time.Hour)

	tests := []struct {
		name  string
		issue Issue
		extra []Issue
		want  SemanticState
	}{
		{"closed", Issue{ID: "x", Status: StatusClosed}, nil, StateDone},
		{"raw blocked", Issue{ID: "x", Status: StatusBlocked}, nil, StateWaitingBlocked},
		{"raw deferred", Issue{ID: "x", Status: StatusDeferred}, nil, StateDeferred},
		{"working", Issue{ID: "x", Status: StatusInProgress}, nil, StateWorking},
		{"ready", Issue{ID: "x", Status: StatusOpen}, nil, StateReady},
		{
			"future defer",
			Issue{ID: "x", Status: StatusOpen, DeferUntil: &future},
			nil,
			StateDeferred,
		},
		{
			"future defer on in_progress",
			Issue{ID: "x", Status: StatusInProgress, DeferUntil: &future},
			nil,
			StateDeferred,
		},
		{
			"past defer plus open is ready",
			Issue{ID: "x", Status: StatusOpen, DeferUntil: &past},
			nil,
			StateReady,
		},
		{
			"unresolved blocker",
			Issue{ID: "x", Status: StatusOpen, Dependencies: blockingEdge("x", "y")},
			[]Issue{{ID: "y", Status: StatusInProgress}},
			StateWaitingBlocked,
		},
		{
			"dangling blocker",
			Issue{ID: "x", Status: StatusOpen, Dependencies: blockingEdge("x", "y")},
			nil,
			StateWaitingBlocked,
		},
		{
			"resolved blocker is not blocked",
			Issue{ID: "x", Status: StatusOpen, Dependencies: blockingEdge("x", "y")},
			[]Issue{{ID: "y", Status: StatusClosed}},
			StateReady,
		},
		{
			"non-blocking dep type is not blocked",
			Issue{ID: "x", Status: StatusOpen, Dependencies: []Dependency{{IssueID: "x", DependsOnID: "y", Type: "discovered-from"}}},
			[]Issue{{ID: "y", Status: StatusOpen}},
			StateReady,
		},
		{
			"blocked beats future defer",
			Issue{ID: "x", Status: StatusBlocked, DeferUntil: &future},
			nil,
			StateWaitingBlocked,
		},
		{
			"unresolved blocker beats future defer",
			Issue{ID: "x", Status: StatusOpen, DeferUntil: &future, Dependencies: blockingEdge("x", "y")},
			[]Issue{{ID: "y", Status: StatusOpen}},
			StateWaitingBlocked,
		},
		{
			"unresolved blocker beats the epic rule",
			Issue{ID: "x", Status: StatusInProgress, IssueType: TypeEpic, Dependencies: blockingEdge("x", "y")},
			[]Issue{
				{ID: "y", Status: StatusOpen},
				{ID: "x.1", Status: StatusClosed, Dependencies: parentEdge("x.1", "x")},
			},
			StateWaitingBlocked,
		},
		{
			// The epic rule (row 8) outranks deferral (rows 9-10): the epic's
			// own raw status or DeferUntil says nothing about whether its work
			// is finished and waiting on the operator. Swapping the deferred
			// block above the epic block leaves the rest of this table green.
			"deferred epic with every child closed is awaiting review",
			Issue{ID: "x", Status: StatusDeferred, IssueType: TypeEpic},
			[]Issue{{ID: "x.1", Status: StatusClosed, Dependencies: parentEdge("x.1", "x")}},
			StateAwaitingReview,
		},
		{
			"future defer on an epic with every child closed is awaiting review",
			Issue{ID: "x", Status: StatusInProgress, IssueType: TypeEpic, DeferUntil: &future},
			[]Issue{{ID: "x.1", Status: StatusClosed, Dependencies: parentEdge("x.1", "x")}},
			StateAwaitingReview,
		},
		{
			// `parent-child` is hierarchy, not a blocker. Every real child
			// carries one, so if it ever blocked, every nested row on a real
			// board would render Waiting/Blocked instead of its own state.
			"parent-child edge to an open parent is not blocking",
			Issue{ID: "x", Status: StatusOpen, Dependencies: parentEdge("x", "p")},
			[]Issue{{ID: "p", Status: StatusOpen, IssueType: TypeEpic}},
			StateReady,
		},
		{
			"nested epic with an open parent is not blocked",
			Issue{ID: "x", Status: StatusInProgress, IssueType: TypeEpic, Dependencies: parentEdge("x", "p")},
			[]Issue{{ID: "p", Status: StatusOpen, IssueType: TypeEpic}},
			StateWorking,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := append([]Issue{tc.issue}, tc.extra...)
			m := BuildIssueMap(issues)
			got, ok := DeriveState(m["x"], m, DefaultBlockingTypes)
			if !ok {
				t.Fatalf("DeriveState(%q) returned ok=false, want %v", tc.issue.Status, tc.want.Label())
			}
			if got != tc.want {
				t.Fatalf("got %v (%s), want %v (%s)", got, got.Label(), tc.want, tc.want.Label())
			}
		})
	}
}

// TestDeriveStateUnmappedStatusesStayUnmapped is the do-not-collapse invariant:
// a status this wave is not authorised to map is never bucketed, and above all
// is never silently rendered as Ready. Graph facts must not rescue it either —
// rows 2-5 precede every bucket.
func TestDeriveStateUnmappedStatusesStayUnmapped(t *testing.T) {
	future := time.Now().Add(72 * time.Hour)

	for _, status := range []Status{StatusDraft, StatusTombstone, StatusPinned, Status("custom-review")} {
		t.Run(string(status), func(t *testing.T) {
			issues := []Issue{
				{ID: "u", Status: status, IssueType: TypeTask},
				{ID: "u2", Status: status, IssueType: TypeTask, DeferUntil: &future, Dependencies: blockingEdge("u2", "b")},
				{ID: "b", Status: StatusInProgress},
				{ID: "plain", Status: StatusOpen},
			}
			m := BuildIssueMap(issues)

			for _, id := range []string{"u", "u2"} {
				state, ok := DeriveState(m[id], m, DefaultBlockingTypes)
				if ok {
					t.Errorf("%s (%s): ok=true, want unmapped", id, status)
				}
				if state == StateReady {
					t.Errorf("%s (%s): state is Ready, an unmapped status must never collapse into Ready", id, status)
				}
			}

			groups, unmapped := GroupBySemanticState(issues, DefaultBlockingTypes)
			var unmappedIDs []string
			for _, u := range unmapped {
				unmappedIDs = append(unmappedIDs, u.ID)
			}
			if !reflect.DeepEqual(unmappedIDs, []string{"u", "u2"}) {
				t.Errorf("unmapped = %v, want [u u2]", unmappedIDs)
			}
			for _, state := range StateOrder() {
				for _, grouped := range groups[state] {
					if grouped.ID == "u" || grouped.ID == "u2" {
						t.Errorf("%s landed in the %s group", grouped.ID, state.Label())
					}
				}
			}
		})
	}
}

// TestDeriveStateEpicRules pins the boundaries of the settled-descendants rule.
func TestDeriveStateEpicRules(t *testing.T) {
	settledEpic := []Issue{
		{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
		{ID: "e.1", Status: StatusClosed, IssueType: TypeTask, Dependencies: parentEdge("e.1", "e")},
		{ID: "e.2", Status: StatusClosed, IssueType: TypeTask, Dependencies: parentEdge("e.2", "e")},
	}

	tests := []struct {
		name  string
		issue Issue
		extra []Issue
		want  SemanticState
	}{
		{
			"epic with every descendant closed",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			settledEpic[1:],
			StateAwaitingReview,
		},
		{
			"open epic with every descendant closed",
			Issue{ID: "e", Status: StatusOpen, IssueType: TypeEpic},
			settledEpic[1:],
			StateAwaitingReview,
		},
		{
			"epic with zero loaded descendants",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			nil,
			StateWorking,
		},
		{
			"epic with one open descendant",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			[]Issue{
				{ID: "e.1", Status: StatusClosed, Dependencies: parentEdge("e.1", "e")},
				{ID: "e.2", Status: StatusOpen, Dependencies: parentEdge("e.2", "e")},
			},
			StateWorking,
		},
		{
			"a grandchild still counts",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			[]Issue{
				{ID: "e.1", Status: StatusClosed, Dependencies: parentEdge("e.1", "e")},
				{ID: "e.2", Status: StatusOpen, Dependencies: parentEdge("e.2", "e.1")},
			},
			StateWorking,
		},
		{
			"a closed descendant in another epic is not a descendant",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			[]Issue{
				{ID: "other", Status: StatusOpen, IssueType: TypeEpic},
				{ID: "other.1", Status: StatusClosed, Dependencies: parentEdge("other.1", "other")},
			},
			StateWorking,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := append([]Issue{tc.issue}, tc.extra...)
			m := BuildIssueMap(issues)
			got, ok := DeriveState(m["e"], m, DefaultBlockingTypes)
			if !ok {
				t.Fatalf("DeriveState returned ok=false, want %v", tc.want.Label())
			}
			if got == StateAwaitingReview && tc.want != StateAwaitingReview {
				t.Fatalf("epic with %s wrongly settled as Awaiting Review", tc.name)
			}
			if got != tc.want {
				t.Fatalf("got %v (%s), want %v (%s)", got, got.Label(), tc.want, tc.want.Label())
			}
		})
	}
}

// TestDeriveStateEpicWithUnmappedDescendant: whether an unmapped descendant
// counts as settled is exactly what this wave has not decided, so the epic is
// undecidable rather than guessed.
func TestDeriveStateEpicWithUnmappedDescendant(t *testing.T) {
	issues := []Issue{
		{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
		{ID: "e.1", Status: StatusClosed, IssueType: TypeTask, Dependencies: parentEdge("e.1", "e")},
		{ID: "e.2", Status: StatusDraft, IssueType: TypeTask, Dependencies: parentEdge("e.2", "e")},
		{ID: "e.3", Status: StatusClosed, IssueType: TypeTask, Dependencies: parentEdge("e.3", "e")},
	}
	m := BuildIssueMap(issues)

	state, ok := DeriveState(m["e"], m, DefaultBlockingTypes)
	if ok {
		t.Fatalf("epic with a draft descendant returned ok=true (%v), want undecidable", state.Label())
	}
	if state == StateReady {
		t.Fatal("undecidable epic collapsed into Ready")
	}
}

// TestDeriveStateEpicWithOpenAndUnmappedDescendants: undecidability does not
// depend on scan order — an open descendant must not mask a later unmapped one.
func TestDeriveStateEpicWithOpenAndUnmappedDescendants(t *testing.T) {
	issues := []Issue{
		{ID: "e", Status: StatusOpen, IssueType: TypeEpic},
		{ID: "e.1", Status: StatusOpen, IssueType: TypeTask, Dependencies: parentEdge("e.1", "e")},
		{ID: "e.2", Status: StatusPinned, IssueType: TypeTask, Dependencies: parentEdge("e.2", "e")},
	}
	m := BuildIssueMap(issues)

	state, ok := DeriveState(m["e"], m, DefaultBlockingTypes)
	if ok {
		t.Fatalf("epic with a pinned descendant returned ok=true (%v), want undecidable", state.Label())
	}
	if state == StateReady {
		t.Fatal("undecidable epic collapsed into Ready")
	}
}

// TestEpicAwaitingReview tests the settled-descendants rule on its own, without
// the blocked-wins row that DeriveState applies first.
func TestEpicAwaitingReview(t *testing.T) {
	tests := []struct {
		name          string
		issue         Issue
		extra         []Issue
		wantAwaiting  bool
		wantDecidable bool
	}{
		{
			"settled epic",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			[]Issue{{ID: "e.1", Status: StatusClosed, Dependencies: parentEdge("e.1", "e")}},
			true, true,
		},
		{
			"empty epic is not settled",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			nil,
			false, true,
		},
		{
			"open descendant is not settled",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			[]Issue{{ID: "e.1", Status: StatusOpen, Dependencies: parentEdge("e.1", "e")}},
			false, true,
		},
		{
			"draft descendant is undecidable",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic},
			[]Issue{{ID: "e.1", Status: StatusDraft, Dependencies: parentEdge("e.1", "e")}},
			false, false,
		},
		{
			"blocked epic is still settled",
			Issue{ID: "e", Status: StatusInProgress, IssueType: TypeEpic, Dependencies: blockingEdge("e", "b")},
			[]Issue{
				{ID: "b", Status: StatusOpen},
				{ID: "e.1", Status: StatusClosed, Dependencies: parentEdge("e.1", "e")},
			},
			true, true,
		},
		{
			"closed epic is not awaiting review",
			Issue{ID: "e", Status: StatusClosed, IssueType: TypeEpic},
			[]Issue{{ID: "e.1", Status: StatusClosed, Dependencies: parentEdge("e.1", "e")}},
			false, true,
		},
		{
			"a task with settled children is not an epic",
			Issue{ID: "e", Status: StatusOpen, IssueType: TypeTask},
			[]Issue{{ID: "e.1", Status: StatusClosed, Dependencies: parentEdge("e.1", "e")}},
			false, true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := append([]Issue{tc.issue}, tc.extra...)
			m := BuildIssueMap(issues)
			awaiting, decidable := EpicAwaitingReview(m["e"], m)
			if awaiting != tc.wantAwaiting || decidable != tc.wantDecidable {
				t.Fatalf("got (awaiting=%v, decidable=%v), want (awaiting=%v, decidable=%v)",
					awaiting, decidable, tc.wantAwaiting, tc.wantDecidable)
			}
		})
	}
}

// TestStateOrderAndLabel pins the render order and Ata's exact state names.
// The integer order is shared with internal/ui's Exec* vocabulary, so it is
// pinned here too.
func TestStateOrderAndLabel(t *testing.T) {
	want := []struct {
		state SemanticState
		label string
	}{
		{StateWorking, "Working"},
		{StateAwaitingReview, "Awaiting Review"},
		{StateReady, "Ready"},
		{StateDeferred, "Deferred"},
		{StateWaitingBlocked, "Waiting/Blocked"},
		{StateDone, "Done"},
	}

	order := StateOrder()
	if len(order) != len(want) {
		t.Fatalf("StateOrder() has %d states, want %d", len(order), len(want))
	}
	for i, w := range want {
		if order[i] != w.state {
			t.Errorf("StateOrder()[%d] = %d, want %d", i, order[i], w.state)
		}
		if int(w.state) != i {
			t.Errorf("%s has value %d, want %d — internal/ui keys ExecSymbol by this order", w.label, int(w.state), i)
		}
		if got := w.state.Label(); got != w.label {
			t.Errorf("Label(%d) = %q, want %q", w.state, got, w.label)
		}
	}
	if got := SemanticState(99).Label(); got != "" {
		t.Errorf("Label(99) = %q, want empty", got)
	}
}

// TestGroupBySemanticStatePreinitializesAllKeys: callers range StateOrder()
// without a nil-map guard, so every key must exist even when empty.
func TestGroupBySemanticStatePreinitializesAllKeys(t *testing.T) {
	issues := []Issue{
		{ID: "a", Status: StatusOpen},
		{ID: "b", Status: StatusDraft},
	}

	groups, unmapped := GroupBySemanticState(issues, DefaultBlockingTypes)

	if len(groups) != len(StateOrder()) {
		t.Errorf("groups has %d keys, want %d", len(groups), len(StateOrder()))
	}
	for _, state := range StateOrder() {
		if _, exists := groups[state]; !exists {
			t.Errorf("group %s missing from map", state.Label())
		}
	}
	if len(groups[StateReady]) != 1 || groups[StateReady][0].ID != "a" {
		t.Errorf("Ready group = %v, want [a]", issueIDs(groups[StateReady]))
	}
	if len(unmapped) != 1 || unmapped[0].ID != "b" {
		t.Errorf("unmapped = %v, want [b]", issueIDs(unmapped))
	}
}
