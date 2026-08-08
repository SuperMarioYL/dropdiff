// Package spec defines the DropSpec primitive and the curated drop registry.
//
// The core data model is the DropSpec: one row of the weekly 国产模型 carousel,
// sourced from a ModelScope card + a 国产 SDK commit SHA. See mvp_plan §2.
package spec

// PriceDelta captures per-million-token pricing for a single drop.
// Currency/unit is carried explicitly so the differ can refuse to compare
// across mismatched units (the carousel mixes CNY and USD quotes).
type PriceDelta struct {
	Input  float64 `yaml:"input"`
	Output float64 `yaml:"output"`
	Cache  float64 `yaml:"cache"`
	Unit   string  `yaml:"unit"` // "CNY/M" or "USD/M"
}

// ToolCallSpec describes a drop's function-calling contract — the field a
// coding agent (the target audience) cares most about after pricing.
type ToolCallSpec struct {
	Supported bool   `yaml:"supported"`
	Format    string `yaml:"format"` // "openai-tools", "native", "xml", ...
	Quirk     string `yaml:"quirk"`  // free-text quirk note
}

// SourceRef points at the canonical sources of a drop spec. The SDK commit
// URL is load-bearing: the GLM 5.3 drop surfaced first in a z-ai-sdk-java
// commit, not on the ModelScope card.
type SourceRef struct {
	ModelScopeCard string `yaml:"modelscope_card"`
	SDKCommitSHA   string `yaml:"sdk_commit_sha"`
	SDKCommitURL   string `yaml:"sdk_commit_url"`
	RedditThread   string `yaml:"reddit_thread,omitempty"`
}

// DropSpec is one row of the curated registry: a single model drop.
type DropSpec struct {
	Model     string       `yaml:"model"`
	Vendor    string       `yaml:"vendor"`
	DropDate  string       `yaml:"drop_date"`
	Pricing   PriceDelta   `yaml:"pricing"`
	Context   int          `yaml:"context"`
	Quants    []string     `yaml:"quants"`
	ToolCall  ToolCallSpec `yaml:"tool_call"`
	Source    SourceRef    `yaml:"source"`
	Successor string       `yaml:"successor,omitempty"` // model id this drop succeeds
}

// FamilyLabel returns the human vendor label.
func (d DropSpec) FamilyLabel() string { return d.Vendor }
