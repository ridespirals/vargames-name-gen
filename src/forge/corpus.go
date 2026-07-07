package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Game holds minimal IGDB game fields used for title generation and weighting.
type Game struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Genres    []int  `json:"genres"`
	Platforms []int  `json:"platforms"`
}

// Character holds minimal IGDB character fields.
type Character struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Games []int  `json:"games"`
}

// AlternativeName holds a regional or alternate title linked to a game.
type AlternativeName struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Game int    `json:"game"`
}

// Collection holds a minimal IGDB game collection / series record.
type Collection struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Company holds a minimal IGDB company record.
type Company struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Genre holds a minimal IGDB genre record.
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Platform holds a minimal IGDB platform record.
type Platform struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Corpus is the in-memory snapshot of IGDB entities loaded from disk.
type Corpus struct {
	Dir              string
	Games            []Game
	Characters       []Character
	AlternativeNames []AlternativeName
	Collections      []Collection
	Companies        []Company
	Genres           []Genre
	Platforms        []Platform
}

// LoadFromDir reads IGDB JSON arrays from dir and returns a Corpus.
// games.json is required; other entity files are optional.
func LoadFromDir(dir string) (*Corpus, error) {
	games, err := loadRequiredEntity[Game](dir, "games")
	if err != nil {
		return nil, err
	}

	characters, err := loadOptionalEntity[Character](dir, "characters")
	if err != nil {
		return nil, err
	}
	altNames, err := loadOptionalEntity[AlternativeName](dir, "alternative_names")
	if err != nil {
		return nil, err
	}
	collections, err := loadOptionalEntity[Collection](dir, "collections")
	if err != nil {
		return nil, err
	}
	companies, err := loadOptionalEntity[Company](dir, "companies")
	if err != nil {
		return nil, err
	}
	genres, err := loadOptionalEntity[Genre](dir, "genres")
	if err != nil {
		return nil, err
	}
	platforms, err := loadOptionalEntity[Platform](dir, "platforms")
	if err != nil {
		return nil, err
	}

	return &Corpus{
		Dir:              dir,
		Games:            games,
		Characters:       characters,
		AlternativeNames: altNames,
		Collections:      collections,
		Companies:        companies,
		Genres:           genres,
		Platforms:        platforms,
	}, nil
}

func loadRequiredEntity[T any](dir, entity string) ([]T, error) {
	path := entityJSONPath(dir, entity)
	items, err := loadEntityJSON[T](path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("required corpus file missing: %s", path)
		}
		return nil, fmt.Errorf("load %s: %w", entity, err)
	}
	return items, nil
}

func loadOptionalEntity[T any](dir, entity string) ([]T, error) {
	path := entityJSONPath(dir, entity)
	items, err := loadEntityJSON[T](path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load %s: %w", entity, err)
	}
	return items, nil
}

func entityJSONPath(dir, entity string) string {
	return filepath.Join(dir, entity+".json")
}

func loadEntityJSON[T any](path string) ([]T, error) {
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".partial.json") {
		return nil, fmt.Errorf("skip partial corpus file: %s", path)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []T
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return items, nil
}
