package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func menuEntries() []menuEntry {
	return []menuEntry{
		{"◢◤", "Install skills", "pick a source, then jack specific skills into this folder", actInstall},
		{"⟳", "Update folder", "pull latest from GitHub for skills in the current dir", actUpdateHere},
		{"⟳⟳", "Update all folders", "refresh every folder you've installed into", actUpdateAll},
		{"⌖", "Browse skills.sh", "search the directory and remember new sources", actBrowse},
		{"＋", "Add a source", "owner/repo · git URL · local path", actAdd},
		{"✕", "Remove a source", "forget a saved source", actRemove},
		{"⚙", "Default agents", "choose which agents to target", actAgents},
		{"⤓", "Tidy global skills", "move ~/.claude & ~/.codex skills into the library", actTidy},
		{"⏻", "Quit", "exit swoop", actQuit},
	}
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
	m.input.Placeholder = "owner/repo | https://… | ~/path/to/skill"
	m.input.SetValue("")
	m.input.Focus()
	return m, nil
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

func actQuit(m *model) (tea.Model, tea.Cmd) { return m, tea.Quit }

// installSelected builds the engine call from the marked skills.
func (m *model) installSelected() tea.Cmd {
	sel := m.pick.selected()
	if len(sel) == 0 {
		return flashFor("nothing marked — press SPACE to mark, then ENTER", 0)
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

func short(src string) string {
	s := strings.TrimSuffix(src, ".git")
	if i := strings.LastIndex(s, "github.com"); i >= 0 {
		s = strings.TrimLeft(s[i+len("github.com"):], ":/")
	}
	return s
}
