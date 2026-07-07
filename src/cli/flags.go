package cli

import (
	"os"
	"path/filepath"
)

// EnvDataDir is the environment variable for the IGDB corpus directory.
const EnvDataDir = "VARGAMES_DATA_DIR"

// DefaultDataDir returns the corpus directory to use when -data-dir is unset.
// Order: VARGAMES_DATA_DIR, then testdata/corpus if games.json exists, else data/.
func DefaultDataDir() string {
	if v := os.Getenv(EnvDataDir); v != "" {
		return v
	}
	for _, dir := range []string{"testdata/corpus", "data"} {
		if _, err := os.Stat(filepath.Join(dir, "games.json")); err == nil {
			return dir
		}
	}
	return "data"
}

// ResolveDataDir returns dir when non-empty, otherwise DefaultDataDir().
func ResolveDataDir(dir string) string {
	if dir != "" {
		return dir
	}
	return DefaultDataDir()
}
