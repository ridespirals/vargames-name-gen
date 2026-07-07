package identity

import (
	"errors"
	"fmt"

	"vargames-name-gen/src/forge"
)

// CharacterName generates a character-style name from the corpus.
func (g *Generator) CharacterName(opts Options) (string, error) {
	if g == nil || g.corpus == nil {
		return "", errors.New("identity: nil generator")
	}
	if len(g.corpus.Characters) == 0 {
		return "", errors.New("identity: no characters in corpus")
	}

	pool := g.characterNamePool(opts)
	existing := g.sourceNameSet()
	rng := forge.NewRand(opts.Seed)
	strategy := strategyName(opts)

	const maxAttempts = 32
	var lastErr error
	for range maxAttempts {
		name, err := generateIdentity(rng, pool, existing, strategy)
		if err != nil {
			lastErr = err
			continue
		}
		if name != "" {
			return name, nil
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("identity: %w", lastErr)
	}
	return "", errors.New("identity: could not generate acceptable character name")
}

func (g *Generator) characterNamePool(opts Options) []string {
	gameGenres := gameGenreIndex(g.corpus.Games)
	filtered := filterCharacterPool(g.corpus.Characters, opts.GenreID, gameGenres)
	pool := characterNames(filtered)
	if len(pool) > 0 {
		return pool
	}
	return characterNames(g.corpus.Characters)
}

func (g *Generator) sourceNameSet() map[string]struct{} {
	names := make([]string, 0, len(g.corpus.Characters)+len(g.corpus.Companies))
	for _, character := range g.corpus.Characters {
		names = append(names, character.Name)
	}
	for _, company := range g.corpus.Companies {
		names = append(names, company.Name)
	}
	return forge.SourceTitleSet(names...)
}
