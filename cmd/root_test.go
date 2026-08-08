package cmd

import "testing"

func TestRootHasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"init", "sync", "check", "list"} {
		if !names[want] {
			t.Errorf("root command missing subcommand %q (have: %v)", want, names)
		}
	}
}

func TestVersion(t *testing.T) {
	if Version != "0.1.0" {
		t.Errorf("Version=%q want 0.1.0", Version)
	}
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should be set so --version works")
	}
}

func TestDefaultRegistryURL(t *testing.T) {
	if DefaultRegistryURL == "" {
		t.Error("DefaultRegistryURL must be non-empty")
	}
}
