package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) layout() {
	// chrome: header(2) + spacer(1) + border(2) + spacer(1) + status(1) = 7
	m.innerW = m.width - 4
	if m.innerW < 10 {
		m.innerW = 10
	}
	m.innerH = m.height - 7
	if m.innerH < 3 {
		m.innerH = 3
	}
	// body layout for list screens = heading(1) + blank(1) + list + footer(1)
	listH := m.innerH - 3
	if listH < 1 {
		listH = 1
	}
	if m.pick != nil {
		m.pick.setSize(m.innerW, listH)
	}
	if m.menu != nil {
		m.menu.setSize(m.innerW, listH)
	}
	m.input.Width = m.innerW - 4
	if m.vpReady {
		m.vp.Width = m.innerW
		m.vp.Height = m.innerH - 2 // result body = head(1) + blank(1) + viewport
		if m.vp.Height < 1 {
			m.vp.Height = 1
		}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.vpReady {
			m.vp = viewport.New(msg.Width-4, msg.Height-7)
			m.vpReady = true
		}
		m.layout()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case flashMsg:
		m.flash = string(msg)
		return m, nil
	case clearFlashMsg:
		m.flash = ""
		return m, nil

	case sourcesMsg:
		if len(msg.items) == 0 {
			m.screen = scMenu
			return m, flashFor("no saved sources yet — Browse or Add one first", 0)
		}
		m.enterPicker(newPicker(msg.items, false))
		m.screen = scSources
		return m, nil

	case skillsMsg:
		if msg.err != nil || len(msg.items) == 0 {
			m.screen = scResult
			m.resultTitle = "could not list skills for " + short(m.curSource)
			m.resultErr = true
			body := "The engine could not read individual skills.\n\n"
			if msg.err != nil {
				body += msg.err.Error() + "\n\n"
			}
			body += "Tip: run `SWOOP_DEBUG=1 swoop _skills " + m.curSource + "` to see clone/API errors."
			m.setResult(body)
			return m, nil
		}
		m.enterPicker(newPicker(msg.items, true))
		m.screen = scSkills
		return m, nil

	case searchMsg:
		if msg.err != nil || len(msg.items) == 0 {
			m.screen = scMenu
			return m, flashFor("no results (or search failed) — try another keyword", 0)
		}
		m.enterPicker(newPicker(msg.items, true))
		m.screen = scBrowseResults
		return m, nil

	case opDoneMsg:
		m.screen = scResult
		m.resultErr = msg.err != nil
		m.resultTitle = msg.title
		out := strings.TrimRight(msg.output, "\n")
		if strings.TrimSpace(out) == "" {
			if msg.err != nil {
				out = "failed: " + msg.err.Error()
			} else {
				out = "done."
			}
		}
		m.setResult(out)
		m.agents = loadAgents()
		return m, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.screen == scResult {
				m.vp.LineUp(2)
			} else if m.activeList() != nil {
				m.activeList().move(-1)
			}
		case tea.MouseButtonWheelDown:
			if m.screen == scResult {
				m.vp.LineDown(2)
			} else if m.activeList() != nil {
				m.activeList().move(1)
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.onKey(msg)
	}

	// route to active text input / viewport for non-key msgs
	if m.screen == scAdd || m.screen == scAgents || m.screen == scBrowseInput || m.screen == scRename {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// activeList returns whichever picker the current screen is driving (or nil).
func (m *model) activeList() *picker {
	switch m.screen {
	case scMenu:
		return m.menu
	case scSources, scSkills, scBrowseResults, scRemove:
		return m.pick
	}
	return nil
}

// refreshSources rebuilds the source list in place (after an alias change),
// keeping the cursor on roughly the same row.
func (m *model) refreshSources() {
	cur := 0
	if m.pick != nil {
		cur = m.pick.cursor
	}
	m.enterPicker(newPicker(sourceItems(), false))
	if cur >= m.pick.len() {
		cur = m.pick.len() - 1
	}
	if cur > 0 {
		m.pick.cursor = cur
		m.pick.clampWindow()
	}
}

func (m *model) setResult(body string) {
	m.vp.SetContent(body)
	m.vp.GotoTop()
}

func (m *model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// global
	if k == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.screen {

	case scMenu:
		switch k {
		case "up", "k":
			m.menu.move(-1)
		case "down", "j":
			m.menu.move(1)
		case "home", "g":
			// 'g' also toggles scope; use only home for top
			m.menu.home()
		case "G", "end":
			m.menu.end()
		case "tab":
			m.global = !m.global
		case "q":
			return m, tea.Quit
		case "enter", " ":
			return m.entries[m.menu.cursor].act(m)
		}
		return m, nil

	case scSources:
		switch k {
		case "esc", "q":
			m.screen = scMenu
		case "up", "k":
			m.pick.move(-1)
		case "down", "j":
			m.pick.move(1)
		case "home":
			m.pick.home()
		case "end":
			m.pick.end()
		case "tab":
			m.global = !m.global
		case "ctrl+r":
			if it, ok := m.pick.current(); ok {
				m.renameURL = it.id
				m.prev = scSources
				m.screen = scRename
				m.input.Placeholder = "friendly name (blank = show the repo URL)"
				m.input.SetValue(loadAliases()[it.id])
				m.input.CursorEnd()
				m.input.Focus()
			}
		case "enter":
			if it, ok := m.pick.current(); ok {
				m.curSource = it.id
				m.busyTitle = "scanning " + short(it.id)
				m.screen = scRunning
				return m, loadSkillsCmd(it.id)
			}
		}
		return m, nil

	case scRename:
		switch k {
		case "esc":
			m.screen = scSources
			return m, nil
		case "enter":
			_ = saveAlias(m.renameURL, m.input.Value())
			m.refreshSources()
			m.screen = scSources
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case scSkills:
		switch k {
		case "esc":
			m.busyTitle = "loading saved sources"
			m.screen = scRunning
			return m, loadSourcesCmd()
		case "q":
			m.screen = scMenu
		case "up", "k":
			m.pick.move(-1)
		case "down", "j":
			m.pick.move(1)
		case "home":
			m.pick.home()
		case "end":
			m.pick.end()
		case " ":
			m.pick.toggle()
		case "a":
			all := m.pick.selectedCount() < m.pick.len()
			m.pick.selectAll(all)
		case "tab":
			m.global = !m.global
		case "enter":
			return m, m.installSelected()
		}
		return m, nil

	case scBrowseResults:
		switch k {
		case "esc":
			m.screen = scBrowseInput
			m.input.Focus()
		case "q":
			m.screen = scMenu
		case "up", "k":
			m.pick.move(-1)
		case "down", "j":
			m.pick.move(1)
		case " ":
			m.pick.toggle()
		case "a":
			all := m.pick.selectedCount() < m.pick.len()
			m.pick.selectAll(all)
		case "enter":
			sel := m.pick.selected()
			if len(sel) == 0 {
				return m, flashFor("mark repos with SPACE first", 0)
			}
			args := []string{"add"}
			for _, it := range sel {
				args = append(args, it.id)
			}
			m.busyTitle = "remembering " + itoa(len(sel)) + " source(s)"
			m.screen = scRunning
			return m, opCmd("remembered sources", args...)
		}
		return m, nil

	case scRemove:
		switch k {
		case "esc", "q":
			m.screen = scMenu
		case "up", "k":
			m.pick.move(-1)
		case "down", "j":
			m.pick.move(1)
		case " ":
			m.pick.toggle()
		case "enter":
			sel := m.pick.selected()
			if len(sel) == 0 {
				return m, flashFor("mark sources with SPACE first", 0)
			}
			args := []string{"remove"}
			for _, it := range sel {
				args = append(args, it.id)
			}
			m.busyTitle = "removing sources"
			m.screen = scRunning
			return m, opCmd("removed sources", args...)
		}
		return m, nil

	case scBrowseInput:
		switch k {
		case "esc":
			m.screen = scMenu
			return m, nil
		case "enter":
			q := strings.TrimSpace(m.input.Value())
			m.busyTitle = "searching skills.sh"
			m.screen = scRunning
			return m, searchCmd(q)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case scAdd:
		switch k {
		case "esc":
			m.screen = scMenu
			return m, nil
		case "enter":
			v := strings.TrimSpace(m.input.Value())
			if v == "" {
				m.screen = scMenu
				return m, nil
			}
			m.busyTitle = "adding source"
			m.screen = scRunning
			return m, opCmd("added source", "add", v)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case scAgents:
		switch k {
		case "esc":
			m.screen = scMenu
			return m, nil
		case "enter":
			fields := strings.Fields(m.input.Value())
			if len(fields) == 0 {
				m.screen = scMenu
				return m, nil
			}
			args := append([]string{"agents"}, fields...)
			m.busyTitle = "saving default agents"
			m.screen = scRunning
			return m, opCmd("default agents", args...)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case scConfirm:
		switch k {
		case "y", "Y", "enter":
			return m, m.confirmCmd(m)
		case "n", "N", "esc", "q":
			m.screen = m.prev
		}
		return m, nil

	case scResult:
		switch k {
		case "esc", "enter", "q", "backspace":
			m.screen = scMenu
			return m, nil
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case scRunning:
		// ignore keys while working (except ctrl+c handled above)
		return m, nil
	}

	return m, nil
}
