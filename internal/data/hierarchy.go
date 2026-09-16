package data

import "strings"

// Hierarchy in Beads is established by the `parent-child` dependency edge, never
// by dotted ID text. Both facts exist independently: `bd create --parent` writes
// the edge *and* a dotted ID, but removing the edge while keeping the ID is the
// reparenting case that made dotted-ID depth unreliable (see
// ParentRelationshipID). Every walker here therefore reads edges only, and the
// one function that does look at ID text — RelativeDisplayID — still checks the
// edge before treating a prefix as redundant.

// ChildrenOf returns the issues naming parentID through a parent-child edge.
// A dotted ID whose edge was removed or repointed is not a child, and an issue
// with an edge but no dotted prefix is one.
func ChildrenOf(parentID string, issues []Issue) []*Issue {
	if parentID == "" {
		return nil
	}
	var children []*Issue
	for i := range issues {
		if issues[i].ParentRelationshipID() == parentID {
			children = append(children, &issues[i])
		}
	}
	return children
}

// Descendants returns every transitive parent-child descendant of rootID,
// excluding the root. Cycle-safe and missing-parent-safe, like
// ParentRelationshipDepth: a descendant whose own parent is not loaded simply
// ends that branch. The root's own record need not be in issueMap — the edge
// naming rootID is the authority, so a filtered view still reports the children
// it can see.
func Descendants(rootID string, issueMap map[string]*Issue) []*Issue {
	if rootID == "" || len(issueMap) == 0 {
		return nil
	}

	children := make(map[string][]*Issue, len(issueMap))
	for _, iss := range issueMap {
		parentID := iss.ParentRelationshipID()
		if parentID == "" || parentID == iss.ID {
			continue // no edge, or a self-edge that would only re-yield itself
		}
		children[parentID] = append(children[parentID], iss)
	}

	var descendants []*Issue
	// The root counts as visited so a cycle back into it ends the walk instead
	// of yielding the root as its own descendant.
	visited := map[string]bool{rootID: true}
	queue := []string{rootID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, child := range children[id] {
			if visited[child.ID] {
				continue
			}
			visited[child.ID] = true
			descendants = append(descendants, child)
			queue = append(queue, child.ID)
		}
	}
	return descendants
}

// EpicAncestor returns i when i is an epic, else its nearest ancestor of type
// epic, else nil. Walks parent-child edges only.
func EpicAncestor(i *Issue, issueMap map[string]*Issue) *Issue {
	if i == nil {
		return nil
	}
	seen := map[string]bool{i.ID: true}
	for current := i; current != nil; {
		if current.IssueType == TypeEpic {
			return current
		}
		parentID := current.ParentRelationshipID()
		parent, ok := issueMap[parentID]
		if parentID == "" || !ok || seen[parentID] {
			return nil // missing parent or cycle: no epic is reachable
		}
		seen[parentID] = true
		current = parent
	}
	return nil
}

// ScopeToSubtree returns rootID plus its descendants, preserving input order.
// An unknown rootID yields nil, so a stale scope empties the view instead of
// silently showing everything. A cycle among the descendants is emitted once
// per issue, never dropped.
func ScopeToSubtree(issues []Issue, rootID string) []Issue {
	if rootID == "" {
		return nil
	}
	issueMap := BuildIssueMap(issues)
	if _, ok := issueMap[rootID]; !ok {
		return nil
	}

	inSubtree := map[string]bool{rootID: true}
	for _, descendant := range Descendants(rootID, issueMap) {
		inSubtree[descendant.ID] = true
	}

	var scoped []Issue
	for i := range issues {
		if inSubtree[issues[i].ID] {
			scoped = append(scoped, issues[i])
		}
	}
	return scoped
}

// RelativeDisplayID compacts an ID whose dotted prefix is redundant because
// its parent-child parent is that prefix: "mard-nob.7" under "mard-nob" gives
// ".7". Returns the full ID when the prefix is not redundant — a reparented
// issue keeps its old dotted ID, and an edge-only child has no dotted prefix.
// It needs no issueMap: the comparison is between this issue's own dotted
// prefix and its own parent-child edge.
func RelativeDisplayID(i *Issue) string {
	if i == nil {
		return ""
	}
	prefix := i.ParentID()
	if prefix == "" || prefix != i.ParentRelationshipID() {
		return i.ID // nothing to strip, or the prefix is stale paint
	}
	return strings.TrimPrefix(i.ID, prefix)
}
