package backlog

// An issue's size tier (ADR-0064, §3): a forge label that multiplies
// every size target of the runs on it — the context budget, the token
// cap, the lines and bytes of code given, the lines of a change read at
// once. `large` (×2) the product owner grants alone, by its `size` act,
// when the need is atomic; `x-large` (×4) is a person's answer only. The
// tier only goes up; a person takes the label off in one click.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// The size tiers, as the act and the line's input name them.
const (
	TierLarge  = "large"
	TierXLarge = "x-large"
)

// The tiers' labels, and the colour and description each is created with.
const (
	LabelSizeLarge  = "workline:size/large"
	LabelSizeXLarge = "workline:size/x-large"
	// SizeColor is the tiers' labels' colour.
	SizeColor = "c5def5"
)

// SizeTier is the tier an issue's labels give it, the top one when both
// are on: "" at the normal size.
func SizeTier(labels []string) string {
	has := func(l string) bool {
		return slices.ContainsFunc(labels, func(x string) bool { return strings.EqualFold(strings.TrimSpace(x), l) })
	}
	switch {
	case has(LabelSizeXLarge):
		return TierXLarge
	case has(LabelSizeLarge):
		return TierLarge
	}
	return ""
}

// TierFactor is what a tier multiplies every size target by: 1 at the
// normal size, or for a tier not known.
func TierFactor(tier string) int {
	switch tier {
	case TierLarge:
		return 2
	case TierXLarge:
		return 4
	}
	return 1
}

// TierLabel is the label of a tier.
func TierLabel(tier string) string {
	if tier == TierXLarge {
		return LabelSizeXLarge
	}
	return LabelSizeLarge
}

// EnsureTierLabel creates a tier's label where the project has none.
func EnsureTierLabel(f forge.Backlog, tier string) error {
	return f.EnsureLabel(TierLabel(tier), SizeColor, TierSays(tier))
}

// TierSays is a tier label's description in the forge's list.
func TierSays(tier string) string {
	return fmt.Sprintf("Every size target of the runs on this issue ×%d (ADR-0064); take it off to undo", TierFactor(tier))
}

// checkSize checks a size act: before the issue's state is read (st
// nil), `large` alone — `x-large` is a person's answer —, on an open
// issue whose tier is lower; once read, that the role never granted it
// before: a person who took the label off decided.
func (p *Plan) checkSize(c *Proposal, st *State) (rule, why string) {
	is, ok := p.issues[c.Issue]
	switch {
	case !ok || c.Issue == p.Report:
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	case st != nil && st.Size != "":
		return "size-once", fmt.Sprintf("the role granted it `%s` once already: never again; a person's removal stands", TierLabel(st.Size))
	case st != nil:
		return "", ""
	}
	c.Tier = strings.ToLower(strings.TrimSpace(c.Tier))
	switch {
	case c.Tier == TierXLarge:
		return "size-person", "`x-large` is a person's answer, never the role's: grant `large`, or leave the issue blocked for a person"
	case c.Tier != TierLarge:
		return "size-unknown", fmt.Sprintf("`%s` is no tier the role grants: `%s`", c.Tier, TierLarge)
	case SizeTier(is.Labels) != "":
		return "size-kept", fmt.Sprintf("its label `%s` says its tier already: a tier only goes up, and past `%s` a person's", TierLabel(SizeTier(is.Labels)), TierLarge)
	}
	return "", ""
}

// SplitOff says whether the project turned `split` off itself, not by the
// role's default: no split then, even past an issue's size (ADR-0064, §3).
func (c Config) SplitOff() bool {
	s, ok := c.Acts["split"]
	return ok && s.Mode == Off && (c.Origins["split"] == FromSetting || slices.Contains(c.ModeSet, "split"))
}
