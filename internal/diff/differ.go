// Package diff implements the DropSpec × WiredConfig → ConfigDelta triple —
// the core primitive. The differ walks the curated registry, finds successor
// drops of the user's wired model, and emits only the config-affecting
// deltas. The Affects boolean is the moat: "does this delta touch a field
// the user actually wired?"
package diff

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SuperMarioYL/dropdiff/internal/config"
	"github.com/SuperMarioYL/dropdiff/internal/spec"
)

// Differ is stateless; the zero value is ready to use.
type Differ struct{}

// ConfigDelta is one config-affecting change in a single drop.
type ConfigDelta struct {
	Model    string
	Vendor   string
	DropDate string
	Field    string
	Before   string
	After    string
	Affects  bool   // does this delta touch a field the user wired?
	Severity string // "red" | "green" | "amber"
	Note     string
}

// New returns a ready Differ.
func New() *Differ { return &Differ{} }

// Diff compares the registry's drops against the user's wired config and
// returns only config-affecting deltas. It:
//  1. Looks up the wired model's own DropSpec in the registry (the baseline).
//  2. Finds successor drops (explicit `successor` field, else same-vendor
//     newer drops when no explicit successor exists).
//  3. Diffs baseline → successor per field, marking Affects by whether the
//     user wired that field.
//
// When the wired model is NOT in the registry, it falls back to diffing the
// newest same-vendor drop against the literal wired fields (context, cache).
func (d *Differ) Diff(wired *config.WiredConfig, reg *spec.Registry) []ConfigDelta {
	var deltas []ConfigDelta
	if wired == nil || reg == nil || wired.Model == "" {
		return deltas
	}

	baseline := reg.FindByModel(wired.Model)

	// Primary path: explicit successor chain.
	succs := reg.SuccessorsOf(wired.Model)
	// Fallback: if no explicit successor but baseline exists, treat same-vendor
	// newer drops as successors (helps when the registry omits `successor`).
	if len(succs) == 0 && baseline != nil {
		for i := range reg.Drops {
			drop := &reg.Drops[i]
			if drop.Vendor == baseline.Vendor && drop.DropDate > baseline.DropDate && drop.Model != baseline.Model {
				succs = append(succs, drop)
			}
		}
	}

	if baseline == nil {
		// Edge case: wired model not curated. Diff the newest same-vendor drop
		// against the literal wired fields so the user still gets a signal.
		vendor := inferVendor(wired.Model)
		if vendor == "" {
			return deltas
		}
		if drop := reg.NewestOfVendor(vendor); drop != nil {
			deltas = append(deltas, diffAgainstWired(wired, drop)...)
		}
		return deltas
	}

	for _, drop := range succs {
		deltas = append(deltas, diffSpecs(wired, baseline, drop)...)
	}
	return deltas
}

// diffSpecs diffs a baseline DropSpec against a successor drop, producing one
// ConfigDelta per changed field.
func diffSpecs(wired *config.WiredConfig, base, drop *spec.DropSpec) []ConfigDelta {
	var out []ConfigDelta

	cacheWired := wired.CacheMode != "" &&
		!strings.EqualFold(wired.CacheMode, "off") &&
		!strings.EqualFold(wired.CacheMode, "none") &&
		!strings.EqualFold(wired.CacheMode, "false")
	ctxWired := wired.Context != ""

	// Pricing — input/output/cache per M tokens. Refuse to diff across
	// mismatched units (the carousel mixes CNY and USD quotes), but never go
	// silent on a unit switch: surface it as an amber informational delta so
	// a currency change cannot hide a pricing change (m5).
	if base.Pricing.Unit == drop.Pricing.Unit {
		out = appendDelta(out, drop, "pricing.input",
			fmtPrice(base.Pricing.Input, base.Pricing.Unit), fmtPrice(drop.Pricing.Input, drop.Pricing.Unit),
			true, severityPrice(base.Pricing.Input, drop.Pricing.Input), "input token price")
		out = appendDelta(out, drop, "pricing.output",
			fmtPrice(base.Pricing.Output, base.Pricing.Unit), fmtPrice(drop.Pricing.Output, drop.Pricing.Unit),
			true, severityPrice(base.Pricing.Output, drop.Pricing.Output), "output token price")
		out = appendDelta(out, drop, "pricing.cache",
			fmtPrice(base.Pricing.Cache, base.Pricing.Unit), fmtPrice(drop.Pricing.Cache, drop.Pricing.Unit),
			cacheWired, severityPrice(base.Pricing.Cache, drop.Pricing.Cache), "implicit cache price")
	} else {
		out = append(out, ConfigDelta{
			Model: drop.Model, Vendor: drop.Vendor, DropDate: drop.DropDate,
			Field: "pricing.unit", Before: base.Pricing.Unit, After: drop.Pricing.Unit,
			Affects: false, Severity: "amber",
			Note: "pricing unit changed — prices not comparable across units; verify vs your bill",
		})
	}

	// Context window.
	out = appendDelta(out, drop, "context",
		fmt.Sprintf("%d", base.Context), fmt.Sprintf("%d", drop.Context),
		ctxWired, severityInt(base.Context, drop.Context), "max context window")

	// Available quants (set difference). Informational — the minimal wired
	// config carries no quant field, so this never claims to "affect" the
	// wired config; it's a heads-up for self-hosters.
	removed, added := setDiff(base.Quants, drop.Quants)
	if len(removed) > 0 || len(added) > 0 {
		out = append(out, ConfigDelta{
			Model: drop.Model, Vendor: drop.Vendor, DropDate: drop.DropDate,
			Field:    "quants",
			Before:   strings.Join(base.Quants, ","),
			After:    strings.Join(drop.Quants, ","),
			Affects:  false,
			Severity: severityQuants(removed, added),
			Note:     fmt.Sprintf("quants removed=[%s] added=[%s]", strings.Join(removed, ","), strings.Join(added, ",")),
		})
	}

	// Tool-call contract — the field a coding agent cares most about.
	switch {
	case base.ToolCall.Supported != drop.ToolCall.Supported:
		out = appendDelta(out, drop, "tool_call.supported",
			fmt.Sprintf("%v", base.ToolCall.Supported), fmt.Sprintf("%v", drop.ToolCall.Supported),
			true, severityBool(base.ToolCall.Supported, drop.ToolCall.Supported), "tool-call support")
	case base.ToolCall.Format != drop.ToolCall.Format:
		out = appendDelta(out, drop, "tool_call.format",
			base.ToolCall.Format, drop.ToolCall.Format,
			true, "amber", "tool-call format changed")
	case base.ToolCall.Quirk != drop.ToolCall.Quirk:
		out = appendDelta(out, drop, "tool_call.quirk",
			base.ToolCall.Quirk, drop.ToolCall.Quirk,
			true, "amber", "tool-call quirk changed")
	}

	return out
}

