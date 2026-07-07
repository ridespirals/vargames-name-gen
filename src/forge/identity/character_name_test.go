package identity

import (
	"testing"

	"vargames-name-gen/src/forge"
)

func TestFilterCharacterPool_genre(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	fighting := 4
	gameGenres := gameGenreIndex(corpus.Games)
	filtered := filterCharacterPool(corpus.Characters, &fighting, gameGenres)
	if len(filtered) != 1 {
		t.Fatalf("genre 4: got %d characters, want 1", len(filtered))
	}
	if filtered[0].Name != "Ryu" {
		t.Fatalf("got %q, want Ryu", filtered[0].Name)
	}
}

func TestCharacterNamePool_fallbackWhenNoMatches(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)

	missing := 99999
	pool := gen.characterNamePool(Options{GenreID: &missing})
	if len(pool) != len(corpus.Characters) {
		t.Fatalf("expected fallback to full pool (%d), got %d", len(corpus.Characters), len(pool))
	}
}

func TestCharacterName_deterministic(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)

	a, err := gen.CharacterName(Options{Seed: 77})
	if err != nil {
		t.Fatalf("CharacterName: %v", err)
	}
	b, err := gen.CharacterName(Options{Seed: 77})
	if err != nil {
		t.Fatalf("CharacterName: %v", err)
	}
	if a != b {
		t.Fatalf("same seed produced different names: %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("expected non-empty name")
	}
}

func TestCharacterName_notExactCorpusMatch(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)
	existing := gen.sourceNameSet()

	for seed := int64(1); seed <= 20; seed++ {
		name, err := gen.CharacterName(Options{Seed: seed})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if forge.RejectTitle(name, existing) {
			t.Fatalf("seed %d produced corpus duplicate or rejected name %q", seed, name)
		}
	}
}

func TestCharacterName_genreFilterUsesMatchingPool(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)

	fighting := 4
	name, err := gen.CharacterName(Options{GenreID: &fighting, Seed: 42})
	if err != nil {
		t.Fatalf("CharacterName: %v", err)
	}

	allowed := map[string]bool{
		"Ryu": true,
		"the": true, "Bold": true, "Wise": true, "of": true, "Astora": true,
		"First": true, "Jr.": true,
	}
	for _, part := range forge.TokenizeTitle(name) {
		if !allowed[part] {
			t.Fatalf("genre-filtered name %q contains token %q outside fighting pool", name, part)
		}
	}
}

func TestCharacterName_nilGenerator(t *testing.T) {
	var gen *Generator
	if _, err := gen.CharacterName(Options{}); err == nil {
		t.Fatal("expected error for nil generator")
	}
}

func TestCharacterName_emptyCorpus(t *testing.T) {
	gen := New(&forge.Corpus{Games: []forge.Game{{ID: 1, Name: "Placeholder"}}})
	if _, err := gen.CharacterName(Options{}); err == nil {
		t.Fatal("expected error for empty character corpus")
	}
}

func TestCharacterName_unknownStrategy(t *testing.T) {
	dir := forge.TestCorpusDir(t)
	corpus, err := forge.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	gen := New(corpus)
	if _, err := gen.CharacterName(Options{Strategy: "markov", Seed: 1}); err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}
