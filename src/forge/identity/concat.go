package identity

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"vargames-name-gen/src/forge"
)

func concatIdentity(rng *rand.Rand, pool []string, existing map[string]struct{}) string {
	if len(pool) == 0 {
		return ""
	}
	if len(pool) == 1 {
		return mutateSingleIdentity(rng, pool[0], existing)
	}

	for range 20 {
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
			candidate = forge.NormalizeTitle(candidate + epithetSuffixes[rng.IntN(len(epithetSuffixes))])
		}
		if candidate != "" && !forge.RejectTitle(candidate, existing) {
			return candidate
		}
	}
	return ""
}

func mutateSingleIdentity(rng *rand.Rand, name string, existing map[string]struct{}) string {
	frag := pickFragment(rng, name)
	if frag == "" {
		return ""
	}
	candidate := forge.NormalizeTitle(frag + epithetSuffixes[rng.IntN(len(epithetSuffixes))])
	if forge.RejectTitle(candidate, existing) {
		return ""
	}
	return candidate
}

func pickFragment(rng *rand.Rand, name string) string {
	tokens := forge.TokenizeTitle(name)
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

func generateIdentity(rng *rand.Rand, pool []string, existing map[string]struct{}, strategy string) (string, error) {
	switch strategy {
	case "concat":
		if name := concatIdentity(rng, pool, existing); name != "" {
			return name, nil
		}
		return "", fmt.Errorf("concat could not produce an acceptable name")
	case "pick":
		for i := 0; i < len(pool)*2; i++ {
			candidate := pickFromPool(rng, pool)
			if candidate != "" && !forge.RejectTitle(candidate, existing) {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("pick could not produce an acceptable name")
	default:
		return "", fmt.Errorf("unknown strategy %q", strategy)
	}
}
