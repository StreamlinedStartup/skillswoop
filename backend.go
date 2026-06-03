package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// corePath returns the bash engine to run. An explicit SWOOP_CORE wins (handy for
// development); otherwise we use the engine embedded in this binary.
func corePath() string {
	if p := os.Getenv("SWOOP_CORE"); p != "" {
		return p
	}
	if p, err := extractedCore(); err == nil {
		return p
	}
	// last-ditch fallback: a sibling file next to the binary
	if exe, err := os.Executable(); err == nil {
		sib := filepath.Join(filepath.Dir(exe), "swoop-core")
		if _, err := os.Stat(sib); err == nil {
			return sib
		}
	}
	return "swoop-core"
}

// core runs the engine and returns combined stdout+stderr.
func core(args ...string) (string, error) {
	cmd := exec.Command(corePath(), args...)
	// NO_COLOR keeps the result pane clean; ASSUME_YES stops the engine launching
	// its own gum prompts (the TUI does its own confirmations).
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "SWOOP_ASSUME_YES=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\a]*(\a|\x1b\\)|[\r\x07]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// ---- config readers (plain-text files the engine maintains) -------------
func configDir() string {
	if p := os.Getenv("XDG_CONFIG_HOME"); p != "" {
		return filepath.Join(p, "swoop")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "swoop")
}

func readLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func loadSources() []string  { return readLines(filepath.Join(configDir(), "sources")) }
func loadProjects() []string { return readLines(filepath.Join(configDir(), "projects")) }

func loadAgents() string {
	a := readLines(filepath.Join(configDir(), "agents"))
	if len(a) == 0 {
		return "claude-code codex"
	}
	return strings.Join(a, " ")
}

// listSkills asks the engine for "name<TAB>desc" lines for a source.
func listSkills(src string) ([]item, error) {
	out, err := core("_skills", src)
	if err != nil {
		return nil, err
	}
	return parseTabbed(out, 2), nil
}

// searchSkills hits skills.sh via the engine and returns repos to remember.
func searchSkills(q string) ([]item, error) {
	out, err := core("_search", q)
	if err != nil {
		return nil, err
	}
	var items []item
	for _, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		f := strings.Split(ln, "\t")
		src := f[0]
		name := ""
		inst := ""
		if len(f) > 1 {
			name = f[1]
		}
		if len(f) > 2 {
			inst = f[2]
		}
		desc := name
		if inst != "" {
			desc = name + "  ·  " + inst + " installs"
		}
		items = append(items, item{id: src, title: src, desc: desc})
	}
	return items, nil
}

// parseTabbed turns "a<TAB>b" lines into items (id=title=col1, desc=col2).
func parseTabbed(out string, _ int) []item {
	var items []item
	for _, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		f := strings.SplitN(ln, "\t", 2)
		it := item{id: f[0], title: f[0]}
		if len(f) > 1 {
			it.desc = f[1]
		}
		items = append(items, it)
	}
	return items
}
