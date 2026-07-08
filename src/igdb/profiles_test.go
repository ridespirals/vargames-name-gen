package igdb

import "testing"

func TestProfileFor_minimalGames(t *testing.T) {
	p, err := ProfileFor(EntityGames, ProfileMinimal)
	if err != nil {
		t.Fatalf("ProfileFor: %v", err)
	}
	if p.QueryPrefix != "fields id,name,genres,platforms,checksum;" {
		t.Fatalf("got %q", p.QueryPrefix)
	}
}

func TestProfileFor_checksum(t *testing.T) {
	p, err := ProfileFor(EntityGenres, ProfileChecksum)
	if err != nil {
		t.Fatalf("ProfileFor: %v", err)
	}
	if p.QueryPrefix != "fields id,checksum;" {
		t.Fatalf("got %q", p.QueryPrefix)
	}
}

func TestProfileFor_full(t *testing.T) {
	p, err := ProfileFor(EntityPlatforms, ProfileFull)
	if err != nil {
		t.Fatalf("ProfileFor: %v", err)
	}
	if p.QueryPrefix != DefaultQueryPrefix {
		t.Fatalf("got %q", p.QueryPrefix)
	}
}

func TestProfileFor_unknown(t *testing.T) {
	_, err := ProfileFor(EntityGames, "bogus")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultProfile(t *testing.T) {
	p, err := DefaultProfile(EntityCharacters)
	if err != nil {
		t.Fatalf("DefaultProfile: %v", err)
	}
	if p.Name != ProfileMinimal {
		t.Fatalf("got profile %q", p.Name)
	}
}

func TestQueryPrefixForEntity_usesDefaultProfile(t *testing.T) {
	got := QueryPrefixForEntity(EntityGames)
	want := "fields id,name,genres,platforms,checksum;"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
