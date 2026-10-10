package diff_test

import (
	"strings"
	"testing"

	"github.com/SuperMarioYL/dropdiff/internal/config"
	"github.com/SuperMarioYL/dropdiff/internal/diff"
	"github.com/SuperMarioYL/dropdiff/internal/spec"
)

// miniRegistry mirrors the hero catch from the GTM: glm-4.6 -> glm-5.3 with
// cache price 0.28 -> 0.14 CNY/M, plus a quant added and unchanged pricing.
func miniRegistry() *spec.Registry {
	return &spec.Registry{Updated: "2026-08-09", Drops: []spec.DropSpec{
		{
			Model: "glm-4.6", Vendor: "GLM", DropDate: "2026-06-17",
			Pricing: spec.PriceDelta{Input: 0.5, Output: 0.5, Cache: 0.28, Unit: "CNY/M"},
			Context: 128000, Quants: []string{"fp16", "int8"},
			ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"},
			Source:   spec.SourceRef{ModelScopeCard: "m", SDKCommitSHA: "e5f6a7b"},
		},
		{
			Model: "glm-5.3", Vendor: "GLM", DropDate: "2026-07-29",
			Pricing: spec.PriceDelta{Input: 0.5, Output: 0.5, Cache: 0.14, Unit: "CNY/M"},
			Context: 128000, Quants: []string{"fp16", "int8", "fp8"},
			ToolCall:  spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"},
			Successor: "glm-4.6",
			Source:    spec.SourceRef{SDKCommitSHA: "9f0a1b2"},
		},
	}}
}

func findDelta(deltas []diff.ConfigDelta, field string) *diff.ConfigDelta {
	for i := range deltas {
		if deltas[i].Field == field {
			return &deltas[i]
		}
	}
	return nil
}

func TestDiffGLMCacheCatch(t *testing.T) {
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, miniRegistry())

	cache := findDelta(deltas, "pricing.cache")
	if cache == nil {
		t.Fatalf("no pricing.cache delta; got %+v", deltas)
	}
	if cache.Severity != "green" {
		t.Errorf("cache severity=%q want green", cache.Severity)
	}
	if !cache.Affects {
		t.Error("cache delta must Affects=true when cache_mode wired")
	}
	if !strings.Contains(cache.Before, "0.28") || !strings.Contains(cache.After, "0.14") {
		t.Errorf("cache before/after=%q -> %q", cache.Before, cache.After)
	}
	if cache.Model != "glm-5.3" {
		t.Errorf("delta Model=%q want glm-5.3", cache.Model)
	}
}

func TestDiffGLMQuantsInformational(t *testing.T) {
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit"}
	deltas := diff.New().Diff(wired, miniRegistry())
	q := findDelta(deltas, "quants")
	if q == nil {
		t.Fatalf("no quants delta; got %+v", deltas)
	}
	if q.Affects {
		t.Error("quants delta must be informational (Affects=false) — no quant in minimal wired config")
	}
	if q.Severity != "green" {
		t.Errorf("quants severity=%q want green (fp8 added)", q.Severity)
	}
}

func TestDiffGLMUnchangedPricingOmitted(t *testing.T) {
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit"}
	deltas := diff.New().Diff(wired, miniRegistry())
	// input/output price unchanged (0.5 -> 0.5) → no delta emitted
	if findDelta(deltas, "pricing.input") != nil {
		t.Error("unchanged input price should not produce a delta")
	}
	if findDelta(deltas, "pricing.output") != nil {
		t.Error("unchanged output price should not produce a delta")
	}
	// context unchanged (128000 -> 128000) → no delta
	if findDelta(deltas, "context") != nil {
		t.Error("unchanged context should not produce a delta")
	}
}

func TestDiffCacheNotWiredIsNotAffecting(t *testing.T) {
	// cache_mode unset → cache price delta is informational, not affecting
	wired := &config.WiredConfig{Model: "glm-4.6"}
	deltas := diff.New().Diff(wired, miniRegistry())
	cache := findDelta(deltas, "pricing.cache")
	if cache == nil {
		t.Fatal("expected a cache price delta even when cache not wired")
	}
	if cache.Affects {
		t.Error("cache delta must Affects=false when cache_mode is unset")
	}
}

