package data

import (
	"slices"
	"testing"
)

// parentEdge declares the only thing that establishes hierarchy in Beads: a
// `parent-child` dependency edge. Dotted IDs are paint and carry no authority.
func parentEdge(child, parent string) []Dependency {
	return []Dependency{{IssueID: child, DependsOnID: parent, Type: "parent-child"}}
}

// idsOf renders a pointer slice as a sorted ID list so assertions over
// map-derived results (whose order is not part of the contract) stay stable.
func idsOf(issues []*Issue) []string {
	ids := make([]string, 0, len(issues))
	for _, iss := range issues {
		ids = append(ids, iss.ID)
	}
	slices.Sort(ids)
	return ids
}

func TestChildrenOfIgnoresDottedPaint(t *testing.T) {
	// "mard-nob.2" looks like a child of "mard-nob" but has no edge;
	// "mard-nob.3" was reparented (edge now names mg-other) while keeping its
	// old dotted ID; "mg-900" is an edge-only child with no dotted prefix.
	issues := []Issue{
		{ID: "mard-nob", IssueType: TypeEpic},
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
		{ID: "mard-nob.2"},
		{ID: "mard-nob.3", Dependencies: parentEdge("mard-nob.3", "mg-other")},
		{ID: "mg-900", Dependencies: parentEdge("mg-900", "mard-nob")},
	}

	got := ChildrenOf("mard-nob", issues)
	want := []string{"mard-nob.1", "mg-900"}
	if ids := idsOf(got); !slices.Equal(ids, want) {
		t.Fatalf("ChildrenOf(mard-nob) = %v, want %v", ids, want)
	}
}

func TestChildrenOfNoMatches(t *testing.T) {
	issues := []Issue{
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
		{ID: "mard-nob.2"},
	}

	if got := ChildrenOf("mg-nothing", issues); len(got) != 0 {
		t.Fatalf("ChildrenOf(mg-nothing) = %v, want no children", idsOf(got))
	}
	if got := ChildrenOf("", issues); len(got) != 0 {
		t.Fatalf("ChildrenOf(\"\") = %v, want no children", idsOf(got))
	}
}

func TestDescendantsTransitive(t *testing.T) {
	issues := []Issue{
		{ID: "mard-nob", IssueType: TypeEpic},
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
		{ID: "mard-nob.1.1", Dependencies: parentEdge("mard-nob.1.1", "mard-nob.1")},
		{ID: "mard-nob.2", Dependencies: parentEdge("mard-nob.2", "mard-nob")},
		{ID: "mg-unrelated"},
	}

	got := idsOf(Descendants("mard-nob", BuildIssueMap(issues)))
	want := []string{"mard-nob.1", "mard-nob.1.1", "mard-nob.2"}
	if !slices.Equal(got, want) {
		t.Fatalf("Descendants(mard-nob) = %v, want %v", got, want)
	}
}

func TestDescendantsExcludesRoot(t *testing.T) {
	// The root has its own parent, but a descendant walk never yields the root.
	issues := []Issue{
		{ID: "mg-top"},
		{ID: "mard-nob", Dependencies: parentEdge("mard-nob", "mg-top")},
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
	}

	got := idsOf(Descendants("mard-nob", BuildIssueMap(issues)))
	if !slices.Equal(got, []string{"mard-nob.1"}) {
		t.Fatalf("Descendants(mard-nob) = %v, want [mard-nob.1]", got)
	}
}

func TestDescendantsCycleSafe(t *testing.T) {
	issues := []Issue{
		{ID: "mg-a", Dependencies: parentEdge("mg-a", "mg-b")},
		{ID: "mg-b", Dependencies: parentEdge("mg-b", "mg-a")},
	}

	got := idsOf(Descendants("mg-a", BuildIssueMap(issues)))
	if !slices.Equal(got, []string{"mg-b"}) {
		t.Fatalf("Descendants(mg-a) = %v, want [mg-b] — a cycle must terminate and never re-yield the root", got)
	}
}

