package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/SuperMarioYL/dropdiff/internal/config"
	"github.com/SuperMarioYL/dropdiff/internal/spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull the latest drop-spec registry from the hosted URL",
	Long: `Fetches the curated drop-spec registry (the JSON/YAML refresh that
records each week's DeepSeek/Qwen/Kimi/GLM drops with pricing/context/cache/
quant/tool-call fields, each row sourced from a ModelScope card + a 国产 SDK
commit SHA) and writes it to ~/.dropdiff/registry.yaml.

This is m2 — the curated registry is the labor; sync is the one command that
refreshes it without you wading through ModelScope cards + commit logs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		if url == "" {
			url = viper.GetString("registry_url")
		}

		dir, err := config.DropDiffDir()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		outPath := dir + string(os.PathSeparator) + "registry.yaml"

		client := &http.Client{Timeout: 20 * time.Second}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("sync: build request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("sync: fetch %s: %w\n  (offline? `dropdiff check` still works on the bundled registry)", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("sync: HTTP %d for %s", resp.StatusCode, url)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("sync: read body: %w", err)
		}
		reg, err := spec.Parse(body)
		if err != nil {
			return fmt.Errorf("sync: invalid registry YAML: %w", err)
		}
		if err := os.WriteFile(outPath, body, 0o644); err != nil {
			return fmt.Errorf("sync: write %s: %w", outPath, err)
		}
		fmt.Printf("synced %d drops (updated %s) -> %s\n", len(reg.Drops), reg.Updated, outPath)
		fmt.Printf("source: %s\n", url)
		fmt.Println("\nnext: run `dropdiff list` to see the drops, `dropdiff check` to diff.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().String("url", "", "registry URL (default: hosted raw GitHub; override with $DROPDIFF_REGISTRY_URL)")
}
