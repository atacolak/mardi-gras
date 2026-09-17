package data

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// LoadIssues reads and parses a Beads JSONL file.
// Malformed lines are skipped rather than aborting the entire load.
// Returns the parsed issues, the count of skipped lines, and any scanner error.
func LoadIssues(path string) ([]Issue, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open issues file: %w", err)
	}
	defer f.Close()

	var issues []Issue
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	skipped := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var issue Issue
		if err := json.Unmarshal(line, &issue); err != nil {
			skipped++
			continue
		}
		issues = append(issues, issue)
	}
	if err := scanner.Err(); err != nil {
		return nil, skipped, fmt.Errorf("scan issues file: %w", err)
	}

	SortIssues(issues)
	return issues, skipped, nil
}

// SortIssues sorts by priority (ascending), then ID (ascending).
func SortIssues(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.ID < b.ID
	})
}
