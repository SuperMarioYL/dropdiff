package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanKeyValue(t *testing.T) {
	in := "MODEL=glm-4.6\nBASE_URL=https://open.bigmodel.cn\n# a comment\nCACHE_MODE=implicit\nMAX_CONTEXT=128000\n"
	m := scanKeyValue(in)
	cases := map[string]string{
		"MODEL":       "glm-4.6",
		"BASE_URL":    "https://open.bigmodel.cn",
		"CACHE_MODE":  "implicit",
		"MAX_CONTEXT": "128000",
	}
	for k, want := range cases {
		if m[k] != want {
			t.Errorf("scanKeyValue[%q] = %q, want %q", k, m[k], want)
		}
	}
}

func TestScanKeyValueStripsQuotes(t *testing.T) {
	m := scanKeyValue(`MODEL="glm-4.6"` + "\n" + `BASE_URL='https://x'` + "\n")
	if m["MODEL"] != "glm-4.6" {
		t.Fatalf("MODEL=%q want glm-4.6", m["MODEL"])
	}
	if m["BASE_URL"] != "https://x" {
		t.Fatalf("BASE_URL=%q want https://x", m["BASE_URL"])
	}
}

// m4: "- " list bullets (markdown's default, also YAML lists) must parse to
// the same key as the unbulleted form — previously the hyphen-to-underscore
// normalization ran first and "- MODEL=..." parsed as key "_ MODEL".
func TestScanKeyValueStripsListBullets(t *testing.T) {
	lines := []string{
		"- MODEL=glm-4.6",
		"* MODEL=glm-4.6",
		"- OPENAI_API_BASE=https://open.bigmodel.cn",
		"- model: glm-4.6",
		"- MAX_CONTEXT: 128000",
	}
	for _, line := range lines {
		m := scanKeyValue(line)
		if len(m) != 1 {
			t.Errorf("scanKeyValue(%q) = %v, want exactly one key", line, m)
			continue
		}
		for k, v := range m {
			switch k {
			case "MODEL":
				if v != "glm-4.6" {
					t.Errorf("scanKeyValue(%q)[%q] = %q, want glm-4.6", line, k, v)
				}
			case "OPENAI_API_BASE":
				if v != "https://open.bigmodel.cn" {
					t.Errorf("scanKeyValue(%q)[%q] = %q", line, k, v)
				}
			case "MAX_CONTEXT":
				if v != "128000" {
					t.Errorf("scanKeyValue(%q)[%q] = %q", line, k, v)
				}
			default:
				t.Errorf("scanKeyValue(%q) produced mangled key %q", line, k)
			}
		}
	}
}

// m8: shell-style "export KEY=VALUE" lines (a common real-world .env form)
// must parse to the same key as the unprefixed form — previously the key
// normalized to "EXPORT KEY" and never matched kvFieldMap.
func TestScanKeyValueStripsExportPrefix(t *testing.T) {
	lines := []string{
		"export MODEL=glm-4.6",
		"EXPORT OPENAI_API_BASE=https://open.bigmodel.cn",
		"export CACHE_MODE=implicit",
		"export MAX_CONTEXT: 128000",
	}
	for _, line := range lines {
		m := scanKeyValue(line)
		if len(m) != 1 {
			t.Errorf("scanKeyValue(%q) = %v, want exactly one key", line, m)
			continue
		}
		for k, v := range m {
			switch k {
			case "MODEL":
				if v != "glm-4.6" {
					t.Errorf("scanKeyValue(%q)[%q] = %q, want glm-4.6", line, k, v)
				}
			case "OPENAI_API_BASE":
				if v != "https://open.bigmodel.cn" {
					t.Errorf("scanKeyValue(%q)[%q] = %q", line, k, v)
				}
			case "CACHE_MODE":
				if v != "implicit" {
					t.Errorf("scanKeyValue(%q)[%q] = %q", line, k, v)
				}
			case "MAX_CONTEXT":
				if v != "128000" {
					t.Errorf("scanKeyValue(%q)[%q] = %q", line, k, v)
				}
			default:
				t.Errorf("scanKeyValue(%q) produced mangled key %q", line, k)
			}
		}
	}
}

func TestScanKeyValueMixedPlainAndExport(t *testing.T) {
	m := scanKeyValue("MODEL=glm-4.6\nexport OPENAI_API_BASE=https://open.bigmodel.cn\n")
	if m["MODEL"] != "glm-4.6" {
		t.Errorf("MODEL=%q want glm-4.6", m["MODEL"])
	}
	if m["OPENAI_API_BASE"] != "https://open.bigmodel.cn" {
		t.Errorf("OPENAI_API_BASE=%q want https://open.bigmodel.cn", m["OPENAI_API_BASE"])
	}
}

func TestParseEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := "OPENAI_API_BASE=https://api.deepseek.com\nMODEL=deepseek-v4\nCACHE_MODE=implicit\nMAX_CONTEXT=128000\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseEnv(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "deepseek-v4" {
		t.Errorf("Model=%q want deepseek-v4", cfg.Model)
	}
	if cfg.BaseURL != "https://api.deepseek.com" {
		t.Errorf("BaseURL=%q want api.deepseek.com", cfg.BaseURL)
	}
	if cfg.CacheMode != "implicit" {
		t.Errorf("CacheMode=%q want implicit", cfg.CacheMode)
	}
	if cfg.Context != "128000" {
		t.Errorf("Context=%q want 128000", cfg.Context)
	}
}

func TestParseClaudeJSON(t *testing.T) {
	in := []byte(`{"primaryModel":"glm-4.6","env":{"ANTHROPIC_BASE_URL":"https://open.bigmodel.cn","MAX_CONTEXT":"128000","CACHE_MODE":"implicit"}}`)
	cfg, err := parseClaudeJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "glm-4.6" {
		t.Errorf("Model=%q want glm-4.6", cfg.Model)
	}
	if cfg.BaseURL != "https://open.bigmodel.cn" {
		t.Errorf("BaseURL=%q want open.bigmodel.cn", cfg.BaseURL)
	}
	if cfg.Context != "128000" {
		t.Errorf("Context=%q want 128000", cfg.Context)
	}
	if cfg.CacheMode != "implicit" {
		t.Errorf("CacheMode=%q want implicit", cfg.CacheMode)
	}
}

func TestParseAider(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "conf.yml")
	content := "model: qwen3-vl\nopenai_api_base: https://dashscope.aliyuncs.com\ncache_mode: implicit\nmax_context: 131072\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseAider(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "qwen3-vl" {
		t.Errorf("Model=%q want qwen3-vl", cfg.Model)
	}
	if cfg.BaseURL != "https://dashscope.aliyuncs.com" {
		t.Errorf("BaseURL=%q", cfg.BaseURL)
	}
	if cfg.Context != "131072" {
		t.Errorf("Context=%q want 131072 (int->string)", cfg.Context)
	}
	if cfg.CacheMode != "implicit" {
		t.Errorf("CacheMode=%q", cfg.CacheMode)
	}
}

func TestDetectAndParse(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// claude.json carries model; .env carries base_url → merge
	claude := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(claude, []byte(`{"primaryModel":"glm-4.6","env":{"CACHE_MODE":"implicit","MAX_CONTEXT":"128000"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte("OPENAI_API_BASE=https://open.bigmodel.cn\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := DetectAndParse(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "glm-4.6" {
		t.Errorf("Model=%q", cfg.Model)
	}
	if cfg.BaseURL != "https://open.bigmodel.cn" {
		t.Errorf("BaseURL=%q", cfg.BaseURL)
	}
	if cfg.CacheMode != "implicit" {
		t.Errorf("CacheMode=%q", cfg.CacheMode)
	}
	if cfg.Context != "128000" {
		t.Errorf("Context=%q", cfg.Context)
	}
	if !strings.Contains(cfg.Source, ".claude.json") || !strings.Contains(cfg.Source, ".env") {
		t.Errorf("Source=%q should list both sources", cfg.Source)
	}
}

func TestDetectAndParseNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if _, err := DetectAndParse(dir); err == nil {
		t.Fatal("expected error when no config sources are present")
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	in := &WiredConfig{
		Model: "glm-4.6", BaseURL: "https://open.bigmodel.cn",
		Context: "128000", CacheMode: "implicit", Source: "test",
	}
	path, err := Save(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "wired.yaml") {
		t.Errorf("path=%q", path)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != in.Model || got.BaseURL != in.BaseURL || got.Context != in.Context || got.CacheMode != in.CacheMode {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestInferKVFieldMap(t *testing.T) {
	// ensure common keys map so init doesn't silently drop fields
	for _, k := range []string{"MODEL", "OPENAI_API_BASE", "ANTHROPIC_BASE_URL", "CACHE_MODE", "MAX_CONTEXT"} {
		if _, ok := kvFieldMap[k]; !ok {
			t.Errorf("kvFieldMap missing %q", k)
		}
	}
}
