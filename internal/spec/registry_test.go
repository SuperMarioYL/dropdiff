package spec

import (
	"strings"
	"testing"
)

func TestDefaultRegistryParses(t *testing.T) {
	reg, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Drops) < 8 {
		t.Fatalf("bundled registry has %d drops, want >=8 (m2 done-check)", len(reg.Drops))
	}
	if reg.Updated == "" {
		t.Error("registry has empty `updated` field")
	}
	vendors := map[string]bool{}
	for _, d := range reg.Drops {
		vendors[d.Vendor] = true
		if d.Model == "" || d.DropDate == "" {
			t.Errorf("drop missing required fields: %+v", d)
		}
		if d.Source.ModelScopeCard == "" && d.Source.SDKCommitSHA == "" {
			t.Errorf("drop %q has no source (m2 requires sourced rows)", d.Model)
		}
	}
	for _, want := range []string{"DeepSeek", "Qwen", "GLM", "Kimi"} {
		if !vendors[want] {
			t.Errorf("registry missing vendor %q", want)
		}
	}
}

func TestParseRoundtrip(t *testing.T) {
	in := []byte("updated: 2026-01-01\ndrops:\n  - model: glm-5.3\n    vendor: GLM\n    drop_date: 2026-07-29\n    pricing: {input: 0.5, output: 0.5, cache: 0.14, unit: CNY/M}\n    context: 128000\n    quants: [fp16]\n    tool_call: {supported: true, format: openai-tools}\n    successor: glm-4.6\n")
	reg, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Drops) != 1 {
		t.Fatalf("got %d drops want 1", len(reg.Drops))
	}
	d := reg.Drops[0]
	if d.Model != "glm-5.3" || d.Vendor != "GLM" || d.Context != 128000 {
		t.Errorf("parsed drop wrong: %+v", d)
	}
	if d.Pricing.Cache != 0.14 || d.Pricing.Unit != "CNY/M" {
		t.Errorf("pricing wrong: %+v", d.Pricing)
	}
	if !d.ToolCall.Supported || d.ToolCall.Format != "openai-tools" {
		t.Errorf("toolcall wrong: %+v", d.ToolCall)
	}
	if d.Successor != "glm-4.6" {
		t.Errorf("successor=%q want glm-4.6", d.Successor)
	}
}

func TestFindByModel(t *testing.T) {
	reg, _ := Default()
	if d := reg.FindByModel("GLM-4.6"); d == nil || d.Model != "glm-4.6" {
		t.Error("FindByModel case-insensitive lookup failed for GLM-4.6")
	}
	if d := reg.FindByModel("does-not-exist"); d != nil {
		t.Error("FindByModel should return nil for unknown model")
	}
}

func TestSuccessorsOf(t *testing.T) {
	reg, _ := Default()
	succs := reg.SuccessorsOf("glm-4.6")
	if len(succs) != 1 || succs[0].Model != "glm-5.3" {
		t.Errorf("successors of glm-4.6 = %v, want [glm-5.3]", succs)
	}
	if s := reg.SuccessorsOf("deepseek-v4"); len(s) != 1 || s[0].Model != "deepseek-v4.1" {
		t.Errorf("successors of deepseek-v4 = %v", s)
	}
}

func TestNewestOfVendor(t *testing.T) {
	reg, _ := Default()
	d := reg.NewestOfVendor("GLM")
	if d == nil || d.Model != "glm-5.3" {
		t.Errorf("newest GLM = %+v want glm-5.3", d)
	}
}

func TestDropString(t *testing.T) {
	d := DropSpec{Model: "glm-5.3", Vendor: "GLM", DropDate: "2026-07-29"}
	s := d.String()
	if !strings.Contains(s, "glm-5.3") || !strings.Contains(s, "GLM") {
		t.Errorf("String()=%q", s)
	}
}