func TestDiffDeepSeekContextUp(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "deepseek-v4", Vendor: "DeepSeek", DropDate: "2026-06-10", Pricing: spec.PriceDelta{Input: 1, Output: 2, Cache: 0.1, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}},
		{Model: "deepseek-v4.1", Vendor: "DeepSeek", DropDate: "2026-07-15", Pricing: spec.PriceDelta{Input: 1, Output: 2, Cache: 0.1, Unit: "CNY/M"}, Context: 256000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}, Successor: "deepseek-v4"},
	}}
	wired := &config.WiredConfig{Model: "deepseek-v4", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	ctx := findDelta(deltas, "context")
	if ctx == nil {
		t.Fatalf("no context delta; got %+v", deltas)
	}
	if ctx.Severity != "green" {
		t.Errorf("context severity=%q want green (128000->256000)", ctx.Severity)
	}
	if !ctx.Affects {
		t.Error("context delta must Affects when wired.Context set")
	}
}

func TestDiffQwenPriceUpRed(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "qwen3-vl", Vendor: "Qwen", DropDate: "2026-06-03", Pricing: spec.PriceDelta{Input: 0.5, Output: 1.5, Cache: 0.07, Unit: "CNY/M"}, Context: 131072, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}},
		{Model: "qwen3.8-max", Vendor: "Qwen", DropDate: "2026-07-20", Pricing: spec.PriceDelta{Input: 14, Output: 42, Cache: 1.75, Unit: "CNY/M"}, Context: 1048576, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "y"}, Successor: "qwen3-vl"},
	}}
	wired := &config.WiredConfig{Model: "qwen3-vl", CacheMode: "implicit", Context: "131072"}
	deltas := diff.New().Diff(wired, reg)
	for _, field := range []string{"pricing.input", "pricing.output", "pricing.cache"} {
		d := findDelta(deltas, field)
		if d == nil {
			t.Errorf("missing %s delta", field)
			continue
		}
		if d.Severity != "red" {
			t.Errorf("%s severity=%q want red (price up)", field, d.Severity)
		}
	}
	ctx := findDelta(deltas, "context")
	if ctx == nil || ctx.Severity != "green" {
		t.Errorf("context delta should be green (131072->1048576); got %+v", ctx)
	}
}

// m5: when a successor switches pricing unit (CNY/M -> USD/M), the differ
// must surface one amber pricing.unit delta instead of dropping every pricing
// signal — while still refusing numeric comparison across units.
func TestDiffPricingUnitChangeSurfaced(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "glm-4.6", Vendor: "GLM", DropDate: "2026-06-17", Pricing: spec.PriceDelta{Input: 2, Output: 2, Cache: 0.28, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}},
		{Model: "glm-5.3", Vendor: "GLM", DropDate: "2026-07-29", Pricing: spec.PriceDelta{Input: 0.28, Output: 0.28, Cache: 0.04, Unit: "USD/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}, Successor: "glm-4.6"},
	}}
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)

	unit := findDelta(deltas, "pricing.unit")
	if unit == nil {
		t.Fatalf("unit change must surface a pricing.unit delta; got %+v", deltas)
	}
	if unit.Before != "CNY/M" || unit.After != "USD/M" {
		t.Errorf("pricing.unit before/after=%q -> %q, want CNY/M -> USD/M", unit.Before, unit.After)
	}
	if unit.Severity != "amber" || unit.Affects {
		t.Errorf("pricing.unit delta should be amber informational; got severity=%q affects=%v", unit.Severity, unit.Affects)
	}
	// numbers are not comparable across units → no numeric pricing deltas
	for _, field := range []string{"pricing.input", "pricing.output", "pricing.cache"} {
		if d := findDelta(deltas, field); d != nil {
			t.Errorf("%s delta must not be emitted across units; got %+v", field, d)
		}
	}
	// same fields, same units → unchanged behavior
	sameUnit := &spec.Registry{Drops: []spec.DropSpec{
		reg.Drops[0],
		{Model: "glm-5.3", Vendor: "GLM", DropDate: "2026-07-29", Pricing: spec.PriceDelta{Input: 2, Output: 2, Cache: 0.14, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}, Successor: "glm-4.6"},
	}}
	deltas = diff.New().Diff(wired, sameUnit)
	if findDelta(deltas, "pricing.unit") != nil {
		t.Error("no pricing.unit delta when units match")
	}
	if cache := findDelta(deltas, "pricing.cache"); cache == nil {
		t.Error("same-unit cache price delta must still be emitted")
	}
}

