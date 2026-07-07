package forge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCorpusDir returns the path to the committed sample corpus used in unit tests.
func TestCorpusDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "testdata", "corpus"),         // src/forge
		filepath.Join("..", "..", "..", "testdata", "corpus"),   // src/forge/title, identity, ...
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "games.json")); err == nil {
			return dir
		}
	}
	t.Fatalf("test corpus not found (tried %v)", candidates)
	return ""
}
