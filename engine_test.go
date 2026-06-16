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
