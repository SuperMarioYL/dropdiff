package cmd

import (
	"fmt"

	"github.com/SuperMarioYL/dropdiff/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Detect & snapshot your wired model config to ~/.dropdiff/wired.yaml",
	Long: `Detects ~/.claude.json (or ./CLAUDE.md), ~/.aider/conf.yml, and .env,
extracts the model + base_url + context + cache_mode you have wired into
your coding agent, and snapshots them to ~/.dropdiff/wired.yaml.

This is m1 — the wired-config artifact is the baseline every later diff
runs against. A generic Renovate cannot diff the carousel because it does
not know your base_url; dropdiff does.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, _ := cmd.Flags().GetString("cwd")
		cfg, err := config.DetectAndParse(workDir)
		if err != nil {
			return err
		}
		path, err := config.Save(cfg)
		if err != nil {
			return err
		}
		fmt.Printf("snapshotted wired config -> %s\n", path)
		fmt.Printf("  model      : %s\n", cfg.Model)
		fmt.Printf("  base_url   : %s\n", cfg.BaseURL)
		fmt.Printf("  context    : %s\n", cfg.Context)
		fmt.Printf("  cache_mode : %s\n", cfg.CacheMode)
		fmt.Printf("  source     : %s\n", cfg.Source)
		fmt.Println("\nnext: run `dropdiff check` to diff this week's drops against this config.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("cwd", "", "working directory to scan for .env / CLAUDE.md (default: current dir)")
}
