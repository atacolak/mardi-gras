package data

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultCommentAuthor = "operator"

// commentAuthorPath is overridden in tests.
var commentAuthorPath string

func authorFile() string {
	if commentAuthorPath != "" {
		return commentAuthorPath
	}
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		cfg = filepath.Join(os.TempDir(), "mardi-gras")
	}
	return filepath.Join(cfg, "mardi-gras", "author")
}

// CommentAuthor is who mg signs comments as.
// MG_AUTHOR (session) wins, then ~/.config/mardi-gras/author, then "operator".
func CommentAuthor() string {
	if a := strings.TrimSpace(os.Getenv("MG_AUTHOR")); a != "" {
		return a
	}
	b, err := os.ReadFile(authorFile())
	if err == nil {
		if a := strings.TrimSpace(string(b)); a != "" {
			return a
		}
	}
	return defaultCommentAuthor
}

// SaveCommentAuthor writes the name used by `mg --author`.
func SaveCommentAuthor(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	path := authorFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o644)
}
