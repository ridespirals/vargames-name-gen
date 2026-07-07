package title

import (
	"testing"

	"vargames-name-gen/src/forge"
)

func TestFilterGamePool_genre(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	fighting := 4
	filtered := filterGamePool(corpus.Games, &fighting)
	if len(filtered) != 1 {
		t.Fatalf("genre 4: got %d games, want 1", len(filtered))
	}
	if filtered[0].Name != "Street Fighter II" {
		t.Fatalf("got %q, want Street Fighter II", filtered[0].Name)
	}
}

func TestGameTitlePool_fallbackWhenNoMatches(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)

	missing := 99999
	pool := gen.gameTitlePool(Options{GenreID: &missing})
	if len(pool) != len(corpus.Games) {
		t.Fatalf("expected fallback to full pool (%d), got %d", len(corpus.Games), len(pool))
	}
}

func TestGameTitle_genreFilterUsesMatchingPool(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)

	fighting := 4
	title, err := gen.GameTitle(Options{GenreID: &fighting, Seed: 42})
	if err != nil {
		t.Fatalf("GameTitle: %v", err)
	}

	allowed := map[string]bool{
		"Street": true, "Fighter": true, "II": true,
		"Reborn": true, "Remastered": true, "Redux": true,
		"III": true, "HD": true, "DX": true, "Unlimited": true,
	}
	for _, part := range forge.TokenizeTitle(title) {
		if !allowed[part] {
			t.Fatalf("genre-filtered title %q contains token %q outside fighting pool", title, part)
		}
	}
}
