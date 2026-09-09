// Package config reads the developer's already-wired model config from the
// usual suspects — ~/.claude.json, ~/.aider/conf.yml, .env — and snapshots it
// to ~/.dropdiff/wired.yaml. That snapshot is the baseline the differ diffs
// the weekly drop carousel against.
//
// This is milestone m1. The "wired config" artifact is the moat: a generic
// Renovate cannot diff the carousel because it doesn't know your base_url /
// model / cache_mode. DropDiff does.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WiredConfig is the normalized snapshot of what the user has wired into
// their coding agent. Field set matches mvp_plan §2's WiredConfig.
type WiredConfig struct {
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
	Context   string `yaml:"context,omitempty"`
	CacheMode string `yaml:"cache_mode,omitempty"`
	Source    string `yaml:"source"`
}

// DropDiffDir returns ~/.dropdiff.
func DropDiffDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".dropdiff"), nil
}

// WiredPath returns the path to ~/.dropdiff/wired.yaml.
func WiredPath() (string, error) {
	d, err := DropDiffDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "wired.yaml"), nil
}

// RegistryPath returns the path to the synced ~/.dropdiff/registry.yaml.
func RegistryPath() (string, error) {
	d, err := DropDiffDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "registry.yaml"), nil
}

// Save writes the wired config to ~/.dropdiff/wired.yaml, creating the dir.
func Save(cfg *WiredConfig) (string, error) {
	dir, err := DropDiffDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "wired.yaml")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads the wired config from ~/.dropdiff/wired.yaml.
func Load() (*WiredConfig, error) {
	p, err := WiredPath()
	if err != nil {
		return nil, err
	}
	return LoadFile(p)
}

// LoadFile reads a wired config from a specific path.
func LoadFile(path string) (*WiredConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg WiredConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DetectAndParse auto-detects ~/.claude.json, ~/.aider/conf.yml, and .env
// (in that order) and merges them into a single WiredConfig. The first source
// found for each field wins; later sources only fill gaps. This is the m1
// happy path: run `dropdiff init`, then `cat ~/.dropdiff/wired.yaml`.
func DetectAndParse(workDir string) (*WiredConfig, error) {
	if workDir == "" {
		workDir = "."
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var sources []string
	cfg := &WiredConfig{}

	// 1. Claude Code config (JSON preferred; CLAUDE.md scanned as text).
	for _, p := range []string{
		filepath.Join(home, ".claude.json"),
		filepath.Join(home, ".claude", "config.json"),
		filepath.Join(workDir, ".claude.json"),
		filepath.Join(workDir, "CLAUDE.md"),
	} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			if c, err := parseClaude(p); err == nil {
				mergeWired(cfg, c)
				sources = append(sources, p)
				break
			}
		}
	}

	// 2. Aider config (YAML).
	for _, p := range []string{
		filepath.Join(home, ".aider", "conf.yml"),
		filepath.Join(home, ".aider.conf.yml"),
		filepath.Join(workDir, ".aider.conf.yml"),
	} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			if c, err := parseAider(p); err == nil {
				mergeWired(cfg, c)
				sources = append(sources, p)
				break
			}
		}
	}

	// 3. .env (cwd wins over $HOME).
	for _, p := range []string{
		filepath.Join(workDir, ".env"),
		filepath.Join(home, ".env"),
	} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			if c, err := parseEnv(p); err == nil {
				mergeWired(cfg, c)
				sources = append(sources, p)
				break
			}
		}
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no wired config found — looked for ~/.claude.json, ~/.aider/conf.yml, ./.env")
	}
	cfg.Source = strings.Join(sources, ", ")
	return cfg, nil
}

// parseClaude reads a Claude Code config file. JSON files are parsed
// structurally; CLAUDE.md (markdown) is scanned for key: value / key=value
// hints since developers sometimes inline the wired endpoint there.
func parseClaude(path string) (*WiredConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".json") {
		return parseClaudeJSON(b)
	}
	cfg := &WiredConfig{}
	applyKV(cfg, scanKeyValue(string(b)))
	return cfg, nil
}

func parseClaudeJSON(b []byte) (*WiredConfig, error) {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	cfg := &WiredConfig{}
	if v, ok := raw["primaryModel"].(string); ok && v != "" {
		cfg.Model = v
	}
	if v, ok := raw["model"].(string); ok && v != "" && cfg.Model == "" {
		cfg.Model = v
	}
	if env, ok := raw["env"].(map[string]any); ok {
		for k, v := range env {
			s, _ := v.(string)
			if s == "" {
				continue
			}
			switch strings.ToUpper(k) {
			case "ANTHROPIC_BASE_URL", "OPENAI_BASE_URL", "BASE_URL":
				if cfg.BaseURL == "" {
					cfg.BaseURL = s
				}
			case "ANTHROPIC_MODEL", "OPENAI_MODEL", "MODEL":
				if cfg.Model == "" {
					cfg.Model = s
				}
			case "MAX_CONTEXT", "CONTEXT", "MAX_CONTEXT_TOKENS":
				if cfg.Context == "" {
					cfg.Context = s
				}
			case "CACHE_MODE", "USE_CACHE":
				if cfg.CacheMode == "" {
					cfg.CacheMode = s
				}
			}
		}
	}
	return cfg, nil
}

