package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/matt-wright86/mardi-gras/internal/views"
)

// boardPrefs is the per-project cockpit file at <project>/.beads/mg.json.
// It is local to the beads graph (gitignored with .beads/) so a reopen
// resumes the last S/s sort for that project only.
type boardPrefs struct {
	EpicSort string `json:"epic_sort"`
	BeadSort string `json:"bead_sort"`
}

func boardPrefsPath(projectDir string) string {
	if projectDir == "" {
		return ""
	}
	return filepath.Join(projectDir, ".beads", "mg.json")
}

func loadBoardPrefs(projectDir string) (epic, bead views.SortMode) {
	path := boardPrefsPath(projectDir)
	if path == "" {
		return views.SortAttention, views.SortAttention
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return views.SortAttention, views.SortAttention
	}
	var p boardPrefs
	if json.Unmarshal(raw, &p) != nil {
		return views.SortAttention, views.SortAttention
	}
	return views.ParseSortMode(p.EpicSort), views.ParseSortMode(p.BeadSort)
}

func saveBoardPrefs(projectDir string, epic, bead views.SortMode) {
	path := boardPrefsPath(projectDir)
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	raw, err := json.MarshalIndent(boardPrefs{
		EpicSort: epic.Label(),
		BeadSort: bead.Label(),
	}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, append(raw, '\n'), 0o644)
}
