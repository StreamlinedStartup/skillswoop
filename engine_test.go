package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
	// flags union both formats' dirs (commands from the claude dir, hooks/mcp/
	// skills from the codex dir); relpath must be the CODEX manifest's dir
	if !strings.Contains(out, "hooky\tA plugin with hooks and mcp.\tclaude,codex,hooks,mcp,commands,skills\tplugins/hooky\n") {
		t.Fatalf("_plugins output missing hooky flags/relpath:\n%s", out)
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

	// project scope: claude installs with --scope project; codex hooks auto-enable
	// (SWOOP_ASSUME_YES) and the plugin is vendored into the repo, not codex-added
	out, err = run("plugin", "install", mkt, "hooky")
	if err != nil {
		t.Fatalf("plugin install failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"[dry-run] claude plugin install hooky@test-mkt --scope project",
		"[dry-run] codex features enable hooks",
		"[dry-run] vendor plugins/hooky -> ./plugins/hooky",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plugin install output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "codex plugin add") {
		t.Fatalf("project scope must vendor, not codex plugin add:\n%s", out)
	}
	// --no-hooks-enable suppresses the features toggle; -g maps to --scope user
	// and installs user-wide via codex plugin add
	out, err = run("-g", "plugin", "install", mkt, "hooky", "--no-hooks-enable")
	if err != nil {
		t.Fatalf("plugin install --no-hooks-enable failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "features enable hooks") {
		t.Fatalf("--no-hooks-enable still enabled hooks:\n%s", out)
	}
	for _, want := range []string{
		"[dry-run] claude plugin install hooky@test-mkt --scope user",
		"[dry-run] codex plugin add hooky@test-mkt-codex",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("-g install output missing %q:\n%s", want, out)
		}
	}
}

func TestEngineCodexProjectInstallVendorsPlugin(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine plugin walker")
	}
	root := t.TempDir()
	mkt := filepath.Join(root, "mkt")
	writeMarketplace(t, mkt)
	proj := filepath.Join(root, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := writeStubCLIs(t, root)
	engine, err := filepath.Abs("engine/swoop-core")
	if err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{engine}, args...)...)
		cmd.Dir = proj // live file ops must land in the project dir
		cmd.Env = append(os.Environ(),
			"XDG_CONFIG_HOME="+filepath.Join(root, "config"),
			"XDG_DATA_HOME="+filepath.Join(root, "data"),
			"NO_COLOR=1",
			"SWOOP_ASSUME_YES=1",
			"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	if out, err := run("mkt", "add", mkt); err != nil {
		t.Fatalf("mkt add failed: %v\n%s", err, out)
	}
	out, err := run("plugin", "install", mkt, "hooky")
	if err != nil {
		t.Fatalf("plugin install failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, "plugins", "hooky", "hooks", "hooks.json")); err != nil {
		t.Fatalf("vendored hooks.json missing: %v\n%s", err, out)
	}
	mj, err := os.ReadFile(filepath.Join(proj, ".agents", "plugins", "marketplace.json"))
	if err != nil {
		t.Fatalf("repo marketplace.json missing: %v\n%s", err, out)
	}
	for _, want := range []string{`"hooky"`, `"./plugins/hooky"`, `"local"`} {
		if !strings.Contains(string(mj), want) {
			t.Fatalf("marketplace.json missing %s:\n%s", want, mj)
		}
	}
	if !strings.Contains(out, "/plugins") || !strings.Contains(out, "/hooks") {
		t.Fatalf("expected /plugins install and /hooks trust guidance:\n%s", out)
	}

	// removal un-vendors the plugin and drops its marketplace entry
	if out, err = run("plugin", "remove", "hooky"); err != nil {
		t.Fatalf("plugin remove failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, "plugins", "hooky")); !os.IsNotExist(err) {
		t.Fatalf("vendored plugin dir still present after remove\n%s", out)
	}
	mj, err = os.ReadFile(filepath.Join(proj, ".agents", "plugins", "marketplace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mj), `"hooky"`) {
		t.Fatalf("marketplace.json still lists hooky after remove:\n%s", mj)
	}
}

func TestEngineMktAddDedupesSourceSpellings(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine plugin walker")
	}
	root := t.TempDir()
	mkt := filepath.Join(root, "mkt")
	writeMarketplace(t, mkt)
	bin := writeStubCLIs(t, root)

	run := func(args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{"engine/swoop-core"}, args...)...)
		cmd.Env = append(os.Environ(),
			"XDG_CONFIG_HOME="+filepath.Join(root, "config"),
			"XDG_DATA_HOME="+filepath.Join(root, "data"),
			"NO_COLOR=1",
			"SWOOP_DRYRUN=1",
			"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// two spellings of the same local marketplace must collapse to one entry
	rel, err := filepath.Rel(mustGetwd(t), mkt)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := run("mkt", "add", "./"+rel); err != nil {
		t.Fatalf("mkt add (relative) failed: %v\n%s", err, out)
	}
	if out, err := run("mkt", "add", mkt); err != nil {
		t.Fatalf("mkt add (absolute) failed: %v\n%s", err, out)
	}
	out, err := run("_mkts")
	if err != nil {
		t.Fatalf("_mkts failed: %v\n%s", err, out)
	}
	lines := strings.Count(strings.TrimSpace(out), "\n") + 1
	if strings.TrimSpace(out) == "" || lines != 1 {
		t.Fatalf("expected exactly one marketplace entry, got:\n%s", out)
	}
	if !strings.HasPrefix(out, mkt+"\t") {
		t.Fatalf("entry not stored under the canonical path:\n%s", out)
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
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
// plugin with hooks + mcp components. The two formats point at different
// plugin dirs (like real dual-format repos) — codex vendoring must use the
// codex manifest's dir.
func writeMarketplace(t *testing.T, mkt string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(mkt, ".claude-plugin", "marketplace.json"), `{
  "name": "test-mkt",
  "plugins": [
    { "name": "hooky", "source": "./plugins/hooky-claude", "description": "A plugin with hooks and mcp." },
    { "name": "plain", "source": "./plugins/plain", "description": "Nothing fancy." }
  ]
}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "hooky-claude", ".claude-plugin", "plugin.json"),
		`{"name":"hooky","commands":"./commands/"}`)
	writeJSONFile(t, filepath.Join(mkt, ".agents", "plugins", "marketplace.json"), `{
  "name": "test-mkt-codex",
  "plugins": [ { "name": "hooky", "source": { "source": "local", "path": "./plugins/hooky" }, "description": "A plugin with hooks and mcp." } ]
}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "hooky", "hooks", "hooks.json"), `{}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "hooky", ".mcp.json"), `{}`)
	writeJSONFile(t, filepath.Join(mkt, "plugins", "hooky", ".codex-plugin", "plugin.json"),
		`{"name":"hooky","version":"1.0.0","skills":"./skills/","hooks":"./hooks/hooks.json"}`)
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

// runUpdate runs the engine in dir with a stub `skills` that records each call.
func runUpdate(t *testing.T, dir, config string, args ...string) (out string, calls string, err error) {
	t.Helper()
	return runUpdateWith(t, dir, config, "", nil, args...)
}

// runUpdateWith is runUpdate with extra lines for the skills stub (run after
// logging) and extra stub executables. Stubs live on PATH and find their files
// through $SWOOP_TEST_DIR, so paths with spaces stay intact.
func runUpdateWith(t *testing.T, dir, config, skillsTail string, stubs map[string]string, args ...string) (out string, calls string, err error) {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	all := map[string]string{"swoop-test-skills": "#!/bin/sh\necho \"$*\" >> \"$SWOOP_TEST_DIR/calls\"\n" + skillsTail}
	for name, body := range stubs {
		all[name] = body
	}
	for name, body := range all {
		writeJSONFile(t, filepath.Join(bin, name), body)
		if err := os.Chmod(filepath.Join(bin, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("bash", append([]string{filepath.Join(mustGetwd(t), "engine/swoop-core")}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+config,
		"XDG_DATA_HOME="+filepath.Join(root, "data"),
		"NO_COLOR=1",
		"SWOOP_SKILLS_CMD=swoop-test-skills",
		"SWOOP_TEST_DIR="+root,
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"PWD="+dir,
	)
	b, err := cmd.CombinedOutput()
	c, _ := os.ReadFile(filepath.Join(root, "calls"))
	return string(b), string(c), err
}

func realTempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func writeProjects(t *testing.T, config string, dirs ...string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(config, "swoop", "projects"), strings.Join(dirs, "\n")+"\n")
}

func TestEngineUpdateUnknownProjectFails(t *testing.T) {
	config, dir := t.TempDir(), realTempDir(t)
	writeJSONFile(t, filepath.Join(dir, "skills-lock.json"), "{}")
	writeProjects(t, config, filepath.Join(dir, "other"))

	out, calls, err := runUpdate(t, dir, config, "update")
	if err == nil {
		t.Fatalf("expected non-zero exit:\n%s", out)
	}
	if !strings.Contains(out, "swoop does not know this project") {
		t.Fatalf("missing error message:\n%s", out)
	}
	if calls != "" {
		t.Fatalf("skills was called: %q", calls)
	}
}

func TestEngineUpdateKnownProjectRuns(t *testing.T) {
	config, dir := t.TempDir(), realTempDir(t)
	writeJSONFile(t, filepath.Join(dir, "skills-lock.json"), "{}")
	writeProjects(t, config, dir)

	out, calls, err := runUpdate(t, dir, config, "update")
	if err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}
	if calls != "update -p -y\n" {
		t.Fatalf("calls = %q", calls)
	}
}

func TestEngineUpdateAllIncludesGlobal(t *testing.T) {
	config, root := t.TempDir(), realTempDir(t)
	withLock := filepath.Join(root, "with-lock")
	noLock := filepath.Join(root, "no-lock")
	gone := filepath.Join(root, "gone")
	writeJSONFile(t, filepath.Join(withLock, "skills-lock.json"), "{}")
	if err := os.MkdirAll(noLock, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProjects(t, config, withLock, noLock, gone)

	out, calls, err := runUpdate(t, root, config, "update", "--all", "--dry-run")
	if err != nil {
		t.Fatalf("update --all failed: %v\n%s", err, out)
	}
	if calls != "" {
		t.Fatalf("dry run called skills: %q", calls)
	}
	for _, want := range []string{
		"updating " + withLock,
		"skip " + noLock,
		"dropped " + gone,
		"updating global skills",
		"update -g -y",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	reg, _ := os.ReadFile(filepath.Join(config, "swoop", "projects"))
	if strings.Contains(string(reg), gone) {
		t.Fatalf("gone dir kept in registry:\n%s", reg)
	}

	out, calls, err = runUpdate(t, root, config, "update", "--all")
	if err != nil {
		t.Fatalf("update --all failed: %v\n%s", err, out)
	}
	if calls != "update -p -y\nupdate -g -y\n" {
		t.Fatalf("calls = %q", calls)
	}
}

func TestEngineUpdateAllWithoutProjectsStillUpdatesGlobal(t *testing.T) {
	out, calls, err := runUpdate(t, realTempDir(t), t.TempDir(), "update", "--all")
	if err != nil {
		t.Fatalf("update --all failed: %v\n%s", err, out)
	}
	if calls != "update -g -y\n" {
		t.Fatalf("calls = %q", calls)
	}
}

func TestEngineGlobalUpdateUnchanged(t *testing.T) {
	out, calls, err := runUpdate(t, realTempDir(t), t.TempDir(), "-g", "update")
	if err != nil {
		t.Fatalf("-g update failed: %v\n%s", err, out)
	}
	if calls != "update -g -y\n" {
		t.Fatalf("calls = %q", calls)
	}
}

func TestEngineUpdateAllFailsWhenGlobalFails(t *testing.T) {
	out, calls, err := runUpdateWith(t, realTempDir(t), t.TempDir(), "exit 3\n", nil, "update", "--all")
	if err == nil {
		t.Fatalf("expected non-zero exit:\n%s", out)
	}
	if !strings.Contains(out, "failed global skills") {
		t.Fatalf("missing failure line:\n%s", out)
	}
	if calls != "update -g -y\n" {
		t.Fatalf("calls = %q", calls)
	}
}

func TestEngineMenuUpdateThisFolderIgnoresGlobalFlag(t *testing.T) {
	// a space in TMPDIR must not break the stubs (the menu would loop forever)
	tmp := filepath.Join(t.TempDir(), "tmp dir")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmp)
	// stub gum: pick "Update this folder" once, then quit the menu
	gum := "#!/bin/sh\n[ \"$1\" = choose ] || exit 1\n[ -f \"$SWOOP_TEST_DIR/picked\" ] && exit 1\ntouch \"$SWOOP_TEST_DIR/picked\"\ngrep '^Update this folder'\n"

	out, calls, err := runUpdateWith(t, realTempDir(t), t.TempDir(), "", map[string]string{"gum": gum}, "-g", "menu")
	if err != nil {
		t.Fatalf("menu failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "swoop does not know this project") {
		t.Fatalf("missing unknown-project error:\n%s", out)
	}
	if calls != "" {
		t.Fatalf("skills was called: %q", calls)
	}
}

func TestEngineLibraryIsOneBuiltInSource(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine skill walker")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	writeSkill(t, repo, "tdd", "Test-driven development.")
	writeSkill(t, repo, "review", "Code review.")
	home := filepath.Join(root, "home")
	writeSkill(t, filepath.Join(home, ".claude"), "tidied", "Moved by Tidy.")

	config := filepath.Join(root, "config")
	data := filepath.Join(root, "data")
	lib := filepath.Join(data, "swoop", "library")
	run := func(args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{"engine/swoop-core"}, args...)...)
		cmd.Env = append(os.Environ(),
			"HOME="+home,
			"XDG_CONFIG_HOME="+config,
			"XDG_DATA_HOME="+data,
			"NO_COLOR=1",
			"SWOOP_ASSUME_YES=1",
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	must := func(args ...string) string {
		t.Helper()
		out, err := run(args...)
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
		return out
	}

	must("add", filepath.Join(repo, "skills", "tdd"), filepath.Join(repo, "skills", "review"), "owner/other")
	must("stash")
	// a legacy per-skill line from older versions must collapse into the Library
	sourcesFile := filepath.Join(config, "swoop", "sources")
	f, err := os.OpenFile(sourcesFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(filepath.Join(lib, "tdd") + "\n")
	f.Close()

	if got, want := must("_sources"), lib+"\nowner/other\n"; got != want {
		t.Fatalf("_sources = %q, want %q", got, want)
	}
	if out := must("list"); strings.Count(out, "library") != 1 || !strings.Contains(out, "Library") {
		t.Fatalf("list should show the Library once:\n%s", out)
	}
	skills := must("_skills", lib)
	for _, want := range []string{"tdd\tTest-driven development.", "review\tCode review.", "tidied\tMoved by Tidy."} {
		if !strings.Contains(skills, want) {
			t.Fatalf("_skills on Library missing %q:\n%s", want, skills)
		}
	}
	must("star", lib, "tdd")

	snapshot := func() string {
		var b strings.Builder
		for _, p := range []string{sourcesFile, filepath.Join(config, "swoop", "stars")} {
			c, _ := os.ReadFile(p)
			b.Write(c)
		}
		ents, _ := os.ReadDir(lib)
		for _, e := range ents {
			b.WriteString(e.Name() + "\n")
		}
		return b.String()
	}
	before := snapshot()
	out, err := run("remove", lib, filepath.Join(lib, "tdd"))
	if err == nil || !strings.Contains(out, "cannot be removed") {
		t.Fatalf("remove Library should fail with a message, err=%v\n%s", err, out)
	}
	if after := snapshot(); after != before {
		t.Fatalf("remove Library changed state:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if out, err := run("remove", lib, "owner/other"); err == nil {
		t.Fatalf("mixed remove should still fail:\n%s", out)
	}
	if got, want := must("_sources"), lib+"\n"; got != want {
		t.Fatalf("other source should be removed alongside the refusal, _sources = %q", got)
	}
}

// covers the VRTS-5 review: relative Library paths, legacy stars, nested
// sources under the Library, and remove with no saved sources.
func TestEngineLibraryEdgeCases(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine skill walker")
	}
	script, err := filepath.Abs("engine/swoop-core")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	writeSkill(t, repo, "tdd", "Test-driven development.")
	config := filepath.Join(root, "config")
	data := filepath.Join(root, "data")
	lib := filepath.Join(data, "swoop", "library")
	sourcesFile := filepath.Join(config, "swoop", "sources")
	starsFile := filepath.Join(config, "swoop", "stars")
	run := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{script}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"HOME="+root,
			"XDG_CONFIG_HOME="+config,
			"XDG_DATA_HOME="+data,
			"NO_COLOR=1",
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }

	// a star made before any source was saved survives `remove` of that source
	if out, err := run(root, "star", repo, "tdd"); err != nil {
		t.Fatalf("star: %v\n%s", err, out)
	}
	if out, err := run(root, "remove", repo); err != nil || !strings.Contains(out, "no saved sources") {
		t.Fatalf("remove with no saved sources should be a no-op: %v\n%s", err, out)
	}
	if !strings.Contains(read(starsFile), repo+"\ttdd\t") {
		t.Fatalf("star was pruned:\n%s", read(starsFile))
	}

	// relative Library path is refused like the absolute one
	if out, err := run(root, "add", filepath.Join(repo, "skills", "tdd"), "owner/other"); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	before := read(sourcesFile)
	out, err := run(filepath.Join(data, "swoop"), "remove", "./library")
	if err == nil || !strings.Contains(out, "cannot be removed") {
		t.Fatalf("relative Library remove should be refused: %v\n%s", err, out)
	}
	if read(sourcesFile) != before {
		t.Fatalf("relative Library remove changed sources")
	}
	// a quoted home-relative spelling is refused too (HOME is root here)
	out, err = run(root, "remove", "~/data/swoop/library")
	if err == nil || !strings.Contains(out, "cannot be removed") {
		t.Fatalf("~/ Library remove should be refused: %v\n%s", err, out)
	}
	if read(sourcesFile) != before {
		t.Fatalf("~/ Library remove changed sources")
	}

	// listing stars keeps a last record that has no trailing newline
	if err := os.WriteFile(starsFile, []byte(repo+"\ttdd\tdesc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := run(root, "stars"); err != nil || !strings.Contains(out, repo+"\ttdd\tdesc") {
		t.Fatalf("stars dropped the unterminated record: %v\n%s", err, out)
	}
	if !strings.Contains(read(starsFile), repo+"\ttdd\tdesc") {
		t.Fatalf("stars file lost the unterminated record:\n%s", read(starsFile))
	}

	// a legacy library/<name> star and a Library star are the same star
	legacy := filepath.Join(lib, "tdd")
	if err := os.WriteFile(starsFile, []byte(legacy+"\ttdd\tTest-driven development.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := run(root, "star", lib, "tdd"); err != nil {
		t.Fatalf("star Library: %v\n%s", err, out)
	}
	if got := read(starsFile); strings.Count(got, "\ttdd\t") != 1 || !strings.HasPrefix(got, lib+"\t") {
		t.Fatalf("legacy and Library star should be one Library star:\n%s", got)
	}
	if err := os.WriteFile(starsFile, []byte(legacy+"\ttdd\t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := run(root, "unstar", lib, "tdd"); err != nil {
		t.Fatalf("unstar: %v\n%s", err, out)
	}
	if strings.Contains(read(starsFile), "tdd") {
		t.Fatalf("unstar on the Library should remove the legacy star:\n%s", read(starsFile))
	}

	// a multi-skill folder nested in the Library is an ordinary source
	nested := filepath.Join(lib, "collections", "team")
	writeSkill(t, nested, "review", "Code review.")
	if out, err := run(root, "--no-copy", "add", nested); err != nil {
		t.Fatalf("add nested: %v\n%s", err, out)
	}
	if out, _ := run(root, "_sources"); !strings.Contains(out, nested+"\n") {
		t.Fatalf("nested source should be listed:\n%s", out)
	}
	if out, _ := run(root, "_skills", nested); !strings.Contains(out, "review\t") {
		t.Fatalf("nested source should list its skills:\n%s", out)
	}
	if out, err := run(root, "remove", nested); err != nil {
		t.Fatalf("nested source should be removable: %v\n%s", err, out)
	}
	if strings.Contains(read(sourcesFile), nested) {
		t.Fatalf("nested source still saved:\n%s", read(sourcesFile))
	}
}

// clashEnv runs the engine with no terminal (CombinedOutput pipes stdout), so
// a Name clash is refused rather than asked about.
func clashEnv(t *testing.T) (run func(args ...string) (string, error), root, lib, config string) {
	t.Helper()
	root = t.TempDir()
	config = filepath.Join(root, "config")
	data := filepath.Join(root, "data")
	lib = filepath.Join(data, "swoop", "library")
	run = func(args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{"engine/swoop-core"}, args...)...)
		cmd.Env = append(os.Environ(),
			"HOME="+filepath.Join(root, "home"),
			"XDG_CONFIG_HOME="+config,
			"XDG_DATA_HOME="+data,
			"NO_COLOR=1",
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	return run, root, lib, config
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEngineAddNameClashRefusedWithoutTerminal(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	writeSkill(t, filepath.Join(root, "b"), "notes", "Second.")
	if out, err := run("add", filepath.Join(root, "a", "skills", "notes")); err != nil {
		t.Fatalf("first add failed: %v\n%s", err, out)
	}
	before := readFile(t, filepath.Join(lib, "notes", "SKILL.md"))

	out, err := run("add", filepath.Join(root, "b", "skills", "notes"))
	if err == nil {
		t.Fatalf("clashing add should exit non-zero:\n%s", out)
	}
	if !strings.Contains(out, "Name clash") || !strings.Contains(out, "notes") ||
		!strings.Contains(out, "rename the incoming skill") || !strings.Contains(out, "rename the Library skill") || !strings.Contains(out, "skip") {
		t.Fatalf("refusal should name the skill and the three choices:\n%s", out)
	}
	if got := readFile(t, filepath.Join(lib, "notes", "SKILL.md")); got != before {
		t.Fatalf("Library skill changed:\n%s", got)
	}
}

func TestEngineAddIdenticalIsNoOp(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, root, "notes", "Same.")
	src := filepath.Join(root, "skills", "notes")
	if out, err := run("add", src); err != nil {
		t.Fatalf("first add failed: %v\n%s", err, out)
	}
	out, err := run("add", src)
	if err != nil {
		t.Fatalf("identical re-add should exit 0: %v\n%s", err, out)
	}
	if !strings.Contains(out, "already in the Library") {
		t.Fatalf("expected an already-in-Library note:\n%s", out)
	}
	ents, _ := os.ReadDir(lib)
	if len(ents) != 1 {
		t.Fatalf("Library should hold one skill, has %d", len(ents))
	}
}

func TestEngineResolveRenameIncoming(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	writeSkill(t, filepath.Join(root, "b"), "notes", "Second.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	incoming := filepath.Join(root, "b", "skills", "notes")
	original := readFile(t, filepath.Join(incoming, "SKILL.md"))

	for _, bad := range []string{"", "Bad", "has space", "notes"} {
		if out, err := run("_resolve", "add", "incoming", bad, incoming); err == nil {
			t.Fatalf("new name %q should be rejected:\n%s", bad, out)
		}
	}
	if out, err := run("_resolve", "add", "incoming", "notes-b", incoming); err != nil {
		t.Fatalf("rename incoming failed: %v\n%s", err, out)
	}
	if !strings.Contains(readFile(t, filepath.Join(lib, "notes", "SKILL.md")), "First.") {
		t.Fatal("old Library skill should stay under its name")
	}
	got := readFile(t, filepath.Join(lib, "notes-b", "SKILL.md"))
	if !strings.Contains(got, "name: notes-b\n") || !strings.Contains(got, "Second.") {
		t.Fatalf("incoming skill not renamed in the Library:\n%s", got)
	}
	if readFile(t, filepath.Join(incoming, "SKILL.md")) != original {
		t.Fatal("the person's original folder must not change")
	}
}

func TestEngineResolveRenameLibrarySkillMovesStar(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine skill walker (star)")
	}
	run, root, lib, config := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	writeSkill(t, filepath.Join(root, "b"), "notes", "Second.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	if out, err := run("star", lib, "notes"); err != nil {
		t.Fatalf("star failed: %v\n%s", err, out)
	}
	if out, err := run("_resolve", "add", "library", "notes-old", filepath.Join(root, "b", "skills", "notes")); err != nil {
		t.Fatalf("rename Library skill failed: %v\n%s", err, out)
	}
	old := readFile(t, filepath.Join(lib, "notes-old", "SKILL.md"))
	if !strings.Contains(old, "name: notes-old\n") || !strings.Contains(old, "First.") {
		t.Fatalf("Library skill not renamed:\n%s", old)
	}
	if !strings.Contains(readFile(t, filepath.Join(lib, "notes", "SKILL.md")), "Second.") {
		t.Fatal("incoming skill should take the original name")
	}
	stars := readFile(t, filepath.Join(config, "swoop", "stars"))
	if !strings.Contains(stars, "\tnotes-old\t") || strings.Contains(stars, "\tnotes\t") {
		t.Fatalf("star should follow the renamed skill:\n%s", stars)
	}
}

func TestEngineStashNameClashLeftInPlace(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "Library copy.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	home := filepath.Join(root, "home")
	writeSkill(t, filepath.Join(home, ".claude"), "notes", "Global copy.")
	writeSkill(t, filepath.Join(home, ".claude"), "fresh", "No clash.")

	dry, err := run("stash", "--dry-run")
	if err != nil || !strings.Contains(dry, "clash") {
		t.Fatalf("dry run should list the clash: %v\n%s", err, dry)
	}
	if _, err := os.Stat(filepath.Join(lib, "fresh")); err == nil {
		t.Fatal("dry run moved a skill")
	}

	// no SWOOP_ASSUME_YES, stdin at EOF, and no controlling terminal (new session)
	cmd := exec.Command("bash", "engine/swoop-core", "stash")
	cmd.Env = append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(root, "config"),
		"XDG_DATA_HOME="+filepath.Join(root, "data"), "NO_COLOR=1")
	cmd.Env = withoutEnv(cmd.Env, "SWOOP_ASSUME_YES")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	b, err := cmd.CombinedOutput()
	out := string(b)
	if err != nil {
		t.Fatalf("stash failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(lib, "fresh", "SKILL.md")); err != nil {
		t.Fatalf("non-clashing skill should move:\n%s", out)
	}
	if !strings.Contains(readFile(t, filepath.Join(home, ".claude", "skills", "notes", "SKILL.md")), "Global copy.") {
		t.Fatal("clashing skill should stay in the global folder")
	}
	if !strings.Contains(readFile(t, filepath.Join(lib, "notes", "SKILL.md")), "Library copy.") {
		t.Fatal("Library skill changed")
	}
	if !strings.Contains(out, "Name clash: notes") || !strings.Contains(out, "not moved 1") {
		t.Fatalf("summary should name the clash and count it as not moved:\n%s", out)
	}
	clashes, err := run("_clashes", "stash")
	if err != nil || clashes != filepath.Join(home, ".claude", "skills", "notes")+"\tnotes\n" {
		t.Fatalf("_clashes stash = %q, %v", clashes, err)
	}
}

func withoutEnv(env []string, key string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, key+"=") {
			out = append(out, kv)
		}
	}
	return out
}

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEngineRenameLibraryStarUsesManifestName(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine skill walker (star)")
	}
	run, root, lib, config := clashEnv(t)
	writeManifest(t, filepath.Join(root, "a", "notes"), "---\nname: meeting-notes  \ndescription: First.\n---\n") // trailing spaces, unquoted
	writeManifest(t, filepath.Join(root, "b", "notes"), "---\nname: notes\ndescription: Second.\n---\n")
	run("add", filepath.Join(root, "a", "notes"))
	if out, err := run("star", lib, "meeting-notes"); err != nil {
		t.Fatalf("star failed: %v\n%s", err, out)
	}
	if out, err := run("_resolve", "add", "library", "notes-old", filepath.Join(root, "b", "notes")); err != nil {
		t.Fatalf("rename failed: %v\n%s", err, out)
	}
	stars := readFile(t, filepath.Join(config, "swoop", "stars"))
	if !strings.Contains(stars, "\tnotes-old\t") || strings.Contains(stars, "meeting-notes") {
		t.Fatalf("star should follow the renamed skill:\n%s", stars)
	}
}

func TestEngineRenameIncomingSymlinkedManifestKeepsOriginal(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	real := filepath.Join(root, "real-SKILL.md")
	original := "---\nname: notes\ndescription: Linked.\n---\n"
	if err := os.WriteFile(real, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	incoming := filepath.Join(root, "b", "notes")
	os.MkdirAll(incoming, 0o755)
	if err := os.Symlink(real, filepath.Join(incoming, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if out, err := run("_resolve", "add", "incoming", "notes-b", incoming); err != nil {
		t.Fatalf("rename failed: %v\n%s", err, out)
	}
	if got := readFile(t, real); got != original {
		t.Fatalf("original manifest changed:\n%s", got)
	}
	copyPath := filepath.Join(lib, "notes-b", "SKILL.md")
	if fi, err := os.Lstat(copyPath); err != nil || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("Library manifest should be a regular file: %v", err)
	}
	if got := readFile(t, copyPath); !strings.Contains(got, "name: notes-b\n") || !strings.Contains(got, "Linked.") {
		t.Fatalf("Library copy not renamed:\n%s", got)
	}
}

func TestEngineRenameKeepsCRLFFrontmatter(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	incoming := filepath.Join(root, "b", "notes")
	writeManifest(t, incoming, "---\r\nname: notes\r\ndescription: Keep me.\r\nauthor: someone\r\n---\r\nbody\r\n")
	if out, err := run("_resolve", "add", "incoming", "notes-crlf", incoming); err != nil {
		t.Fatalf("rename failed: %v\n%s", err, out)
	}
	want := "---\r\nname: notes-crlf\r\ndescription: Keep me.\r\nauthor: someone\r\n---\r\nbody\r\n"
	if got := readFile(t, filepath.Join(lib, "notes-crlf", "SKILL.md")); got != want {
		t.Fatalf("CRLF manifest = %q, want %q", got, want)
	}
}

func TestEngineResolveTakesNameLiterally(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	writeSkill(t, filepath.Join(root, "b"), "notes", "Second.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	if out, err := run("_resolve", "add", "incoming", "--help", filepath.Join(root, "b", "skills", "notes")); err != nil {
		t.Fatalf("rename failed: %v\n%s", err, out)
	}
	if got := readFile(t, filepath.Join(lib, "--help", "SKILL.md")); !strings.Contains(got, "name: --help\n") || !strings.Contains(got, "Second.") {
		t.Fatalf("--help rename not on disk:\n%s", got)
	}
}

func TestEngineRenameIncomingRelativeManifestLink(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	original := "---\nname: notes\ndescription: Relative.\n---\n"
	if err := os.WriteFile(filepath.Join(root, "manifest.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	incoming := filepath.Join(root, "notes")
	os.MkdirAll(incoming, 0o755)
	if err := os.Symlink("../manifest.md", filepath.Join(incoming, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if out, err := run("_resolve", "add", "incoming", "notes-rel", incoming); err != nil {
		t.Fatalf("rename failed: %v\n%s", err, out)
	}
	if got := readFile(t, filepath.Join(root, "manifest.md")); got != original {
		t.Fatalf("original manifest changed:\n%s", got)
	}
	got := readFile(t, filepath.Join(lib, "notes-rel", "SKILL.md"))
	if !strings.Contains(got, "name: notes-rel\n") || !strings.Contains(got, "Relative.") {
		t.Fatalf("Library copy not renamed:\n%s", got)
	}
}

func TestEngineRenameIncomingFailureLeavesNoFolder(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	writeSkill(t, filepath.Join(root, "b"), "notes", "Second.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	incoming := filepath.Join(root, "b", "skills", "notes")
	manifest := filepath.Join(incoming, "SKILL.md")
	if err := os.Chmod(manifest, 0o000); err != nil { // the rewrite cannot read it
		t.Fatal(err)
	}
	defer os.Chmod(manifest, 0o644)
	if os.Getuid() == 0 {
		t.Skip("root can read a 0000 file")
	}
	if out, err := run("_resolve", "add", "incoming", "notes-b", incoming); err == nil {
		t.Fatalf("unreadable manifest should fail:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(lib, "notes-b")); !os.IsNotExist(err) {
		t.Fatalf("a failed rename must not leave the Library folder: %v", err)
	}
}

func TestEngineRenameIncomingSymlinkedFolderKeepsOriginal(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "a"), "notes", "First.")
	writeSkill(t, filepath.Join(root, "real"), "notes", "Linked folder.")
	run("add", filepath.Join(root, "a", "skills", "notes"))
	realManifest := filepath.Join(root, "real", "skills", "notes", "SKILL.md")
	original := readFile(t, realManifest)
	incoming := filepath.Join(root, "link", "notes")
	os.MkdirAll(filepath.Dir(incoming), 0o755)
	if err := os.Symlink(filepath.Join(root, "real", "skills", "notes"), incoming); err != nil {
		t.Fatal(err)
	}
	if out, err := run("_resolve", "add", "incoming", "archived", incoming); err != nil {
		t.Fatalf("rename failed: %v\n%s", err, out)
	}
	if got := readFile(t, realManifest); got != original {
		t.Fatalf("original manifest changed:\n%s", got)
	}
	fi, err := os.Lstat(filepath.Join(lib, "archived"))
	if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("Library entry should be a real directory: %v %v", fi, err)
	}
	if got := readFile(t, filepath.Join(lib, "archived", "SKILL.md")); !strings.Contains(got, "name: archived\n") {
		t.Fatalf("Library copy not renamed:\n%s", got)
	}
}

func TestEngineAddSymlinkedFolderCopiesRealDirectory(t *testing.T) {
	run, root, lib, _ := clashEnv(t)
	writeSkill(t, filepath.Join(root, "real"), "notes", "Linked folder.")
	incoming := filepath.Join(root, "link", "notes")
	os.MkdirAll(filepath.Dir(incoming), 0o755)
	if err := os.Symlink(filepath.Join(root, "real", "skills", "notes"), incoming); err != nil {
		t.Fatal(err)
	}
	if out, err := run("add", incoming); err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}
	if fi, err := os.Lstat(filepath.Join(lib, "notes")); err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		t.Fatalf("Library entry should be a real directory: %v %v", fi, err)
	}
}

// VRTS-7: removing a Source with Stars warns, lists them, and asks first.
// Each case runs with a known prompt backend: "plain" hides gum and fzf so
// the [y/N] fallback reads stdin; "gum" puts a stub gum first on PATH.
func TestEngineRemoveWarnsAboutStars(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required by the engine skill walker")
	}
	script, err := filepath.Abs("engine/swoop-core")
	if err != nil {
		t.Fatal(err)
	}
	root := realTempDir(t)
	starred := filepath.Join(root, "starred")
	writeSkill(t, starred, "tdd", "Test-driven development.")
	other := filepath.Join(root, "other")
	writeSkill(t, other, "review", "Code review.")
	plain := filepath.Join(root, "plain")
	writeSkill(t, plain, "lint", "Lint.")
	config := filepath.Join(root, "config")
	data := filepath.Join(root, "data")
	lib := filepath.Join(data, "swoop", "library")
	sourcesFile := filepath.Join(config, "swoop", "sources")
	starsFile := filepath.Join(config, "swoop", "stars")

	// PATH without any gum or fzf from the host: a directory holding either
	// is replaced by a mirror of its other entries, so tools that share it
	// (node, say) stay on PATH
	var keep []string
	for i, dir := range filepath.SplitList(os.Getenv("PATH")) {
		_, gumErr := os.Stat(filepath.Join(dir, "gum"))
		_, fzfErr := os.Stat(filepath.Join(dir, "fzf"))
		if gumErr != nil && fzfErr != nil {
			keep = append(keep, dir)
			continue
		}
		mirror := filepath.Join(root, "pathmirror", strconv.Itoa(i))
		if err := os.MkdirAll(mirror, 0o755); err != nil {
			t.Fatal(err)
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range ents {
			if e.Name() == "gum" || e.Name() == "fzf" {
				continue
			}
			if err := os.Symlink(filepath.Join(dir, e.Name()), filepath.Join(mirror, e.Name())); err != nil {
				t.Fatal(err)
			}
		}
		keep = append(keep, mirror)
	}
	plainPath := strings.Join(keep, string(os.PathListSeparator))
	// stub gum: confirm answers from stdin and applies its default on empty
	// input like real gum (yes unless --default=false); choose picks the
	// line equal to $PICK
	gumBin := filepath.Join(root, "gumbin")
	if err := os.MkdirAll(gumBin, 0o755); err != nil {
		t.Fatal(err)
	}
	gum := `#!/bin/sh
case "$1" in
confirm)
  def=0; for a in "$@"; do [ "$a" = "--default=false" ] && def=1; done
  IFS= read -r ans || ans=""
  case "$ans" in y) exit 0 ;; n) exit 1 ;; "") exit $def ;; esac; exit 1 ;;
