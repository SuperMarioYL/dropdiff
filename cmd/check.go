package cmd

import (
	"fmt"

	"github.com/SuperMarioYL/dropdiff/internal/diff"
	"github.com/SuperMarioYL/dropdiff/internal/render"
	"github.com/spf13/cobra"
)

var checkStrict bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Print red/green config-affecting deltas for this week's drops",
	Long: `Diffs the (synced or bundled) drop-spec registry against your wired
config (~/.dropdiff/wired.yaml) and prints only the config-affecting
deltas in red/green via lipgloss — pricing, context, cache price, available
quants, tool-call quirks. Deltas that touch a field you actually wired are
flagged "affects your wired config"; the rest are informational.

This is m3 — the "yes I'd star it" moment is check printing
  GLM-5.3: cache 0.28 -> 0.14 CNY/M (affects your wired GLM-4.6)  green
before the ModelScope card even updates — caught from the SDK commit log.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wiredPath, err := wiredConfigPath()
		if err != nil {
			return err
		}
		wired, err := loadWired(wiredPath)
		if err != nil {
			return fmt.Errorf("check: wired config not found at %s — run `dropdiff init` first: %w", wiredPath, err)
		}
		reg, regSource, err := loadRegistry()
		if err != nil {
			return err
		}
		d := diff.New()
		deltas := d.Diff(wired, reg)

		fmt.Printf("wired: %s  (model=%s, base_url=%s)\n", wiredPath, wired.Model, wired.BaseURL)
		fmt.Printf("registry: %s  (%d drops, updated %s)\n\n", regSource, len(reg.Drops), reg.Updated)
		fmt.Print(render.Render(deltas))
		if len(deltas) > 0 {
			fmt.Println(render.Summary(deltas))
		}
		// m9: opt-in CI/agent gating — non-zero exit only when a delta is both
		// red (regressive) and Affects=true (touches a field the user wired).
		// The default print-only exit-0 contract is unchanged without --strict.
		if checkStrict {
			red := 0
			for _, d := range deltas {
				if d.Severity == "red" && d.Affects {
					red++
				}
			}
			if red > 0 {
				cmd.SilenceUsage = true
				return fmt.Errorf("check: %d red config-affecting delta(s) — failing due to --strict", red)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "exit non-zero when red config-affecting deltas exist (CI gating)")
}
