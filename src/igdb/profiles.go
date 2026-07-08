package igdb

import (
	"fmt"
	"strings"
)

// Named fetch profiles for Apicalypse query prefixes.
const (
	ProfileFull     = "full"
	ProfileMinimal  = "minimal"
	ProfileChecksum = "checksum"
)

// DefaultProfileName is the profile used when none is specified.
const DefaultProfileName = ProfileMinimal

// Profile holds a named Apicalypse query prefix (semicolon-terminated).
type Profile struct {
	Name        string
	QueryPrefix string
}

var minimalPrefixes = map[Entity]string{
	EntityGames:            "fields id,name,genres,platforms,checksum;",
	EntityCharacters:       "fields id,name,games,checksum;",
	EntityGenres:           "fields id,name,slug,checksum;",
	EntityPlatforms:        "fields id,name,abbreviation,checksum;",
	EntityCollections:      "fields id,name,checksum;",
	EntityCompanies:        "fields id,name,checksum;",
	EntityAlternativeNames: "fields id,name,game,comment,checksum;",
}

// ProfileFor returns the fetch profile for an entity by name (full, minimal, checksum).
func ProfileFor(entity Entity, name string) (Profile, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = DefaultProfileName
	}
	switch name {
	case ProfileFull:
		return Profile{Name: ProfileFull, QueryPrefix: DefaultQueryPrefix}, nil
	case ProfileMinimal:
		prefix, ok := minimalPrefixes[entity]
		if !ok {
			return Profile{}, fmt.Errorf("no minimal profile for entity %q", entity)
		}
		return Profile{Name: ProfileMinimal, QueryPrefix: prefix}, nil
	case ProfileChecksum:
		return Profile{Name: ProfileChecksum, QueryPrefix: "fields id,checksum;"}, nil
	default:
		return Profile{}, fmt.Errorf("unknown fetch profile %q (valid: full, minimal, checksum)", name)
	}
}

// DefaultProfile returns the default (minimal) profile for an entity.
func DefaultProfile(entity Entity) (Profile, error) {
	return ProfileFor(entity, DefaultProfileName)
}

// ValidProfileName reports whether name is a known profile name.
func ValidProfileName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ProfileFull, ProfileMinimal, ProfileChecksum:
		return true
	default:
		return false
	}
}
