package role

import (
	"os"
	"strconv"
)

// SizeFactorEnv is the variable the engine gives a role's scripts when
// the issue it runs on has a size tier (ADR-0064, §3): what the tier
// multiplies every size target by. A step that measures a prompt against
// the context budget itself multiplies the budget by it (SizeFactor).
const SizeFactorEnv = "WORKLINE_SIZE_FACTOR"

// TieredSettings are the settings a size tier multiplies: what a run may
// spend, and how much code or change it is given at once.
var TieredSettings = []string{"ai-max-tokens", "code-lines-max", "code-bytes-max", "diff-lines-max"}

// SizeFactor is the factor the engine gave this process's run: 1 when
// none, or one that does not read.
func SizeFactor() int {
	n, err := strconv.Atoi(os.Getenv(SizeFactorEnv))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// Tiered is settings with each of TieredSettings set above 0 multiplied
// by factor; settings itself when factor is 1 or less.
func Tiered(settings map[string]any, factor int) map[string]any {
	if factor <= 1 {
		return settings
	}
	out := make(map[string]any, len(settings))
	for k, v := range settings {
		out[k] = v
	}
	for _, k := range TieredSettings {
		switch v := settings[k].(type) {
		case int:
			if v > 0 {
				out[k] = v * factor
			}
		case float64:
			if v > 0 {
				out[k] = v * float64(factor)
			}
		}
	}
	return out
}
