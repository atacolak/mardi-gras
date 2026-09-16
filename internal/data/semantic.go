package data

// SemanticState is the operator-facing execution state mg derives from the
// issue graph. It is not a Beads status: raw status alone cannot tell a
// finished-but-unaccepted epic from one that is still being worked on.
type SemanticState int

// Order is render order: the four pre-existing parade sections keep their
// relative order, and each new state follows the state it derives from.
// internal/ui keys ExecSymbol/ExecColor/ExecSectionStyle by this integer
// order, so the constants are a shared contract, not an implementation detail.
const (
	StateWorking SemanticState = iota
	StateAwaitingReview
	StateReady
	StateDeferred
	StateWaitingBlocked
	StateDone
)

// Label returns Ata's exact state name.
func (s SemanticState) Label() string {
	switch s {
	case StateWorking:
		return "Working"
	case StateAwaitingReview:
		return "Awaiting Review"
	case StateReady:
		return "Ready"
	case StateDeferred:
		return "Deferred"
	case StateWaitingBlocked:
		return "Waiting/Blocked"
	case StateDone:
		return "Done"
	}
	return ""
}

// StateOrder is the single source of render order for every surface. It lists
// each state exactly once, in the order the constants declare.
func StateOrder() []SemanticState {
	return []SemanticState{
		StateWorking,
		StateAwaitingReview,
		StateReady,
		StateDeferred,
		StateWaitingBlocked,
		StateDone,
	}
}

// mappedStatus reports whether this wave is authorised to decide a semantic
// state for a raw status. Recognizing a status is not the same as deciding its
// state: draft, tombstone, pinned, and any custom status are recognized as
// Beads statuses (see StatusDraft and friends) but deliberately unmapped here,
// because which of the six they should render as is an open product question.
func mappedStatus(s Status) bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusBlocked, StatusDeferred, StatusClosed:
		return true
	}
	return false
}

// DeriveState classifies one issue from the dependency / parent-child graph,
// implementing the spec's mapping table top to bottom — first match wins.
//
// ok is false when the raw status is outside the set this wave is authorised
// to map (draft, tombstone, pinned, and any custom status), and when an epic's
// descendants include such a status: such an issue is never silently bucketed,
// and in particular never becomes StateReady. Rows 2-5 are checked before every
// bucket, so an unmapped status cannot leak in through a later rule.
func DeriveState(i *Issue, issueMap map[string]*Issue, blockingTypes map[string]bool) (SemanticState, bool) {
	// Rows 1-5: the raw status decides first, before any graph-derived bucket.
	if i.Status == StatusClosed {
		return StateDone, true
	}
	if !mappedStatus(i.Status) {
		return StateWorking, false
	}

	// Row 6: blocked wins over every remaining bucket — an unresolved blocker
	// (and a blocker that is not loaded at all) outranks Working, Deferred, and
	// the epic rule alike.
	if i.EvaluateDependencies(issueMap, blockingTypes).IsBlocked {
		return StateWaitingBlocked, true
	}

	// Row 7: the raw status still counts on its own.
	if i.Status == StatusBlocked {
		return StateWaitingBlocked, true
	}

	// Row 8, carrying row 13 with it: the settled-descendants rule cannot be
	// evaluated honestly when a descendant has a status this wave does not map,
	// so such an epic is unmapped rather than guessed. Blocked-wins has already
	// been applied above, so a blocked epic never reaches here.
	if i.IssueType == TypeEpic {
		awaiting, decidable := EpicAwaitingReview(i, issueMap)
		if !decidable {
			return StateWorking, false
		}
		if awaiting {
			return StateAwaitingReview, true
		}
	}

	// Rows 9-10: deferral is visible on the row badge whether or not it is the
	// derived state, so blocked already having won is non-lossy.
	if i.Status == StatusDeferred || i.IsDeferred() {
		return StateDeferred, true
	}

	// Rows 11-12.
	if i.Status == StatusInProgress {
		return StateWorking, true
	}
	if i.Status == StatusOpen {
		return StateReady, true
	}

	// Unreachable for the mapped set, but returning a bucket here would let an
	// unrecognized status leak into one, so the conservative default stands.
	return StateWorking, false
}

// EpicAwaitingReview reports the Brief's settled-descendants rule: an epic that
// is not itself closed, has at least one loaded descendant, and whose every
// executable descendant is closed, is awaiting operator acceptance.
//
// decidable is false when a descendant carries a status DeriveState cannot
// classify, so the epic's own state cannot be honestly computed either. An
// empty epic is not settled: `br epic status` withholds eligibility at 0/0.
func EpicAwaitingReview(i *Issue, issueMap map[string]*Issue) (awaiting, decidable bool) {
	if i.IssueType != TypeEpic || i.Status == StatusClosed {
		return false, true
	}

	descendants := Descendants(i.ID, issueMap)
	if len(descendants) == 0 {
		return false, true
	}

	settled := true
	for _, d := range descendants {
		if d.Status == StatusClosed {
			continue
		}
		if !mappedStatus(d.Status) {
			// Scan every descendant before answering: an unmapped descendant
			// makes the epic undecidable even when another one is plainly open.
			return false, false
		}
		settled = false
	}
	return settled, true
}

// GroupBySemanticState buckets issues by derived state, preserving input order
// within each bucket. Issues DeriveState cannot classify are excluded from
// every bucket and returned separately, so an unmapped status can never be
// counted as work or as ready.
//
// All six keys are pre-initialized, so a caller can range StateOrder() without
// a nil-map guard.
func GroupBySemanticState(issues []Issue, blockingTypes map[string]bool) (groups map[SemanticState][]Issue, unmapped []Issue) {
	issueMap := BuildIssueMap(issues)

	groups = make(map[SemanticState][]Issue, len(StateOrder()))
	for _, state := range StateOrder() {
		groups[state] = []Issue{}
	}

	for i := range issues {
		state, ok := DeriveState(&issues[i], issueMap, blockingTypes)
		if !ok {
			unmapped = append(unmapped, issues[i])
			continue
		}
		groups[state] = append(groups[state], issues[i])
	}
	return groups, unmapped
}
