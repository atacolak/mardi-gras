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
	StateReady SemanticState = iota
	StateWorking
	StateWaitingBlocked
	StateDeferred
	StateOperatorReview
	StateDone
)

// Label returns Ata's exact state name.
func (s SemanticState) Label() string {
	switch s {
	case StateReady:
		return "Ready"
	case StateWorking:
		return "Working"
	case StateWaitingBlocked:
		return "Waiting/Blocked"
	case StateDeferred:
		return "Deferred"
	case StateOperatorReview:
		return "Operator Attention"
	case StateDone:
		return "Done"
	}
	return ""
}

// StateOrder is the single source of render order for every surface. It lists
// each state exactly once, in the order the constants declare.
func StateOrder() []SemanticState {
	return []SemanticState{
		StateReady,
		StateWorking,
		StateWaitingBlocked,
		StateDeferred,
		StateOperatorReview,
		StateDone,
	}
}

// AttentionRank is the sibling-sort order for the parade's default mode:
// Operator Attention, Waiting/Blocked, Working, Ready, Deferred, Done. It is
// deliberately not StateOrder() — that stays the header/tally render order, and
// the two must not be collapsed into one list.
func AttentionRank(s SemanticState) int {
	switch s {
	case StateOperatorReview:
		return 0
	case StateWaitingBlocked:
		return 1
	case StateWorking:
		return 2
	case StateReady:
		return 3
	case StateDeferred:
		return 4
	case StateDone:
		return 5
	}
	return 6
}

// mappedStatus reports whether this wave is authorised to decide a semantic
// state for a raw status. Recognizing a status is not the same as deciding its
// state: draft, tombstone, pinned, and any custom status are recognized as
// Beads statuses (see StatusDraft and friends) but deliberately unmapped here,
// because which of the six they should render as is an open product question.
func mappedStatus(s Status) bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusBlocked, StatusDeferred, StatusClosed, StatusReview:
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
	if i.Status == StatusClosed {
		return StateDone, true
	}
	if !mappedStatus(i.Status) {
		return StateWorking, false
	}

	// Stored review is authoritative and must be decided before blockers.
	if i.Status == StatusReview {
		return StateOperatorReview, true
	}

	// Blocked wins over every remaining bucket — an unresolved blocker (and a
	// blocker that is not loaded at all) outranks Working, Deferred, and the
	// legacy epic compatibility rule alike.
	if i.EvaluateDependencies(issueMap, blockingTypes).IsBlocked {
		return StateWaitingBlocked, true
	}
	if i.Status == StatusBlocked {
		return StateWaitingBlocked, true
	}

	// Compatibility for already-converged epics created before review became a
	// persisted status. Delete this branch once mard-nob and mard-r43 are stored
	// with status review; direct review above is the permanent path.
	if i.IssueType == TypeEpic && i.Status != StatusReview {
		operatorReview, decidable := legacyConvergedEpicOperatorReview(i, issueMap)
		if !decidable {
			return StateWorking, false
		}
		if operatorReview {
			return StateOperatorReview, true
		}
	}

	if i.Status == StatusDeferred || i.IsDeferred() {
		return StateDeferred, true
	}
	if i.Status == StatusInProgress {
		return StateWorking, true
	}
	if i.Status == StatusOpen {
		return StateReady, true
	}

	return StateWorking, false
}

// legacyConvergedEpicOperatorReview preserves the settled-descendants rule for
// non-review epics created before review became a persisted status. Delete this
// shim once mard-nob and mard-r43 are stored with status review.
func legacyConvergedEpicOperatorReview(i *Issue, issueMap map[string]*Issue) (operatorReview, decidable bool) {
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