func TestDiffToolCallQuirkAmber(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "kimi-k2.5", Vendor: "Kimi", DropDate: "2026-06-24", Pricing: spec.PriceDelta{Input: 0.5, Output: 1.5, Cache: 0.1, Unit: "CNY/M"}, Context: 256000, Quants: []string{"fp16"}, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "stable json mode"}},
		{Model: "kimi-k3", Vendor: "Kimi", DropDate: "2026-08-05", Pricing: spec.PriceDelta{Input: 0.5, Output: 1.5, Cache: 0.1, Unit: "CNY/M"}, Context: 256000, Quants: []string{"fp16", "fp8"}, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "improved json mode, parallel tool calls"}, Successor: "kimi-k2.5"},
	}}
	wired := &config.WiredConfig{Model: "kimi-k2.5"}
	deltas := diff.New().Diff(wired, reg)
	quirk := findDelta(deltas, "tool_call.quirk")
	if quirk == nil {
		t.Fatalf("no tool_call.quirk delta; got %+v", deltas)
	}
	if quirk.Severity != "amber" {
		t.Errorf("quirk severity=%q want amber", quirk.Severity)
	}
	if !quirk.Affects {
		t.Error("tool-call quirk delta must Affects=true (coding agent depends on tools)")
	}
}

// m7: the tool-call block must emit one delta per changed field. A switch
// here let whichever case matched first hide concurrent format/quirk changes.
func TestDiffToolCallMultiFieldEmitsEach(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "base", Vendor: "GLM", DropDate: "2026-06-01", Pricing: spec.PriceDelta{Input: 1, Output: 1, Cache: 0.1, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "quirk-A"}},
		{Model: "next", Vendor: "GLM", DropDate: "2026-07-01", Pricing: spec.PriceDelta{Input: 1, Output: 1, Cache: 0.1, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "native", Quirk: "quirk-B"}, Successor: "base"},
	}}
	wired := &config.WiredConfig{Model: "base"}
	deltas := diff.New().Diff(wired, reg)

	format := findDelta(deltas, "tool_call.format")
	if format == nil {
		t.Fatalf("missing tool_call.format delta; got %+v", deltas)
	}
	if format.Before != "openai-tools" || format.After != "native" {
		t.Errorf("format before/after=%q -> %q", format.Before, format.After)
	}
	quirk := findDelta(deltas, "tool_call.quirk")
	if quirk == nil {
		t.Fatalf("tool_call.quirk change must not be hidden behind the format change; got %+v", deltas)
	}
	if quirk.Before != "quirk-A" || quirk.After != "quirk-B" {
		t.Errorf("quirk before/after=%q -> %q", quirk.Before, quirk.After)
	}
	if findDelta(deltas, "tool_call.supported") != nil {
		t.Error("unchanged supported must not produce a delta")
	}
}

