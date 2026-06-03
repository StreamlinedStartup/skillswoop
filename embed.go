package main

import (
	"crypto/sha256"
	"encoding/hex"
	_ "embed"
	"os"
	"path/filepath"
)

// The bash engine is bundled into the binary. On first run we extract it to the
// user cache (content-hashed, so upgrades replace it) and exec it from there.
//
//go:embed engine/swoop-core
var coreScript []byte

// extractedCore writes the embedded engine to a stable cache path and returns it.
func extractedCore() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil || cache == "" {
		cache = filepath.Join(os.TempDir())
	}
	dir := filepath.Join(cache, "swoop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha256.Sum256(coreScript)
	path := filepath.Join(dir, "swoop-core-"+hex.EncodeToString(sum[:8]))
	if _, err := os.Stat(path); err == nil {
		return path, nil // already extracted
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, coreScript, 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}
