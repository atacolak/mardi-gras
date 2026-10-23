package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// seenStoreDir overrides the cache directory in tests.
var seenStoreDir string

func seenFile(projectDir string) string {
	dir := seenStoreDir
	if dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil || cache == "" {
			cache = os.TempDir()
		}
		dir = filepath.Join(cache, "mardi-gras", "seen")
	}
	key := projectDir
	if key == "" {
		key, _ = os.Getwd()
	}
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".json")
}

func loadSeenComments(projectDir string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	b, err := os.ReadFile(seenFile(projectDir))
	if err != nil || len(b) == 0 {
		return out
	}
	if json.Unmarshal(b, &out) != nil || out == nil {
		return map[string]map[string]bool{}
	}
	return out
}

func saveSeenComments(projectDir string, seen map[string]map[string]bool) {
	if seen == nil {
		return
	}
	path := seenFile(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	b, err := json.Marshal(seen)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}
