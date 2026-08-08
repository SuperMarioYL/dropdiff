// Package render formats ConfigDeltas into a red/green terminal view via
// lipgloss. Green = beneficial (price down, context up), red = regressive
// (price up, context down, tool-call lost), amber = neutral/ambiguous.
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SuperMarioYL/dropdiff/internal/diff"
	"github.com/charmbracelet/lipgloss"
)

var (
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5484D")).Bold(true)
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("#30A46C")).Bold(true)
	amber  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A623")).Bold(true)
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8E9196"))
	header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0071E3"))
)

func styleFor(sev string) lipgloss.Style {
	switch sev {
	case "red":
		return red
	case "green":
		return green
	default:
		return amber
	}
}

func glyph(sev string) string {
	switch sev {
	case "red":
		return "✗"
	case "green":
		return "✓"
	default:
		return "•"
	}
}

// Render formats the deltas into a colored string grouped by model/drop.
func Render(deltas []diff.ConfigDelta) string {
	if len(deltas) == 0 {
		return green.Render("✓ no config-affecting deltas this week — your wired config is current.") + "\n"
	}
	var b strings.Builder
	byModel := groupByModel(deltas)
	for _, model := range orderedModels(deltas) {
		ds := byModel[model]
		b.WriteString("\n")
		b.WriteString(header.Render(fmt.Sprintf("%s  %s  [%s]", model, ds[0].DropDate, ds[0].Vendor)))
		b.WriteString("\n")
		for _, d := range ds {
			line := fmt.Sprintf("  %s %s: %s -> %s", glyph(d.Severity), d.Field, d.Before, d.After)
			b.WriteString(styleFor(d.Severity).Render(line))
			b.WriteString("\n")
			suffix := "informational"
			if d.Affects {
				suffix = "affects your wired config"
			}
			b.WriteString(dim.Render(fmt.Sprintf("      %s — %s", suffix, d.Note)))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Summary returns a one-line count for the sync/check footers.
func Summary(deltas []diff.ConfigDelta) string {
	var r, g, a int
	for _, d := range deltas {
		switch d.Severity {
		case "red":
			r++
		case "green":
			g++
		default:
			a++
		}
	}
	return fmt.Sprintf("%d delta(s): %d red, %d green, %d amber", len(deltas), r, g, a)
}

func groupByModel(deltas []diff.ConfigDelta) map[string][]diff.ConfigDelta {
	m := map[string][]diff.ConfigDelta{}
	for _, d := range deltas {
		m[d.Model] = append(m[d.Model], d)
	}
	return m
}

func orderedModels(deltas []diff.ConfigDelta) []string {
	seen := map[string]bool{}
	var models []string
	for _, d := range deltas {
		if !seen[d.Model] {
			seen[d.Model] = true
			models = append(models, d.Model)
		}
	}
	sort.Strings(models)
	return models
}
