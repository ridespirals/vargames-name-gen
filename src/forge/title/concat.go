package title

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"vargames-name-gen/src/forge"
)

func concatTitle(rng *rand.Rand, pool []string, existing map[string]struct{}) string {
	if len(pool) == 0 {
		return ""
	}
	if len(pool) == 1 {
		return mutateSingleTitle(rng, pool[0], existing)
	}

	for attempt := 0; attempt < 20; attempt++ {
		a := pool[rng.IntN(len(pool))]
		b := pool[rng.IntN(len(pool))]
		if strings.EqualFold(a, b) {
			continue
		}

		left := pickFragment(rng, a)
		right := pickFragment(rng, b)
		if left == "" || right == "" {
			continue
		}
		if strings.EqualFold(left, right) {
			continue
		}

		candidate := forge.NormalizeTitle(left + " " + right)
		if rng.IntN(3) == 0 {
			candidate = forge.NormalizeTitle(candidate + subtitleSuffixes[rng.IntN(len(subtitleSuffixes))])
		}
		if candidate != "" && !forge.RejectTitle(candidate, existing) {
			return candidate
		}
	}
	return ""
}

func mutateSingleTitle(rng *rand.Rand, title string, existing map[string]struct{}) string {
	frag := pickFragment(rng, title)
	if frag == "" {
		return ""
	}
	candidate := forge.NormalizeTitle(frag + subtitleSuffixes[rng.IntN(len(subtitleSuffixes))])
	if forge.RejectTitle(candidate, existing) {
		return ""
	}
	return candidate
}

func pickFragment(rng *rand.Rand, title string) string {
	tokens := forge.TokenizeTitle(title)
	if len(tokens) == 0 {
		return ""
	}
	return tokens[rng.IntN(len(tokens))]
}

func pickFromPool(rng *rand.Rand, pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	return pool[rng.IntN(len(pool))]
}

func strategyName(opts Options) string {
	if opts.Strategy == "" {
		return "concat"
	}
	return opts.Strategy
}

func generateTitle(rng *rand.Rand, pool []string, existing map[string]struct{}, strategy string) (string, error) {
	switch strategy {
	case "concat":
		if title := concatTitle(rng, pool, existing); title != "" {
			return title, nil
		}
		return "", fmt.Errorf("concat could not produce an acceptable title")
	case "pick":
		for i := 0; i < len(pool)*2; i++ {
			candidate := pickFromPool(rng, pool)
			if candidate != "" && !forge.RejectTitle(candidate, existing) {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("pick could not produce an acceptable title")
	default:
		return "", fmt.Errorf("unknown strategy %q", strategy)
	}
}
