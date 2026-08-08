package cmd_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// findModuleRoot walks up from the test's working dir to the dir holding go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", cwd)
		}
		dir = parent
	}
}

// goEnv caches the host's GOPATH/GOCACHE so we can pass them through to the
// `go run` child. Without this, `go run` writes its read-only module cache
// under the temp HOME and t.TempDir's cleanup fails with "permission denied".
var goEnvCache struct {
	sync.Once
	gopath, gocache string
}

func goEnv() (gopath, gocache string) {
	goEnvCache.Do(func() {
		goEnvCache.gopath = goEnvOne("GOPATH")
		goEnvCache.gocache = goEnvOne("GOCACHE")
	})
	return goEnvCache.gopath, goEnvCache.gocache
}

func goEnvOne(key string) string {
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// dropEnv builds the env for a `go run` child: isolate HOME (so dropdiff's
// ~/.dropdiff lands in the temp dir) while keeping GOPATH/GOCACHE on the host
// so the go toolchain does not pollute the temp dir with read-only files.
func dropEnv(home string) []string {
	env := append(os.Environ(), "HOME="+home)
	if gp, gc := goEnv(); gp != "" {
		env = append(env, "GOPATH="+gp, "GOCACHE="+gc)
	}
	return env
}

// runDrop compiles+runs the dropdiff binary via `go run .` with a temp HOME
// (so ~/.dropdiff lands in the temp dir) and the given args.
func runDrop(t *testing.T, root, home, workdir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = root
	cmd.Env = dropEnv(home)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("dropdiff %v failed: %v\n%s", args, err, out.String())
	}
	return out.String()
}

// TestIntegrationHappyPath exercises the full m1->m2->m3 happy path against
// the bundled registry (offline): init snapshots a wired glm-4.6 config,
// check prints the GLM-5.3 cache-price catch, list shows the registry.
func TestIntegrationHappyPath(t *testing.T) {
	root := findModuleRoot(t)
	home := t.TempDir()
	work := t.TempDir()

	// .env wiring glm-4.6 with cache_mode + context so deltas are "affecting".
	envContent := "MODEL=glm-4.6\nOPENAI_API_BASE=https://open.bigmodel.cn\nCACHE_MODE=implicit\nMAX_CONTEXT=128000\n"
	if err := os.WriteFile(filepath.Join(work, ".env"), []byte(envContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// m1: init snapshots the wired config.
	out := runDrop(t, root, home, work, "init", "--cwd", work)
	for _, want := range []string{"glm-4.6", "wired.yaml", "implicit"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output missing %q\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".dropdiff", "wired.yaml")); err != nil {
		t.Errorf("init did not write ~/.dropdiff/wired.yaml: %v", err)
	}

	// m3: check prints the GLM-5.3 cache-price catch (0.28 -> 0.14), green.
	out = runDrop(t, root, home, work, "check")
	for _, want := range []string{"glm-5.3", "pricing.cache", "0.28", "0.14", "affects your wired config"} {
		if !strings.Contains(out, want) {
			t.Errorf("check output missing %q\n--- output ---\n%s", want, out)
		}
	}

	// m2: list shows the bundled registry (>=8 drops) with sources.
	out = runDrop(t, root, home, work, "list")
	if !strings.Contains(out, "MODEL") || !strings.Contains(out, "deepseek-v4") {
		t.Errorf("list output missing header/rows\n%s", out)
	}
	// count data rows (one per drop) — at least 8
	lines := strings.Split(out, "\n")
	dataRows := 0
	for _, l := range lines {
		if strings.Contains(l, "CNY/M") || strings.Contains(l, "2026-") {
			dataRows++
		}
	}
	if dataRows < 8 {
		t.Errorf("list shows %d drop rows, want >=8\n%s", dataRows, out)
	}

	// --version reports the VERSION.
	out = runDrop(t, root, home, work, "--version")
	if !strings.Contains(out, "0.1.0") {
		t.Errorf("--version output=%q", out)
	}
}

// TestIntegrationCheckBeforeInitErrors confirms check fails clearly when the
// user hasn't run init yet (no wired.yaml).
func TestIntegrationCheckBeforeInitErrors(t *testing.T) {
	root := findModuleRoot(t)
	home := t.TempDir()

	cmd := exec.Command("go", "run", ".", "check")
	cmd.Dir = root
	cmd.Env = dropEnv(home)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		t.Fatalf("check before init should exit non-zero\n%s", out.String())
	}
	if !strings.Contains(out.String(), "dropdiff init") {
		t.Errorf("check before init should hint `dropdiff init`; got\n%s", out.String())
	}
}