func TestDescendantsSelfParent(t *testing.T) {
	issues := []Issue{
		{ID: "mg-self", Dependencies: parentEdge("mg-self", "mg-self")},
	}

	if got := Descendants("mg-self", BuildIssueMap(issues)); len(got) != 0 {
		t.Fatalf("Descendants(mg-self) = %v, want none", idsOf(got))
	}
}

func TestDescendantsMissingParentSafe(t *testing.T) {
	issues := []Issue{
		// Edge names a parent that was never loaded (filtered view, or a parent
		// in another section). The edge is authority, so this is a descendant.
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
		// Edge into a hole: the walk must terminate rather than chase it.
		{ID: "mg-x", Dependencies: parentEdge("mg-x", "mg-gone")},
	}
	m := BuildIssueMap(issues)

	if got := idsOf(Descendants("mard-nob", m)); !slices.Equal(got, []string{"mard-nob.1"}) {
		t.Fatalf("Descendants(mard-nob) = %v, want [mard-nob.1] — the edge names the parent even when that parent is not loaded", got)
	}
	if got := idsOf(Descendants("mg-gone", m)); !slices.Equal(got, []string{"mg-x"}) {
		t.Fatalf("Descendants(mg-gone) = %v, want [mg-x]", got)
	}
	if got := Descendants("mg-absent", m); len(got) != 0 {
		t.Fatalf("Descendants(mg-absent) = %v, want none — nothing names that ID as a parent", idsOf(got))
	}
}

func TestEpicAncestorIsSelfForEpic(t *testing.T) {
	issues := []Issue{
		{ID: "mard-nob", IssueType: TypeEpic},
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
	}

	if got := EpicAncestor(&issues[0], BuildIssueMap(issues)); got == nil || got.ID != "mard-nob" {
		t.Fatalf("EpicAncestor(epic) = %v, want the epic itself", got)
	}
}

func TestEpicAncestorWalksToNearestEpic(t *testing.T) {
	// A nested epic shadows the outer one: the nearest ancestor wins.
	issues := []Issue{
		{ID: "mg-outer", IssueType: TypeEpic},
		{ID: "mg-outer.1", IssueType: TypeEpic, Dependencies: parentEdge("mg-outer.1", "mg-outer")},
		{ID: "mg-outer.1.1", IssueType: TypeTask, Dependencies: parentEdge("mg-outer.1.1", "mg-outer.1")},
		{ID: "mg-outer.2", IssueType: TypeTask, Dependencies: parentEdge("mg-outer.2", "mg-outer")},
	}
	m := BuildIssueMap(issues)

	if got := EpicAncestor(m["mg-outer.1.1"], m); got == nil || got.ID != "mg-outer.1" {
		t.Fatalf("EpicAncestor(mg-outer.1.1) = %v, want mg-outer.1", got)
	}
	if got := EpicAncestor(m["mg-outer.2"], m); got == nil || got.ID != "mg-outer" {
		t.Fatalf("EpicAncestor(mg-outer.2) = %v, want mg-outer", got)
	}
}

func TestEpicAncestorNone(t *testing.T) {
	issues := []Issue{
		{ID: "mg-root", IssueType: TypeTask},
		{ID: "mg-root.1", IssueType: TypeTask, Dependencies: parentEdge("mg-root.1", "mg-root")},
		// Cycle with no epic anywhere: the walk must terminate.
		{ID: "mg-a", IssueType: TypeTask, Dependencies: parentEdge("mg-a", "mg-b")},
		{ID: "mg-b", IssueType: TypeTask, Dependencies: parentEdge("mg-b", "mg-a")},
	}
	m := BuildIssueMap(issues)

	if got := EpicAncestor(m["mg-root.1"], m); got != nil {
		t.Fatalf("EpicAncestor(mg-root.1) = %v, want nil", got)
	}
	if got := EpicAncestor(m["mg-a"], m); got != nil {
		t.Fatalf("EpicAncestor(mg-a) = %v, want nil — a cycle has no epic", got)
	}
	if got := EpicAncestor(m["mg-root.1"], BuildIssueMap(nil)); got != nil {
		t.Fatalf("EpicAncestor with an empty map = %v, want nil", got)
	}
}

