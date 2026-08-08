package render_test

import (
	"strings"
	"testing"

	"github.com/SuperMarioYL/dropdiff/internal/config"
	"github.com/SuperMarioYL/dropdiff/internal/diff"
	"github.com/SuperMarioYL/dropdiff/internal/render"
	"github.com/SuperMarioYL/dropdiff/internal/spec"
)

func TestRenderEmpty(t *testing.T) {
	out := render.Render(nil)
	if !strings.Contains(out, "no config-affecting deltas") {
		t.Errorf("empty render should mention no deltas; got %q", out)
	}
}

func TestRenderBundledGLMCatch(t *testing.T) {
	reg, err := spec.Default()
	if err != nil {
		t.Fatal(err)
	}
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	out := render.Render(deltas)
	for _, want := range []string{"glm-5.3", "pricing.cache", "0.28", "0.14"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\n--- output ---\n%s", want, out)
		}
	}
	if !strings.Contains(out, "affects your wired config") {
		t.Errorf("render should flag affecting deltas\n%s", out)
	}
}

func TestRenderSummary(t *testing.T) {
	reg, err := spec.Default()
	if err != nil {
		t.Fatal(err)
	}
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	s := render.Summary(deltas)
	if !strings.Contains(s, "delta") {
		t.Errorf("summary=%q", s)
	}
}

func TestRenderRedForQwen(t *testing.T) {
	reg, err := spec.Default()
	if err != nil {
		t.Fatal(err)
	}
	wired := &config.WiredConfig{Model: "qwen3-vl", CacheMode: "implicit", Context: "131072"}
	deltas := diff.New().Diff(wired, reg)
	out := render.Render(deltas)
	if !strings.Contains(out, "qwen3.8-max") {
		t.Errorf("render should include qwen3.8-max\n%s", out)
	}
	// the qwen drop is a price-up → at least one red glyph present
	if !strings.Contains(out, "✗") {
		t.Errorf("render should contain a red (✗) glyph for qwen price-up\n%s", out)
	}
}
