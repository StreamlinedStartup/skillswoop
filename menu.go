package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func menuDomains() []menuDomain {
	return []menuDomain{
		{label: "SKILLS", entries: []menuEntry{
			{"◢◤", "Install skills", "pick a source, then swoop specific skills into this folder", actInstall},
			{"★", "Starred skills", "install skills you've starred for quick reuse", actStarred},
			{"⟳", "Update this folder", "pull latest from GitHub for skills in the current dir", actUpdateHere},
			{"⟳⟳", "Update every folder", "refresh every folder you've installed into", actUpdateAll},
			{"⌖", "Browse skills.sh", "search the directory and remember new sources", actBrowse},
			{"⚙", "Sources…", "add or remove saved skill sources", actSourceActions},
		}},
		{label: "PLUGINS", entries: []menuEntry{
			{"⬢", "Install plugins", "pick a marketplace, then install plugins (hooks auto-wire)", actInstallPlugins},
			{"⊖", "Remove plugins", "uninstall plugins from claude + codex", actRemovePlugins},
			{"⊕", "Add a marketplace", "register a plugin marketplace repo", actAddMarketplace},
			{"⚙", "Marketplaces…", "open, rename, remove, or update saved marketplaces", actInstallPlugins},
		}},
		{label: "SETTINGS", entries: []menuEntry{
			{"⚙", "Default agents", "choose which agents to target", actAgents},
			{"⤓", "Tidy global skills", "move global Claude and Codex skills into the library", actTidy},
		}},
	}
}

func sourceActionEntries() []menuEntry {
	return []menuEntry{
		{"＋", "Add a source", "owner/repo · git URL · local path", actAdd},
		{"✕", "Remove sources", "forget one or more saved sources", actRemove},
	}
}

func menuItems(entries []menuEntry) []item {
	items := make([]item, len(entries))
	for i, entry := range entries {
		items[i] = item{
			id:    entry.title,
			title: padRight(entry.icon, 2) + "  " + entry.title,
		}
	}
	return items
}

func (m *model) setDomain(next int) {
	if len(m.domains) == 0 {
		return
	}
	if m.menu != nil {
		m.domainCursors[m.domain] = m.menu.cursor
	}
	next = (next%len(m.domains) + len(m.domains)) % len(m.domains)
	m.domain = next
	m.menu = newPicker(menuItems(m.domains[next].entries), false)
	m.menu.cursor = m.domainCursors[next]
	m.menu.clampWindow()
	m.menu.setSize(m.innerW, m.innerH-3)
}

func actSourceActions(m *model) (tea.Model, tea.Cmd) {
	m.enterPicker(newPicker(menuItems(sourceActionEntries()), false))
	m.screen = scSourceActions
	return m, nil
}

func actStarred(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.busyTitle = "loading starred skills"
	m.screen = scRunning
	return m, loadStarredCmd()
}

func (m *model) enterPicker(p *picker) {
	m.pick = p
	listH := m.innerH - 3
	if listH < 1 {
		listH = 1
	}
	m.pick.setSize(m.innerW, listH)
}

func actInstall(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.busyTitle = "loading saved sources"
	m.screen = scRunning // hold here until sourcesMsg builds the picker
	return m, loadSourcesCmd()
}

func actUpdateHere(m *model) (tea.Model, tea.Cmd) {
	m.busyTitle = "updating skills in this folder"
	m.screen = scRunning
	if m.global {
		return m, opCmd("update (global)", "-g", "update")
	}
	return m, opCmd("update (this folder)", "update")
}

func actUpdateAll(m *model) (tea.Model, tea.Cmd) {
	m.busyTitle = "updating every known folder"
	m.screen = scRunning
	return m, opCmd("update --all", "update", "--all")
}

func actBrowse(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.screen = scBrowseInput
	m.input.Placeholder = "keyword (blank = top results)"
	m.input.SetValue("")
	m.input.Focus()
	return m, nil
}

func actAdd(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.screen = scAdd
	m.addMarketplace = false
	m.input.Placeholder = "owner/repo | https://… | ~/path/to/skill"
	m.input.SetValue("")
	m.input.Focus()
	return m, nil
}

func actInstallPlugins(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.busyTitle = "loading marketplaces"
	m.screen = scRunning // hold here until marketsMsg builds the picker
	return m, loadMarketsCmd()
}

