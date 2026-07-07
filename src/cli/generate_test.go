package cli

import (
	"os"
	"path/filepath"
	"testing"

	"vargames-name-gen/src/forge"
)

func TestGenerateIdentities_Deterministic(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	names1, err := GenerateIdentities(IdentityConfig{DataDir: dir, Seed: 42, Count: 3})
	if err != nil {
		t.Fatalf("GenerateIdentities: %v", err)
	}
	names2, err := GenerateIdentities(IdentityConfig{DataDir: dir, Seed: 42, Count: 3})
	if err != nil {
		t.Fatalf("GenerateIdentities second run: %v", err)
	}
	if len(names1) != 3 {
		t.Fatalf("got %d names, want 3", len(names1))
	}
	for i := range names1 {
		if names1[i] != names2[i] {
			t.Fatalf("seed 42 not deterministic: %q vs %q", names1[i], names2[i])
		}
	}
}

func TestGenerateTitles_Deterministic(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	names1, err := GenerateTitles(TitleConfig{DataDir: dir, Seed: 42, Count: 3})
	if err != nil {
		t.Fatalf("GenerateTitles: %v", err)
	}
	names2, err := GenerateTitles(TitleConfig{DataDir: dir, Seed: 42, Count: 3})
	if err != nil {
		t.Fatalf("GenerateTitles second run: %v", err)
	}
	if len(names1) != 3 {
		t.Fatalf("got %d names, want 3", len(names1))
	}
	for i := range names1 {
		if names1[i] != names2[i] {
			t.Fatalf("seed 42 not deterministic: %q vs %q", names1[i], names2[i])
		}
	}
}

func TestDefaultDataDir_PrefersSampleCorpus(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	if _, err := os.Stat("testdata/corpus/games.json"); err != nil {
		t.Skip("testdata/corpus not present")
	}
	t.Setenv(EnvDataDir, "")
	if got := DefaultDataDir(); got != "testdata/corpus" {
		t.Fatalf("DefaultDataDir() = %q, want testdata/corpus", got)
	}
}

func TestRunGenerate_UnknownSubcommand(t *testing.T) {
	err := RunGenerate([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error")
	}
}
