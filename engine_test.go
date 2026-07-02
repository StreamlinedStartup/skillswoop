package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineStarCommandsWithLocalRepo(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine skill walker")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	writeSkill(t, repo, "tdd", "Test-driven development.")
	writeSkill(t, repo, "review", "Code review.")

	config := filepath.Join(root, "config")
	data := filepath.Join(root, "data")
	run := func(args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{"engine/swoop-core"}, args...)...)
		cmd.Env = append(os.Environ(),
			"XDG_CONFIG_HOME="+config,
			"XDG_DATA_HOME="+data,
			"NO_COLOR=1",
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	if out, err := run("star", repo, "tdd"); err != nil {
		t.Fatalf("star failed: %v\n%s", err, out)
	}
	out, err := run("stars")
	if err != nil {
		t.Fatalf("stars failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, repo+"\ttdd\tTest-driven development.") {
		t.Fatalf("stars output missing tdd entry:\n%s", out)
	}
	if out, err := run("unstar", repo, "tdd"); err != nil {
		t.Fatalf("unstar failed: %v\n%s", err, out)
	}
	out, err = run("stars")
	if err != nil {
		t.Fatalf("stars after unstar failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "\ttdd\t") {
		t.Fatalf("expected tdd to be removed:\n%s", out)
	}
}

func TestEnginePluginsLocalMarketplace(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine plugin walker")
	}
	root := t.TempDir()
	mkt := filepath.Join(root, "mkt")
	writeMarketplace(t, mkt)

	config := filepath.Join(root, "config")
	data := filepath.Join(root, "data")
	bin := writeStubCLIs(t, root)
	run := func(args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{"engine/swoop-core"}, args...)...)
		cmd.Env = append(os.Environ(),
			"XDG_CONFIG_HOME="+config,
			"XDG_DATA_HOME="+data,
			"NO_COLOR=1",
			"SWOOP_DRYRUN=1",
			"SWOOP_ASSUME_YES=1",
			"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	out, err := run("_plugins", mkt)
	if err != nil {
		t.Fatalf("_plugins failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "@marketplace\ttest-mkt\ttest-mkt-codex") {
		t.Fatalf("_plugins output missing marketplace header:\n%s", out)
	}
	if !strings.Contains(out, "hooky\tA plugin with hooks and mcp.\tclaude,codex,hooks,mcp") {
		t.Fatalf("_plugins output missing hooky flags:\n%s", out)
	}

	if out, err = run("mkt", "add", mkt); err != nil {
		t.Fatalf("mkt add failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"[dry-run] claude plugin marketplace add",
		"[dry-run] codex plugin marketplace add",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("mkt add output missing %q:\n%s", want, out)
		}
	}
	out, err = run("_mkts")
	if err != nil {
		t.Fatalf("_mkts failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, mkt+"\ttest-mkt\ttest-mkt-codex") {
		t.Fatalf("_mkts missing cached names:\n%s", out)
	}

	// hook-bearing plugin + codex hooks off ⇒ engine auto-enables (SWOOP_ASSUME_YES)
	out, err = run("plugin", "install", mkt, "hooky")
	if err != nil {
		t.Fatalf("plugin install failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"[dry-run] claude plugin install hooky@test-mkt --scope project",
		"[dry-run] codex features enable hooks",
		"[dry-run] codex plugin add hooky@test-mkt-codex",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plugin install output missing %q:\n%s", want, out)
		}
	}
	// --no-hooks-enable suppresses the features toggle; -g maps to --scope user
	out, err = run("-g", "plugin", "install", mkt, "hooky", "--no-hooks-enable")
	if err != nil {
		t.Fatalf("plugin install --no-hooks-enable failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "features enable hooks") {
		t.Fatalf("--no-hooks-enable still enabled hooks:\n%s", out)
	}
	if !strings.Contains(out, "[dry-run] claude plugin install hooky@test-mkt --scope user") {
		t.Fatalf("-g did not map to --scope user:\n%s", out)
	}
}

func TestEnginePluginsClaudeOnlyMarketplaceSkipsCodex(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine plugin walker")
	}
	root := t.TempDir()
	mkt := filepath.Join(root, "mkt")
	writeJSONFile(t, filepath.Join(mkt, ".claude-plugin", "marketplace.json"),
		`{ "name": "conly", "plugins": [ { "name": "a", "source": "./plugins/a", "description": "d" } ] }`)

	bin := writeStubCLIs(t, root)
	cmd := exec.Command("bash", "engine/swoop-core", "mkt", "add", mkt)
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+filepath.Join(root, "config"),
		"XDG_DATA_HOME="+filepath.Join(root, "data"),
		"NO_COLOR=1",
		"SWOOP_DRYRUN=1",
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	outB, err := cmd.CombinedOutput()
	out := string(outB)
	if err != nil {
		t.Fatalf("mkt add failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "skipped codex") {
		t.Fatalf("expected a codex skip warning:\n%s", out)
	}
	if strings.Contains(out, "codex plugin marketplace add") {
		t.Fatalf("codex marketplace add should not run for a claude-only repo:\n%s", out)
	}
}

// writeMarketplace lays out a repo carrying both manifest formats and one
// plugin with hooks + mcp components.
func writeMarketplace(t *testing.T, mkt string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(mkt, ".claude-plugin", "marketplace.json"), `{
  "name": "test-mkt",
  "plugins": [
    { "name": "hooky", "source": "./plugins/hooky", "description": "A plugin with hooks and mcp." },
    { "name": "plain", "source": "./plugins/plain", "description": "Nothing fancy." }
  ]
}`)
	writeJSONFile(t, filepath.Join(mkt, ".agents", "plugins", "marketplace.json"), `{
  "name": "test-mkt-codex",
  "plugins": [ { "name": "hooky", "source": "./plugins/hooky", "description": "A plugin with hooks and mcp." } ]
}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "hooky", "hooks", "hooks.json"), `{}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "hooky", ".mcp.json"), `{}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "plain", "plugin.json"), `{"name":"plain"}`)
}

func writeJSONFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeStubCLIs drops fake claude/codex binaries on PATH so no real CLI is
// needed in CI; codex reports features.hooks off.
func writeStubCLIs(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	claude := "#!/bin/sh\necho \"stub-claude $*\"\n"
	codex := "#!/bin/sh\nif [ \"$1 $2\" = \"features list\" ]; then echo \"hooks    beta    off\"; exit 0; fi\necho \"stub-codex $*\"\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(claude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(codex), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func writeSkill(t *testing.T, repo, name, desc string) {
	t.Helper()
	dir := filepath.Join(repo, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\ndescription: " + desc + "\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