func actAddMarketplace(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.screen = scAdd
	m.addMarketplace = true
	m.input.Placeholder = "owner/repo | https://… | ~/path/to/marketplace"
	m.input.SetValue("")
	m.input.Focus()
	return m, nil
}

func actRemovePlugins(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.busyTitle = "reading installed plugins"
	m.screen = scRunning
	return m, loadInstalledPluginsCmd()
}

func actRemove(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.screen = scRemove
	srcs := loadSources()
	items := make([]item, len(srcs))
	for i, s := range srcs {
		items[i] = item{id: s, title: s}
	}
	m.enterPicker(newPicker(items, true))
	return m, nil
}

func actAgents(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.screen = scAgents
	m.input.Placeholder = "space-separated, e.g. claude-code codex cursor"
	m.input.SetValue(m.agents)
	m.input.Focus()
	m.input.CursorEnd()
	return m, nil
}

func actTidy(m *model) (tea.Model, tea.Cmd) {
	m.prev = scMenu
	m.screen = scConfirm
	m.confirmMsg = "Move existing global ~/.claude & ~/.codex skills into the library?"
	m.confirmCmd = func(mm *model) tea.Cmd {
		mm.busyTitle = "tidying global skills"
		mm.screen = scRunning
		return opCmd("tidy global skills", "stash")
	}
	return m, nil
}

// installSelected builds the engine call from the marked skills.
func (m *model) installSelected() tea.Cmd {
	sel := m.pick.selected()
	if len(sel) == 0 {
		return flashFor("nothing marked — press SPACE to mark, then ENTER", 0)
	}
	if m.screen == scStarred {
		m.busyTitle = "installing " + itoa(len(sel)) + " starred skill(s)"
		m.screen = scRunning
		return installGroupedCmd("install starred", m.global, sel)
	}
	args := []string{}
	if m.global {
		args = append(args, "-g")
	}
	args = append(args, "use", m.curSource, "--")
	for _, it := range sel {
		args = append(args, "--skill", it.id)
	}
	args = append(args, "-y")
	m.busyTitle = "installing " + itoa(len(sel)) + " skill(s) from " + short(m.curSource)
	m.screen = scRunning
	return opCmd("install", args...)
}

// installSelectedPlugins builds the plugin-install engine call from the marked
// plugins. When a marked plugin bundles hooks and codex is a target, the args
// are parked in pendingInstall and the codex features.hooks state is checked
// first so the user can confirm enabling it (codexHooksMsg finishes the job).
func (m *model) installSelectedPlugins() tea.Cmd {
	sel := m.pick.selected()
	if len(sel) == 0 {
		return flashFor("nothing marked — press SPACE to mark, then ENTER", 0)
	}
	args := []string{}
	if m.global {
		args = append(args, "-g")
	}
	args = append(args, "plugin", "install", m.curMarket)
	needHooks := false
	for _, it := range sel {
		args = append(args, it.id)
		if hasFlag(it.flags, "hooks") {
			needHooks = true
		}
	}
	if needHooks && strings.Contains(m.agents, "codex") {
		m.pendingInstall = args
		m.busyTitle = "checking codex hooks"
		m.screen = scRunning
		return codexHooksCmd()
	}
	m.busyTitle = "installing " + itoa(len(sel)) + " plugin(s)"
	m.screen = scRunning
	return opCmd("install plugins", args...)
}

func installGroupedCmd(title string, global bool, items []item) tea.Cmd {
	return func() tea.Msg {
		groups := map[string][]string{}
		var order []string
		for _, it := range items {
			if it.source == "" || it.id == "" {
				continue
			}
			if _, ok := groups[it.source]; !ok {
				order = append(order, it.source)
			}
			groups[it.source] = append(groups[it.source], it.id)
		}
		var out strings.Builder
		var firstErr error
		for _, src := range order {
			args := []string{}
			if global {
				args = append(args, "-g")
			}
			args = append(args, "use", src, "--")
			for _, skill := range groups[src] {
				args = append(args, "--skill", skill)
			}
			args = append(args, "-y")
			chunk, err := core(args...)
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(stripANSI(chunk))
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return opDoneMsg{title: title, output: out.String(), err: firstErr}
	}
}

func short(src string) string {
	s := strings.TrimSuffix(src, ".git")
	if i := strings.LastIndex(s, "github.com"); i >= 0 {
		s = strings.TrimLeft(s[i+len("github.com"):], ":/")
	}
	return s
}
