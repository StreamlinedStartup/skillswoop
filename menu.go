package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func menuDomains() []menuDomain {
	return []menuDomain{
		{label: "SKILLS", entries: []menuEntry{
			{"◢◤", "Install skills", "pick a source and skills, then choose where to install", actInstall},
			{"★", "Starred skills", "install skills you've starred for quick reuse", actStarred},
			{"⟳", "Update this folder", "pull latest from GitHub for skills in the current dir", actUpdateHere},
			{"⟳⟳", "Update every folder", "refresh every folder you've installed into", actUpdateAll},
			{"⌖", "Browse skills.sh", "search the directory and remember new sources", actBrowse},
			{"⚙", "Sources…", "add or remove saved skill sources", actSourceActions},
		}},
		{label: "PLUGINS", entries: []menuEntry{
			{"⬢", "Install plugins", "pick a marketplace and plugins, then choose where to install", actInstallPlugins},
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

// installSelected opens destination choice for the marked skills.
func (m *model) installSelected() tea.Cmd {
	source := m.curSource
	if m.screen == scStarred {
		source = ""
	}
	return m.beginInstall(installSkills, source)
}

// installSelectedPlugins opens destination choice for the marked plugins.
func (m *model) installSelectedPlugins() tea.Cmd {
	return m.beginInstall(installPlugins, m.curMarket)
}

func (m *model) beginInstall(kind installKind, source string) tea.Cmd {
	sel := m.pick.selected()
	if len(sel) == 0 {
		return flashFor("nothing marked — press SPACE to mark, then ENTER", 0)
	}
	m.install = &installRequest{
		kind:   kind,
		items:  sel,
		source: source,
		origin: m.screen,
	}
	m.screen = scInstallDestination
	return nil
}

func (m *model) executeInstall(req installRequest) tea.Cmd {
	m.busyTitle = "installing " + installTitle(req)
	m.screen = scRunning

	if req.kind == installPlugins {
		if installNeedsHooks(req) && strings.Contains(m.agents, "codex") {
			pending := req
			m.pendingInstall = &pending
			m.busyTitle = "checking codex hooks for " + itoa(len(req.items)) + " " + installNoun(req.kind, len(req.items) != 1)
			return codexHooksCmd()
		}
		return m.runPluginInstall(req, false)
	}

	if req.origin == scStarred {
		return installGroupedCmd("install "+installTitle(req), req.global, req.items)
	}
	return opCmd("install "+installTitle(req), installArgs(req)...)
}

func (m *model) runPluginInstall(req installRequest, denyHooks bool) tea.Cmd {
	args := installArgs(req)
	if denyHooks {
		args = hooksDenyArgs(args)
	}
	m.busyTitle = "installing " + installTitle(req)
	m.screen = scRunning
	return opCmd("install "+installTitle(req), args...)
}

func installArgs(req installRequest) []string {
	args := make([]string, 0, len(req.items)*2+5)
	if req.global {
		args = append(args, "-g")
	}
	if req.kind == installPlugins {
		args = append(args, "plugin", "install", req.source)
		for _, it := range req.items {
			args = append(args, it.id)
		}
		return args
	}

	args = append(args, "use", req.source, "--")
	for _, it := range req.items {
		args = append(args, "--skill", it.id)
	}
	return append(args, "-y")
}

func installNeedsHooks(req installRequest) bool {
	for _, it := range req.items {
		if hasFlag(it.flags, "hooks") {
			return true
		}
	}
	return false
}

func installTitle(req installRequest) string {
	destination := "in this project"
	if req.global {
		destination = "globally"
	}
	return itoa(len(req.items)) + " " + installNoun(req.kind, len(req.items) != 1) + " " + destination
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
