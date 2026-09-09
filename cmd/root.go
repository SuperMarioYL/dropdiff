// Package cmd implements the DropDiff CLI surface: init / sync / check / list.
package cmd

import (
	"fmt"
	"os"

	"github.com/SuperMarioYL/dropdiff/internal/config"
	"github.com/SuperMarioYL/dropdiff/internal/spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is the single source for the --version flag and the release tag.
var Version = "0.2.0"

// DefaultRegistryURL is the hosted raw-GitHub YAML the sync command pulls.
// It self-heals: the live release badge reads the same repo.
const DefaultRegistryURL = "https://raw.githubusercontent.com/SuperMarioYL/dropdiff/main/data/registry.yaml"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "dropdiff",
	Short: "Flag config-affecting deltas in weekly 国产模型 drops",
	Long: `DropDiff — the spec-diff CLI for the 国产模型 carousel.

It diffs this week's DeepSeek / Qwen / Kimi / GLM drops against your wired
config and prints only the config-affecting red/green deltas (pricing,
context, cache price, available quants, tool-call quirks).

  $ dropdiff init     # snapshot your wired model/base_url/context/cache
  $ dropdiff sync     # pull the latest drop-spec registry
  $ dropdiff check    # print red/green config-affecting deltas

Renovate/Dependabot for the Chinese model carousel — where releases arrive as
scattered ModelScope cards + 国产 SDK commit logs, not versioned packages.`,
	Version: Version,
}

// Execute runs the root command. Exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "wired config path (default ~/.dropdiff/wired.yaml)")

	// viper backs the sync URL + future knobs; $DROPDIFF_* env overrides.
	viper.SetEnvPrefix("DROPDIFF")
	viper.AutomaticEnv()
	viper.SetDefault("registry_url", DefaultRegistryURL)
}

// wiredConfigPath resolves the wired config path from --config or the default.
func wiredConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	return config.WiredPath()
}

// loadWired reads the wired config snapshot from path (defaults to the
// --config flag or ~/.dropdiff/wired.yaml).
func loadWired(path string) (*config.WiredConfig, error) {
	return config.LoadFile(path)
}

// loadRegistry prefers the synced ~/.dropdiff/registry.yaml and falls back to
// the bundled registry embedded in the binary — so `dropdiff check` works
// out of the box, with zero network calls, before the user ever syncs.
// Returns the registry plus a human label of where it came from.
func loadRegistry() (*spec.Registry, string, error) {
	if dir, err := config.DropDiffDir(); err == nil {
		synced := dir + string(os.PathSeparator) + "registry.yaml"
		if info, e := os.Stat(synced); e == nil && !info.IsDir() {
			if r, e := spec.LoadFile(synced); e == nil {
				return r, synced, nil
			}
		}
	}
	r, err := spec.Default()
	if err != nil {
		return nil, "", err
	}
	return r, "bundled (run `dropdiff sync` to refresh)", nil
}
