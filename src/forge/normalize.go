package forge

import (
	"math/rand/v2"
	"strings"
	"unicode"
)

const (
	// MinTitleLen is the minimum acceptable generated title length.
	MinTitleLen = 3
	// MaxTitleLen is the maximum acceptable generated title length.
	MaxTitleLen = 48
)

// NewRand returns a PRNG. Seed 0 selects a non-deterministic source.
func NewRand(seed int64) *rand.Rand {
	if seed == 0 {
		return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed)^uint64(0x9e3779b97f4a7c15)))
}

// NormalizeTitle trims and collapses internal whitespace in a title string.
func NormalizeTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

// TokenizeTitle splits a title into fragments on spaces, colons, and hyphens.
func TokenizeTitle(s string) []string {
	s = NormalizeTitle(s)
	if s == "" {
		return nil
	}
	replacer := strings.NewReplacer(":", " ", "-", " ")
	parts := strings.Fields(replacer.Replace(s))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) >= 2 {
			out = append(out, p)
		}
	}
	return out
}

// RejectTitle reports whether a candidate should be discarded.
func RejectTitle(candidate string, existing map[string]struct{}) bool {
	n := NormalizeTitle(candidate)
	if n == "" {
		return true
	}
	if len(n) < MinTitleLen || len(n) > MaxTitleLen {
		return true
	}
	if _, ok := existing[strings.ToLower(n)]; ok {
		return true
	}
	if isAllCapsNoise(n) {
		return true
	}
	return false
}

func isAllCapsNoise(s string) bool {
	letters := 0
	upper := 0
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.IsUpper(r) {
			upper++
		}
	}
	return letters > 4 && upper == letters
}

// SourceTitleSet builds a lowercase lookup of titles that must not be emitted verbatim.
func SourceTitleSet(titles ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(titles))
	for _, t := range titles {
		if n := NormalizeTitle(t); n != "" {
			set[strings.ToLower(n)] = struct{}{}
		}
	}
	return set
}
