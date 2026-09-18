package data

import (
	"sort"
	"strings"
)

// Mention is a loaded issue ID found in text. Start/End are byte offsets
// into the scanned string (ANSI-stripped text).
type Mention struct {
	ID    string
	Start int
	End   int
}

// FindLoadedMentions returns every loaded issue ID that appears in text at
// a token boundary, longest match first so mard-wte.7 wins over mard-wte.
// IDs that are not in known are ignored — a dead link is worse than plain text.
func FindLoadedMentions(text string, known map[string]*Issue) []Mention {
	if text == "" || len(known) == 0 {
		return nil
	}
	ids := make([]string, 0, len(known))
	for id := range known {
		if id != "" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		if len(ids[i]) != len(ids[j]) {
			return len(ids[i]) > len(ids[j])
		}
		return ids[i] < ids[j]
	})

	used := make([]bool, len(text))
	var out []Mention
	for _, id := range ids {
		start := 0
		for {
			rel := strings.Index(text[start:], id)
			if rel < 0 {
				break
			}
			i := start + rel
			end := i + len(id)
			if !rangeUsed(used, i, end) && isMentionBoundary(text, i, end) {
				for k := i; k < end; k++ {
					used[k] = true
				}
				out = append(out, Mention{ID: id, Start: i, End: end})
			}
			start = i + 1
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

func rangeUsed(used []bool, start, end int) bool {
	for i := start; i < end && i < len(used); i++ {
		if used[i] {
			return true
		}
	}
	return false
}

func isMentionBoundary(s string, start, end int) bool {
	if start > 0 && isMentionChar(s[start-1]) {
		return false
	}
	if end < len(s) && isMentionChar(s[end]) {
		return false
	}
	return true
}

func isMentionChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-' || b == '.'
}