func TestScopeToSubtreePreservesInputOrder(t *testing.T) {
	// Input arrives in parade order (priority-then-recency), not tree order; the
	// scoped slice must keep that order rather than re-sorting by hierarchy.
	issues := []Issue{
		{ID: "mg-outside", IssueType: TypeTask},
		{ID: "mard-nob.2", IssueType: TypeTask, Dependencies: parentEdge("mard-nob.2", "mard-nob")},
		{ID: "mard-nob", IssueType: TypeEpic},
		{ID: "mard-nob.1.1", IssueType: TypeTask, Dependencies: parentEdge("mard-nob.1.1", "mard-nob.1")},
		{ID: "mard-nob.1", IssueType: TypeTask, Dependencies: parentEdge("mard-nob.1", "mard-nob")},
	}

	got := ScopeToSubtree(issues, "mard-nob")
	want := []string{"mard-nob.2", "mard-nob", "mard-nob.1.1", "mard-nob.1"}
	if len(got) != len(want) {
		t.Fatalf("ScopeToSubtree(mard-nob) = %d issues, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("ScopeToSubtree(mard-nob)[%d] = %s, want %s (order: %v)", i, got[i].ID, want[i], want)
		}
	}
}

func TestScopeToSubtreeUnknownRootIsNil(t *testing.T) {
	issues := []Issue{
		{ID: "mard-nob.1", Dependencies: parentEdge("mard-nob.1", "mard-nob")},
	}

	if got := ScopeToSubtree(issues, "mg-absent"); got != nil {
		t.Fatalf("ScopeToSubtree(mg-absent) = %v, want nil — a stale scope must empty the view", got)
	}
}

func TestScopeToSubtreeLeafAndCycle(t *testing.T) {
	issues := []Issue{
		{ID: "mg-a", Dependencies: parentEdge("mg-a", "mg-b")},
		{ID: "mg-b", Dependencies: parentEdge("mg-b", "mg-a")},
		{ID: "mg-leaf"},
	}

	if got := ScopeToSubtree(issues, "mg-leaf"); len(got) != 1 || got[0].ID != "mg-leaf" {
		t.Fatalf("ScopeToSubtree(mg-leaf) = %v, want just the leaf", got)
	}

	got := ScopeToSubtree(issues, "mg-a")
	if len(got) != 2 {
		t.Fatalf("ScopeToSubtree(mg-a) = %v, want both cycle members exactly once", got)
	}
	seen := map[string]bool{}
	for _, iss := range got {
		if seen[iss.ID] {
			t.Fatalf("ScopeToSubtree(mg-a) emitted %s twice", iss.ID)
		}
		seen[iss.ID] = true
	}
}

func TestRelativeDisplayID(t *testing.T) {
	tests := []struct {
		name string
		iss  Issue
		want string
	}{
		{
			name: "matching prefix compacts",
			iss:  Issue{ID: "mard-nob.7", Dependencies: parentEdge("mard-nob.7", "mard-nob")},
			want: ".7",
		},
		{
			name: "deeper matching prefix compacts to last segment",
			iss:  Issue{ID: "mard-nob.7.2", Dependencies: parentEdge("mard-nob.7.2", "mard-nob.7")},
			want: ".2",
		},
		{
			name: "reparented issue keeps its stale dotted ID",
			iss:  Issue{ID: "mard-nob.7", Dependencies: parentEdge("mard-nob.7", "mg-other")},
			want: "mard-nob.7",
		},
		{
			name: "dotted ID with no edge stays full",
			iss:  Issue{ID: "mard-nob.7"},
			want: "mard-nob.7",
		},
		{
			name: "edge-only child with no dotted prefix stays full",
			iss:  Issue{ID: "mg-900", Dependencies: parentEdge("mg-900", "mard-nob")},
			want: "mg-900",
		},
		{
			name: "root with neither prefix nor edge stays full",
			iss:  Issue{ID: "mard-nob"},
			want: "mard-nob",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RelativeDisplayID(&tc.iss); got != tc.want {
				t.Errorf("RelativeDisplayID(%s) = %q, want %q", tc.iss.ID, got, tc.want)
			}
		})
	}
}
