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

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\a]*(\a|\x1b\\)|[\x07]`)

func stripANSI(s string) string {
	return collapseTerminalControls(ansiRE.ReplaceAllString(s, ""))
}

func collapseTerminalControls(s string) string {
	var out strings.Builder
	var line []rune
	flush := func() {
		out.WriteString(string(line))
		line = line[:0]
	}
	for _, r := range s {
		switch r {
		case '\r':
			line = line[:0]
		case '\n':
			flush()
			out.WriteByte('\n')
		case '\b':
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
		default:
			line = append(line, r)
		}
	}
	flush()
	return out.String()
}

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

// ---- friendly source aliases (TUI display only) -------------------------
// stored as "url<TAB>alias" lines in ~/.config/swoop/aliases.
func aliasesPath() string { return filepath.Join(configDir(), "aliases") }

func loadAliases() map[string]string {
	m := map[string]string{}
	for _, ln := range readLines(aliasesPath()) {
		f := strings.SplitN(ln, "\t", 2)
		if len(f) == 2 && strings.TrimSpace(f[1]) != "" {
			m[f[0]] = f[1]
		}
	}
	return m
}

// saveAlias sets (or, with an empty name, clears) the alias for a source URL.
func saveAlias(url, name string) error {
	al := loadAliases()
	name = strings.TrimSpace(name)
	if name == "" {
		delete(al, url)
	} else {
		al[url] = name
	}
	var b strings.Builder
	for u, n := range al {
		b.WriteString(u)
		b.WriteByte('\t')
		b.WriteString(n)
		b.WriteByte('\n')
	}
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	tmp := aliasesPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, aliasesPath())
}

// sourceItems builds the install/source list, showing an alias when one is set
// (with the real repo URL as the dim subtitle). item.id is always the real URL.
func sourceItems() []item {
	al := loadAliases()
	srcs := loadSources()
	items := make([]item, len(srcs))
	for i, s := range srcs {
		it := item{id: s, title: s}
		if name, ok := al[s]; ok {
			it.title = name
			it.desc = s
		}
		items[i] = it
	}
	return items
}

func starredItems() []item {
	stars := loadStars()
	items := make([]item, len(stars))
	for i, st := range stars {
		it := st
		if it.desc == "" {
			it.desc = short(it.source)
		} else {
			it.desc = short(it.source) + "  ·  " + it.desc
		}
		items[i] = it
	}
	return items
}

// ---- plugin marketplaces (engine-maintained config) ----------------------
// stored as "source<TAB>claude_name<TAB>codex_name" in ~/.config/swoop/marketplaces.
type marketplace struct {
	source string
	claude string // marketplace name from .claude-plugin/marketplace.json ("" = absent)
	codex  string // marketplace name from .agents/plugins/marketplace.json ("" = absent)
}

func loadMarketplaces() []marketplace {
	var out []marketplace
	for _, ln := range readLines(filepath.Join(configDir(), "marketplaces")) {
		f := strings.SplitN(ln, "\t", 3)
		m := marketplace{source: strings.TrimSpace(f[0])}
		if len(f) > 1 {
			m.claude = strings.TrimSpace(f[1])
		}
		if len(f) > 2 {
			m.codex = strings.TrimSpace(f[2])
		}
		if m.source != "" {
			out = append(out, m)
		}
	}
	return out
}

// marketItems builds the marketplace list, showing an alias when one is set
// (same aliases file as sources; item.id is always the real source).
func marketItems() []item {
	al := loadAliases()
	mkts := loadMarketplaces()
	items := make([]item, len(mkts))
	for i, mk := range mkts {
		desc := ""
		switch {
		case mk.claude != "" && mk.codex != "":
			desc = "claude + codex"
		case mk.claude != "":
			desc = "claude only"
		case mk.codex != "":
			desc = "codex only"
		}
		it := item{id: mk.source, title: mk.source, desc: desc}
		if name, ok := al[mk.source]; ok {
			it.title = name
			if desc == "" {
				it.desc = mk.source
			} else {
				it.desc = mk.source + "  ·  " + desc
			}
		}
		items[i] = it
	}
	return items
}

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
	return dedupeItems(parseTabbed(out, 2)), nil
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
	return dedupeItems(items), nil
}

// listPlugins asks the engine for a marketplace's plugins, annotating plugins
// that only exist for one of the two plugin-capable agents.
func listPlugins(src string) ([]item, error) {
	out, err := core("_plugins", src)
	if err != nil {
		return nil, err
	}
	_, _, items := parsePlugins(out)
	agents := loadAgents()
	wantClaude := strings.Contains(agents, "claude-code")
	wantCodex := strings.Contains(agents, "codex")
	for i, it := range items {
		if !wantClaude || !wantCodex {
			continue
		}
		if hasFlag(it.flags, "claude") && !hasFlag(it.flags, "codex") {
			items[i].desc = suffixNote(it.desc, "claude only")
		} else if hasFlag(it.flags, "codex") && !hasFlag(it.flags, "claude") {
			items[i].desc = suffixNote(it.desc, "codex only")
		}
	}
	return items, nil
}

// parsePlugins parses `_plugins` output: an "@marketplace<TAB>claude<TAB>codex"
// header, then "name<TAB>desc<TAB>flags<TAB>relpath" plugin lines (the relpath
// column is engine-internal vendoring data and ignored here).
func parsePlugins(out string) (claudeName, codexName string, items []item) {
	for _, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		f := strings.SplitN(ln, "\t", 4)
		if f[0] == "@marketplace" {
			if len(f) > 1 {
				claudeName = f[1]
			}
			if len(f) > 2 {
				codexName = f[2]
			}
			continue
		}
		it := item{id: f[0], title: f[0]}
		if len(f) > 1 {
			it.desc = f[1]
		}
		if len(f) > 2 {
			it.flags = f[2]
		}
		items = append(items, it)
	}
	return claudeName, codexName, dedupeItems(items)
}

// listInstalledPlugins merges both agents' installed plugins via the engine
// ("name@marketplace<TAB>agents<TAB>desc"; warn lines have no tab and are skipped).
func listInstalledPlugins() ([]item, error) {
	out, err := core("_plugins_installed")
	if err != nil {
		return nil, err
	}
	var items []item
	for _, ln := range strings.Split(out, "\n") {
		if !strings.Contains(ln, "\t") {
			continue
		}
		f := strings.SplitN(ln, "\t", 3)
		it := item{id: f[0], title: f[0], desc: f[1]}
		if len(f) > 2 && strings.TrimSpace(f[2]) != "" {
			it.desc = f[1] + "  ·  " + f[2]
		}
		items = append(items, it)
	}
	return dedupeItems(items), nil
}

// codexHooksState reports the codex features.hooks flag: "on", "off" or "n/a"
// (anything unreadable maps to "n/a" so installs are never blocked).
func codexHooksState() string {
	out, err := core("_codex_hooks")
	if err != nil {
		return "n/a"
	}
	state := ""
	for _, ln := range strings.Split(out, "\n") {
		if s := strings.TrimSpace(ln); s != "" {
			state = s
		}
	}
	if state != "on" && state != "off" {
		return "n/a"
	}
	return state
}

func hasFlag(flags, f string) bool {
	for _, x := range strings.Split(flags, ",") {
		if x == f {
			return true
		}
	}
	return false
}

func suffixNote(desc, note string) string {
	if desc == "" {
		return "· " + note
	}
	return desc + " · " + note
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

func dedupeItems(items []item) []item {
	seen := map[string]bool{}
	out := items[:0]
	for _, it := range items {
		key := it.source + "\t" + it.id
		if it.source == "" {
			key = it.id
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, it)
	}
	return out
}
