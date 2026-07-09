package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestGenerateTitles_GenreFilter(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	fighting := 4
	names, err := GenerateTitles(TitleConfig{DataDir: dir, Seed: 42, Count: 1, GenreID: &fighting})
	if err != nil {
		t.Fatalf("GenerateTitles: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("got %d names, want 1", len(names))
	}
	allowed := map[string]bool{
		"Street": true, "Fighter": true, "II": true,
		"Reborn": true, "Remastered": true, "Redux": true,
		"III": true, "HD": true, "DX": true, "Unlimited": true,
	}
	for _, part := range forge.TokenizeTitle(names[0]) {
		if !allowed[part] {
			t.Fatalf("genre-filtered title %q contains token %q outside fighting pool", names[0], part)
		}
	}
}

func TestRunGenerate_TitleGenreFlag(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	err := RunGenerate([]string{"title", "-data-dir=" + dir, "-seed=42", "-count=1", "-genre=4"})
	w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}

	out, _ := io.ReadAll(r)
	got := strings.TrimSpace(string(out))
	if got == "" {
		t.Fatal("expected generated title on stdout")
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