choose) grep -xF "$PICK"; exit 0 ;;
esac
exit 2
`
	if err := os.WriteFile(filepath.Join(gumBin, "gum"), []byte(gum), 0o755); err != nil {
		t.Fatal(err)
	}

	// stdin is the answer; Setsid drops the controlling terminal so the
	// engine reads it from stdin instead of /dev/tty
	run := func(backend, stdin string, extraEnv []string, args ...string) (string, error) {
		cmd := exec.Command("bash", append([]string{script}, args...)...)
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(stdin)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		path := plainPath
		if backend == "gum" {
			path = gumBin + string(os.PathListSeparator) + plainPath
		}
		cmd.Env = append(os.Environ(),
			"PATH="+path,
			"HOME="+root,
			"XDG_CONFIG_HOME="+config,
			"XDG_DATA_HOME="+data,
			"NO_COLOR=1",
			"SWOOP_ASSUME_YES=", // the caller's value must not answer for us
		)
		cmd.Env = append(cmd.Env, extraEnv...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }
	must := func(args ...string) {
		t.Helper()
		if out, err := run("plain", "", nil, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	must("--no-copy", "add", starred, other, plain)
	must("star", starred, "tdd")
	must("star", other, "review")

	// a Source with no Stars is removed without a question
	out, err := run("plain", "", nil, "remove", plain)
	if err != nil || strings.Contains(out, "warning") || !strings.Contains(out, "removed "+plain) {
		t.Fatalf("plain remove should not ask: %v\n%s", err, out)
	}

	sources, stars := read(sourcesFile), read(starsFile)
	unchanged := func(label, out string) {
		t.Helper()
		if !strings.Contains(out, "nothing removed.") {
			t.Fatalf("%s: missing 'nothing removed.':\n%s", label, out)
		}
		if read(sourcesFile) != sources || read(starsFile) != stars {
			t.Fatalf("%s changed state:\nsources:\n%s\nstars:\n%s", label, read(sourcesFile), read(starsFile))
		}
	}

	// fallback prompt: "no" and empty input remove nothing; one question for both
	for _, answer := range []string{"n\n", ""} {
		out, err = run("plain", answer, nil, "remove", starred, other)
		if err != nil {
			t.Fatalf("declined remove: %v\n%s", err, out)
		}
		for _, want := range []string{"tdd  (source: " + starred + ")", "review  (source: " + other + ")"} {
			if !strings.Contains(out, want) {
				t.Fatalf("answer %q: output missing %q:\n%s", answer, want, out)
			}
		}
		if strings.Count(out, "[y/N]") != 1 {
			t.Fatalf("expected one question for both sources:\n%s", out)
		}
		unchanged("fallback answer "+strconv.Quote(answer), out)
	}

	// gum: Enter (empty input) must mean no, not gum's default yes
	out, err = run("gum", "\n", nil, "remove", starred)
	if err != nil {
		t.Fatalf("gum enter: %v\n%s", err, out)
	}
	unchanged("gum enter", out)

	// the Library is refused before the prompt; declining keeps the error exit
	out, err = run("plain", "n\n", nil, "remove", lib, starred)
	if err == nil || !strings.Contains(out, "cannot be removed") {
		t.Fatalf("Library + decline should fail with the Library error: %v\n%s", err, out)
	}
	unchanged("Library + decline", out)

	// the no-argument picker path warns the same way
	out, err = run("gum", "n\n", []string{"PICK=" + starred}, "remove")
	if err != nil || !strings.Contains(out, "tdd  (source: "+starred+")") {
		t.Fatalf("picker remove should warn: %v\n%s", err, out)
	}
	unchanged("picker decline", out)

	// "yes" removes the Source and its Stars; other Stars stay
	if out, err = run("plain", "y\n", nil, "remove", starred); err != nil || !strings.Contains(out, "removed "+starred) {
		t.Fatalf("confirmed remove: %v\n%s", err, out)
	}
	if strings.Contains(read(sourcesFile), starred) || strings.Contains(read(starsFile), starred) {
		t.Fatalf("source or star left behind:\nsources:\n%s\nstars:\n%s", read(sourcesFile), read(starsFile))
	}
	if !strings.Contains(read(starsFile), other+"\treview\t") {
		t.Fatalf("star of an unmarked source was deleted:\n%s", read(starsFile))
	}

	// SWOOP_ASSUME_YES skips the question
	if out, err = run("plain", "", []string{"SWOOP_ASSUME_YES=1"}, "remove", other); err != nil || strings.Contains(out, "[y/N]") {
		t.Fatalf("assume-yes remove should not ask: %v\n%s", err, out)
	}
	if strings.Contains(read(sourcesFile), other) || strings.Contains(read(starsFile), other) {
		t.Fatalf("assume-yes left state behind:\nsources:\n%s\nstars:\n%s", read(sourcesFile), read(starsFile))
	}
}
