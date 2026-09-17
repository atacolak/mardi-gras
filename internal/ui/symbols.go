package ui

import (
	"fmt"
	"strings"
)

// Unicode symbols for the Mardi Gras theme.
const (
	FleurDeLis = "⚜"

	// Status indicators
	SymRolling = "●"
	SymStalled = "⊘"
	SymPassed  = "✓"

	// Bead string
	BeadRound   = "●"
	BeadDiamond = "◆"
	BeadDash    = "─"

	// Navigation
	Cursor = ">"

	// Dependencies
	DepArrow       = "→"
	DepTree        = "└─"
	SymMissing     = "!"
	SymResolved    = "✓" // alias of SymPassed
	SymNonBlocking = "·"
	SymNextArrow   = "next →"
	SymAgent       = "⚡"
	SymConvoy      = "◐"
	SymMail        = "✉"
	SymSling       = "➤"
	SymChanged     = "◈"
	SymSelected    = "◉"
	SymUnselected  = "○"

	// Comments
	SymComment = "💬"

	// Due dates
	SymOverdue  = "▲"
	SymDeferred = "⏸"
	SymDueDate  = "◷"

	// Rich dependency types
	SymRelated    = "↔"
	SymDuplicates = "⊜"
	SymSupersedes = "⇢"

	// Section borders (rounded)
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"

	// Separators
	DividerH = "━"
	DividerV = "│"
	CornerTL = "┯"
	CornerBL = "┷"

	// Gas Town panel
	SymIdle          = "○"
	SymWorking       = "●"
	SymBackoff       = "◌"
	SymStuck         = "⚠" // alias of SymWarning — agent requesting help
	SymSpawning      = "◐" // half-filled — session starting
	SymGate          = "◷" // alias of SymDueDate — waiting on gate
	SymPaused        = "⏸" // alias of SymDeferred — intentionally suspended
	SymFixNeeded     = "🔧" // review feedback — needs rework
	SymPropelled     = "⚡" // ACP propulsion — output suppressed
	SymPatrolling    = "⊙" // witness/deacon scanning rounds
	SymProgress      = "█"
	SymProgressEmpty = "░"
	SymDog           = "🐕"
	SymTown          = "⛽"

	// Problems
	SymWarning = "⚠"
	SymDeadRig = "💀"
	SymZombie  = "☠"

	// Molecule steps
	SymStepDone    = "✓"
	SymStepActive  = "●"
	SymStepReady   = "○"
	SymStepBlocked = "⊘"
	SymStepSkipped = "─"
	SymTierLine    = "│"

	// DAG flow connectors
	SymDAGFlow   = "│"
	SymDAGBranch = "┌"
	SymDAGFork   = "├"
	SymDAGJoin   = "└"
	SymDAGArrow  = "↓"

	// Geometric indicators
	SymDiamond = "◆"
	// Semantic execution states (see ExecSymbol). Deliberately distinct from
	// the Gas Town agent-state symbols above: an issue's derived execution
	// state is not an agent's state.
	SymExecReady          = "○"
	SymExecWorking        = "●"
	SymExecWaiting        = "⊘"
	SymExecDeferred       = "⏸"
	SymExecOperatorReview = "◐"
	SymExecDone           = "✓"
)

// ExecSymbol returns the glyph for a semantic execution state. state is the
// data package's SemanticState integer order — Ready=0, Working=1,
// WaitingBlocked=2, Deferred=3, OperatorReview=4, Done=5 — because ui is a
// leaf package and does not import that type.
//
// A value outside 0-5 is a broken contract, not a seventh state: it renders
// the existing "missing" marker so the row reads as unrecognized rather than
// masquerading as ready work.
func ExecSymbol(state int) string {
	switch state {
	case 0:
		return SymExecReady
	case 1:
		return SymExecWorking
	case 2:
		return SymExecWaiting
	case 3:
		return SymExecDeferred
	case 4:
		return SymExecOperatorReview
	case 5:
		return SymExecDone
	default:
		return SymMissing
	}
}

// superscriptDigits maps 0-9 to their Unicode superscript equivalents.
var superscriptDigits = [10]string{"⁰", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"}

// Superscript converts a non-negative integer to superscript Unicode digits.
func Superscript(n int) string {
	if n < 0 {
		n = 0
	}
	if n < 10 {
		return superscriptDigits[n]
	}
	var b strings.Builder
	for _, d := range fmt.Appendf(nil, "%d", n) {
		b.WriteString(superscriptDigits[d-'0'])
	}
	return b.String()
}