// parseAider reads an aider conf.yml (flat YAML) and picks the wired keys.
func parseAider(path string) (*WiredConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		// fall back to a plain key: value scan if it's not clean YAML
		cfg := &WiredConfig{}
		applyKV(cfg, scanKeyValue(string(b)))
		return cfg, nil
	}
	cfg := &WiredConfig{}
	if v, ok := raw["model"].(string); ok && v != "" {
		cfg.Model = v
	}
	if v, ok := raw["openai_api_base"].(string); ok && v != "" {
		cfg.BaseURL = v
	}
	if v, ok := raw["cache_mode"].(string); ok && v != "" {
		cfg.CacheMode = v
	}
	switch c := raw["max_context"].(type) {
	case string:
		cfg.Context = c
	case int:
		cfg.Context = fmt.Sprintf("%d", c)
	}
	return cfg, nil
}

// parseEnv reads a .env file (KEY=VALUE lines).
func parseEnv(path string) (*WiredConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &WiredConfig{}
	applyKV(cfg, scanKeyValue(string(b)))
	return cfg, nil
}

// kvFieldMap maps a normalized (uppercased) key to a WiredConfig field.
var kvFieldMap = map[string]string{
	// base_url
	"OPENAI_API_BASE": "base_url", "OPENAI_BASE_URL": "base_url",
	"ANTHROPIC_BASE_URL": "base_url", "BASE_URL": "base_url",
	"DEEPSEEK_BASE_URL": "base_url", "QWEN_BASE_URL": "base_url",
	"GLM_BASE_URL": "base_url", "KIMI_BASE_URL": "base_url",
	"MOONSHOT_BASE_URL": "base_url", "API_BASE": "base_url",
	"OPENAI_API_BASE_URL": "base_url",
	// model
	"MODEL": "model", "OPENAI_MODEL": "model", "ANTHROPIC_MODEL": "model",
	"DEEPSEEK_MODEL": "model", "QWEN_MODEL": "model", "GLM_MODEL": "model",
	"KIMI_MODEL": "model", "MOONSHOT_MODEL": "model", "PRIMARYMODEL": "model",
	// context
	"MAX_CONTEXT": "context", "CONTEXT": "context", "MAX_CONTEXT_TOKENS": "context",
	"CONTEXT_WINDOW": "context",
	// cache_mode
	"CACHE_MODE": "cache_mode", "USE_CACHE": "cache_mode", "ENABLE_CACHE": "cache_mode",
}

func applyKV(cfg *WiredConfig, kv map[string]string) {
	for k, v := range kv {
		field, ok := kvFieldMap[k]
		if !ok {
			continue
		}
		switch field {
		case "base_url":
			if cfg.BaseURL == "" {
				cfg.BaseURL = v
			}
		case "model":
			if cfg.Model == "" {
				cfg.Model = v
			}
		case "context":
			if cfg.Context == "" {
				cfg.Context = v
			}
		case "cache_mode":
			if cfg.CacheMode == "" {
				cfg.CacheMode = v
			}
		}
	}
}

// scanKeyValue parses KEY=VALUE and key: value lines into a map with
// uppercased keys. Comments (#) and blank lines are skipped. Handles .env,
// aider-style, and loose markdown hints uniformly.
func scanKeyValue(text string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var key, val string
		if i := strings.Index(line, "="); i >= 0 {
			key = strings.TrimSpace(line[:i])
			val = strings.TrimSpace(stripQuotes(line[i+1:]))
		} else if i := strings.Index(line, ":"); i >= 0 {
			key = strings.TrimSpace(line[:i])
			val = strings.TrimSpace(stripQuotes(line[i+1:]))
		} else {
			continue
		}
		// Strip a leading "- " / "* " list bullet BEFORE normalizing hyphens:
		// after ReplaceAll("-","_") the bullet's hyphen is already an underscore
		// and the trim could never fire (m4 — "- KEY=V" used to parse as "_ KEY").
		key = strings.ToUpper(strings.ReplaceAll(strings.TrimLeft(key, "*- "), "-", "_"))
		if val != "" {
			if _, dup := m[key]; !dup {
				m[key] = val
			}
		}
	}
	return m
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func mergeWired(dst, src *WiredConfig) {
	if dst.Model == "" {
		dst.Model = src.Model
	}
	if dst.BaseURL == "" {
		dst.BaseURL = src.BaseURL
	}
	if dst.Context == "" {
		dst.Context = src.Context
	}
	if dst.CacheMode == "" {
		dst.CacheMode = src.CacheMode
	}
}
