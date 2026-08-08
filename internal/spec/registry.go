package spec

import (
	"fmt"
	"os"
	"strings"

	"github.com/SuperMarioYL/dropdiff/data"
	"gopkg.in/yaml.v3"
)

// Registry is the curated collection of DropSpecs the differ diffs against.
type Registry struct {
	Updated string     `yaml:"updated"`
	Drops   []DropSpec `yaml:"drops"`
}

// Parse decodes registry YAML bytes.
func Parse(b []byte) (*Registry, error) {
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// LoadFile reads a registry YAML from disk.
func LoadFile(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Default returns the bundled registry shipped with the binary. This is the
// fallback `dropdiff check` uses before the user ever runs `dropdiff sync`,
// so the happy path is wired-config → check with zero network calls.
func Default() (*Registry, error) {
	return Parse(data.RegistryYAML)
}

// SaveFile writes the registry to disk as YAML.
func (r *Registry) SaveFile(path string) error {
	b, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// FindByModel looks up a drop by model id (case-insensitive).
func (r *Registry) FindByModel(modelID string) *DropSpec {
	for i := range r.Drops {
		if strings.EqualFold(r.Drops[i].Model, modelID) {
			return &r.Drops[i]
		}
	}
	return nil
}

// SuccessorsOf returns drops that explicitly succeed the given model id
// (via the `successor` field). This is the primary matcher the differ uses:
// it keeps the diff to genuine upgrade paths instead of flagging every
// same-vendor drop as a change.
func (r *Registry) SuccessorsOf(modelID string) []*DropSpec {
	var out []*DropSpec
	for i := range r.Drops {
		if strings.EqualFold(r.Drops[i].Successor, modelID) {
			out = append(out, &r.Drops[i])
		}
	}
	return out
}

// NewestOfVendor returns the most recent drop for a vendor, tie-broken by
// model id. Used only in the no-baseline edge case.
func (r *Registry) NewestOfVendor(vendor string) *DropSpec {
	var best *DropSpec
	for i := range r.Drops {
		d := &r.Drops[i]
		if !strings.EqualFold(d.Vendor, vendor) {
			continue
		}
		if best == nil || d.DropDate > best.DropDate {
			best = d
		}
	}
	return best
}

func (d DropSpec) String() string {
	return fmt.Sprintf("%s (%s, %s)", d.Model, d.Vendor, d.DropDate)
}