// diffAgainstWired handles the edge case where the wired model is not in the
// registry. We can only diff the literal wired fields; pricing has no
// baseline so it's surfaced as an amber heads-up rather than a fake delta.
func diffAgainstWired(wired *config.WiredConfig, drop *spec.DropSpec) []ConfigDelta {
	var out []ConfigDelta
	if drop == nil {
		return out
	}
	if wired.Context != "" {
		if wc, ok := parseInt(wired.Context); ok && wc != drop.Context {
			out = append(out, ConfigDelta{
				Model: drop.Model, Vendor: drop.Vendor, DropDate: drop.DropDate,
				Field: "context", Before: wired.Context, After: fmt.Sprintf("%d", drop.Context),
				Affects: true, Severity: severityInt(wc, drop.Context),
				Note: "context: wired vs newest drop (baseline spec not in registry)",
			})
		}
	}
	if wired.CacheMode != "" &&
		!strings.EqualFold(wired.CacheMode, "off") &&
		!strings.EqualFold(wired.CacheMode, "none") {
		out = append(out, ConfigDelta{
			Model: drop.Model, Vendor: drop.Vendor, DropDate: drop.DropDate,
			Field: "pricing.cache", Before: "—", After: fmtPrice(drop.Pricing.Cache, drop.Pricing.Unit),
			Affects: true, Severity: "amber",
			Note: "cache pricing on newest drop (baseline not in registry — verify vs your bill)",
		})
	}
	return out
}

func appendDelta(out []ConfigDelta, drop *spec.DropSpec, field, before, after string, affects bool, severity, note string) []ConfigDelta {
	if before == after {
		return out
	}
	return append(out, ConfigDelta{
		Model: drop.Model, Vendor: drop.Vendor, DropDate: drop.DropDate,
		Field: field, Before: before, After: after,
		Affects: affects, Severity: severity, Note: note,
	})
}

// --- severity helpers --------------------------------------------------------

func fmtPrice(p float64, unit string) string {
	if unit == "" {
		unit = "¥/M"
	}
	return fmt.Sprintf("%.2f %s", p, unit)
}

func severityPrice(before, after float64) string {
	switch {
	case after < before:
		return "green"
	case after > before:
		return "red"
	default:
		return "amber"
	}
}

func severityInt(before, after int) string {
	switch {
	case after > before:
		return "green"
	case after < before:
		return "red"
	default:
		return "amber"
	}
}

func severityBool(before, after bool) string {
	switch {
	case !before && after:
		return "green"
	case before && !after:
		return "red"
	default:
		return "amber"
	}
}

func severityQuants(removed, added []string) string {
	switch {
	case len(removed) > 0 && len(added) == 0:
		return "red"
	case len(added) > 0 && len(removed) == 0:
		return "green"
	default:
		return "amber"
	}
}

// setDiff returns (removed from base, added in drop), lowercased + sorted.
func setDiff(base, drop []string) (removed, added []string) {
	bs := toSet(base)
	ds := toSet(drop)
	for k := range bs {
		if !ds[k] {
			removed = append(removed, k)
		}
	}
	for k := range ds {
		if !bs[k] {
			added = append(added, k)
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	return
}

func toSet(s []string) map[string]bool {
	m := map[string]bool{}
	for _, v := range s {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" {
			m[v] = true
		}
	}
	return m
}

func inferVendor(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "deepseek"):
		return "DeepSeek"
	case strings.Contains(m, "qwen"):
		return "Qwen"
	case strings.Contains(m, "glm"):
		return "GLM"
	case strings.Contains(m, "kimi"), strings.Contains(m, "moonshot"):
		return "Kimi"
	}
	return ""
}

func parseInt(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return n, true
}
