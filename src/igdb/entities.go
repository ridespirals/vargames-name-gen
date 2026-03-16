package igdb

// Entity identifies an IGDB API entity (endpoint path segment).
type Entity string

const (
	EntityGames            Entity = "games"
	EntityCharacters       Entity = "characters"
	EntityGenres           Entity = "genres"
	EntityPlatforms        Entity = "platforms"
	EntityCollections      Entity = "collections"
	EntityCompanies        Entity = "companies"
	EntityAlternativeNames Entity = "alternative_names"
)

// AllEntities returns every entity type supported by the fetcher (matches Bruno requests).
func AllEntities() []Entity {
	return []Entity{
		EntityGames,
		EntityCharacters,
		EntityGenres,
		EntityPlatforms,
		EntityCollections,
		EntityCompanies,
		EntityAlternativeNames,
	}
}

// ValidEntity returns true if s is a known entity name (e.g. "games", "alternative_names").
func ValidEntity(s string) bool {
	for _, e := range AllEntities() {
		if string(e) == s {
			return true
		}
	}
	return false
}