// m7: when every tool-call field changes, all three deltas appear.
func TestDiffToolCallAllFieldsEmit(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "base", Vendor: "Kimi", DropDate: "2026-06-01", Pricing: spec.PriceDelta{Input: 1, Output: 1, Cache: 0.1, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "quirk-A"}},
		{Model: "next", Vendor: "Kimi", DropDate: "2026-07-01", Pricing: spec.PriceDelta{Input: 1, Output: 1, Cache: 0.1, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: false, Format: "xml", Quirk: "quirk-B"}, Successor: "base"},
	}}
	wired := &config.WiredConfig{Model: "base"}
	deltas := diff.New().Diff(wired, reg)
	for _, field := range []string{"tool_call.supported", "tool_call.format", "tool_call.quirk"} {
		if findDelta(deltas, field) == nil {
			t.Errorf("missing %s delta; got %+v", field, deltas)
		}
	}
	if sup := findDelta(deltas, "tool_call.supported"); sup.Severity != "red" {
		t.Errorf("supported true->false severity=%q want red", sup.Severity)
	}
}

// m7: the no-baseline fallback must honor the same cache contract as the
// successor path — cache_mode=false means no Affects=true cache delta.
func TestDiffFallbackCacheModeFalse(t *testing.T) {
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "glm-5.3", Vendor: "GLM", DropDate: "2026-07-29", Pricing: spec.PriceDelta{Input: 0.5, Output: 0.5, Cache: 0.14, Unit: "CNY/M"}, Context: 256000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}},
	}}
	wired := &config.WiredConfig{Model: "glm-4.6-custom", CacheMode: "false", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	if cache := findDelta(deltas, "pricing.cache"); cache != nil {
		t.Errorf("cache_mode=false must not emit a cache delta in the fallback; got %+v", cache)
	}

	wired.CacheMode = "implicit"
	deltas = diff.New().Diff(wired, reg)
	cache := findDelta(deltas, "pricing.cache")
	if cache == nil {
		t.Fatalf("cache_mode=implicit fallback must emit the cache heads-up; got %+v", deltas)
	}
	if !cache.Affects {
		t.Error("fallback cache delta must Affects=true when cache mode is wired")
	}
}

func TestDiffNoSuccessorsNoDeltas(t *testing.T) {
	// wired model has no successor in registry → no deltas
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "glm-4.6", Vendor: "GLM", DropDate: "2026-06-17", Pricing: spec.PriceDelta{Input: 0.5, Output: 0.5, Cache: 0.28, Unit: "CNY/M"}, Context: 128000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}},
	}}
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	if len(deltas) != 0 {
		t.Errorf("expected no deltas, got %+v", deltas)
	}
}

func TestDiffWiredModelNotInRegistryFallback(t *testing.T) {
	// wired model not curated → fallback diffs newest same-vendor drop
	reg := &spec.Registry{Drops: []spec.DropSpec{
		{Model: "glm-5.3", Vendor: "GLM", DropDate: "2026-07-29", Pricing: spec.PriceDelta{Input: 0.5, Output: 0.5, Cache: 0.14, Unit: "CNY/M"}, Context: 256000, ToolCall: spec.ToolCallSpec{Supported: true, Format: "openai-tools", Quirk: "x"}},
	}}
	wired := &config.WiredConfig{Model: "glm-4.6-custom", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	ctx := findDelta(deltas, "context")
	if ctx == nil {
		t.Fatalf("fallback should emit context delta; got %+v", deltas)
	}
	if !ctx.Affects {
		t.Error("fallback context delta must Affects when wired.Context set")
	}
}

func TestDiffNilInputs(t *testing.T) {
	if got := diff.New().Diff(nil, miniRegistry()); len(got) != 0 {
		t.Error("Diff(nil, reg) should be empty")
	}
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit"}
	if got := diff.New().Diff(wired, nil); len(got) != 0 {
		t.Error("Diff(wired, nil) should be empty")
	}
}

func TestDiffBundledRegistryEndToEnd(t *testing.T) {
	reg, err := spec.Default()
	if err != nil {
		t.Fatal(err)
	}
	wired := &config.WiredConfig{Model: "glm-4.6", CacheMode: "implicit", Context: "128000"}
	deltas := diff.New().Diff(wired, reg)
	cache := findDelta(deltas, "pricing.cache")
	if cache == nil || !strings.Contains(cache.After, "0.14") {
		t.Errorf("end-to-end: expected GLM 5.3 cache catch; got %+v", cache)
	}
}
