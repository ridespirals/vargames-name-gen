package forge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDir_SampleCorpus(t *testing.T) {
	dir := TestCorpusDir(t)

	corpus, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if corpus.Dir != dir {
		t.Fatalf("Dir = %q, want %q", corpus.Dir, dir)
	}
	if len(corpus.Games) != 10 {
		t.Fatalf("got %d games, want 10", len(corpus.Games))
	}
	if len(corpus.Characters) != 10 {
		t.Fatalf("got %d characters, want 10", len(corpus.Characters))
	}
	if len(corpus.Genres) != 5 {
		t.Fatalf("got %d genres, want 5", len(corpus.Genres))
	}
	if len(corpus.Platforms) != 3 {
		t.Fatalf("got %d platforms, want 3", len(corpus.Platforms))
	}
	if len(corpus.AlternativeNames) != 5 {
		t.Fatalf("got %d alternative names, want 5", len(corpus.AlternativeNames))
	}
	if len(corpus.Collections) != 3 {
		t.Fatalf("got %d collections, want 3", len(corpus.Collections))
	}
	if len(corpus.Companies) != 3 {
		t.Fatalf("got %d companies, want 3", len(corpus.Companies))
	}
}

func TestLoadFromDir_RequiresGames(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error when games.json is missing")
	}
}

func TestLoadFromDir_OptionalFilesMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "games.json"), []byte(`[{"id":1,"name":"Test Game"}]`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	corpus, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if len(corpus.Games) != 1 {
		t.Fatalf("got %d games, want 1", len(corpus.Games))
	}
	if len(corpus.Characters) != 0 || len(corpus.Genres) != 0 {
		t.Fatalf("expected empty optional entities, got characters=%d genres=%d", len(corpus.Characters), len(corpus.Genres))
	}
}

func TestLoadFromDir_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "games.json"), []byte(`not-json`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func benchmarkCorpusDir(b *testing.B) string {
	b.Helper()
	candidates := []string{
		filepath.Join("..", "..", "testdata", "corpus"),
		filepath.Join("..", "..", "..", "testdata", "corpus"),
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "games.json")); err == nil {
			return dir
		}
	}
	b.Fatalf("test corpus not found (tried %v)", candidates)
	return ""
}

func BenchmarkLoadFromDir_SampleCorpus(b *testing.B) {
	dir := benchmarkCorpusDir(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := LoadFromDir(dir); err != nil {
			b.Fatalf("LoadFromDir: %v", err)
		}
	}
}

// BenchmarkLoadFromDir_DataDir measures production corpus load time when data/ exists.
// Skipped in CI; run locally: go test -bench=BenchmarkLoadFromDir_DataDir -benchtime=3x ./src/forge/...
func BenchmarkLoadFromDir_DataDir(b *testing.B) {
	const dir = "data"
	if _, err := os.Stat(filepath.Join(dir, "games.json")); err != nil {
		b.Skip("data/games.json not present; fetch or copy corpus to benchmark")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := LoadFromDir(dir); err != nil {
			b.Fatalf("LoadFromDir: %v", err)
		}
	}
}
