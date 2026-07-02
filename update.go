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
	if (m.screen == scSkills || m.screen == scPlugins) && m.filtering {
		listH--
	}
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
		m.enterPicker(newPicker(applyStars(msg.items, m.curSource), true))
		m.screen = scSkills
		return m, nil

	case starredMsg:
		if len(msg.items) == 0 {
			m.screen = scMenu
			return m, flashFor("no starred skills yet — open a source and press s on a skill", 0)
		}
		m.enterPicker(newPicker(msg.items, true))
		m.screen = scStarred
		return m, nil

	case marketsMsg:
		if len(msg.items) == 0 {
			m.screen = scMenu
			return m, flashFor("no marketplaces yet — Add a marketplace first", 0)
		}
		m.enterPicker(newPicker(msg.items, false))
		m.screen = scMarkets
		return m, nil

	case pluginsMsg:
		if msg.err != nil || len(msg.items) == 0 {
			m.screen = scResult
			m.resultTitle = "could not list plugins for " + short(m.curMarket)
			m.resultErr = true
			body := "The engine could not read the marketplace manifests.\n\n"
			if msg.err != nil {
				body += msg.err.Error() + "\n\n"
			}
			body += "Tip: run `SWOOP_DEBUG=1 swoop _plugins " + m.curMarket + "` to see clone/parse errors."
			m.setResult(body)
			return m, nil
		}
		m.enterPicker(newPicker(msg.items, true))
		m.screen = scPlugins
		return m, nil

	case installedPluginsMsg:
		if msg.err != nil || len(msg.items) == 0 {
			m.screen = scMenu
			return m, flashFor("no installed plugins found", 0)
		}
		m.enterPicker(newPicker(msg.items, true))
		m.screen = scPluginRemove
		return m, nil

	case codexHooksMsg:
		args := m.pendingInstall
		m.pendingInstall = nil
		if len(args) == 0 {
			m.screen = scMenu
			return m, nil
		}
		if msg.state == "off" {
			m.prev = scPlugins
			m.screen = scConfirm
			m.confirmMsg = "Enable Codex hooks (features.hooks) so these plugins' hooks run?"
			// yes: plain install — the engine auto-enables under SWOOP_ASSUME_YES
			m.confirmCmd = func(mm *model) tea.Cmd {
				mm.busyTitle = "installing plugin(s)"
				mm.screen = scRunning
				return opCmd("install plugins", args...)
			}
			// no: same install, but tell the engine to leave features.hooks alone
			m.denyCmd = func(mm *model) tea.Cmd {
				mm.busyTitle = "installing plugin(s)"
				mm.screen = scRunning
				return opCmd("install plugins", hooksDenyArgs(args)...)
			}
			return m, nil
		}
		m.busyTitle = "installing plugin(s)"
		m.screen = scRunning
		return m, opCmd("install plugins", args...)

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
	case scSources, scSkills, scStarred, scBrowseResults, scRemove,
		scMarkets, scPlugins, scPluginRemove:
		return m.pick
	}
	return nil
}

// refreshSources rebuilds the source list in place (after an alias change),
// keeping the cursor on roughly the same row.
func (m *model) refreshSources() { m.refreshPicker(sourceItems()) }

// refreshMarkets does the same for the marketplace list.
func (m *model) refreshMarkets() { m.refreshPicker(marketItems()) }

func (m *model) refreshPicker(items []item) {
	cur := 0
	if m.pick != nil {
		cur = m.pick.cursor
	}
	m.enterPicker(newPicker(items, false))
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
			m.screen = m.prev
			return m, nil
		case "enter":
			_ = saveAlias(m.renameURL, m.input.Value())
			if m.prev == scMarkets {
				m.refreshMarkets()
			} else {
				m.refreshSources()
			}
			m.screen = m.prev
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case scSkills:
		if m.filtering {
			return m.onFilterKey(msg)
		}
		switch k {
		case "esc":
			m.busyTitle = "loading saved sources"
			m.screen = scRunning
			return m, loadSourcesCmd()
		case "q":
			m.screen = scMenu
		case "/":
			m.filtering = true
			m.input.Placeholder = "filter skills"
			m.input.SetValue(m.pick.filter)
			m.input.Focus()
			m.layout()
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
		case "s":
			return m, m.toggleCurrentStar()
		case "a":
			all := m.pick.visibleSelectedCount() < m.pick.len()
			m.pick.selectAll(all)
		case "tab":
			m.global = !m.global
		case "enter":
			return m, m.installSelected()
		}
		return m, nil

	case scMarkets:
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
				m.prev = scMarkets
				m.screen = scRename
				m.input.Placeholder = "friendly name (blank = show the source)"
				m.input.SetValue(loadAliases()[it.id])
				m.input.CursorEnd()
				m.input.Focus()
			}
		case "u":
			m.busyTitle = "updating marketplaces"
			m.screen = scRunning
			return m, opCmd("marketplace update", "mkt", "update")
		case "x":
			if it, ok := m.pick.current(); ok {
				src := it.id
				m.prev = scMarkets
				m.screen = scConfirm
				m.confirmMsg = "Remove marketplace " + short(src) + " from claude + codex?"
				m.confirmCmd = func(mm *model) tea.Cmd {
					mm.busyTitle = "removing marketplace"
					mm.screen = scRunning
					return opCmd("removed marketplace", "mkt", "remove", src)
				}
			}
		case "enter":
			if it, ok := m.pick.current(); ok {
				m.curMarket = it.id
				m.busyTitle = "reading plugins from " + short(it.id)
				m.screen = scRunning
				return m, loadPluginsCmd(it.id)
			}
		}
		return m, nil

	case scPlugins:
		if m.filtering {
			return m.onFilterKey(msg)
		}
		switch k {
		case "esc":
			m.busyTitle = "loading marketplaces"
			m.screen = scRunning
			return m, loadMarketsCmd()
		case "q":
			m.screen = scMenu
		case "/":
			m.filtering = true
			m.input.Placeholder = "filter plugins"
			m.input.SetValue(m.pick.filter)
			m.input.Focus()
			m.layout()
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
			all := m.pick.visibleSelectedCount() < m.pick.len()
			m.pick.selectAll(all)
		case "tab":
			m.global = !m.global
		case "enter":
			return m, m.installSelectedPlugins()
		}
		return m, nil

	case scPluginRemove:
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
				return m, flashFor("mark plugins with SPACE first", 0)
			}
			args := []string{"plugin", "remove"}
			for _, it := range sel {
				args = append(args, it.id)
			}
			m.busyTitle = "removing plugins"
			m.screen = scRunning
			return m, opCmd("removed plugins", args...)
		}
		return m, nil

	case scStarred:
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
		case " ":
			m.pick.toggle()
		case "a":
			all := m.pick.visibleSelectedCount() < m.pick.len()
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
			all := m.pick.visibleSelectedCount() < m.pick.len()
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
			return m, removeSourcesCmd(args...)
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
			m.addMarketplace = false
			m.screen = scMenu
			return m, nil
		case "enter":
			v := strings.TrimSpace(m.input.Value())
			if v == "" {
				m.addMarketplace = false
				m.screen = scMenu
				return m, nil
			}
			if m.addMarketplace {
				m.addMarketplace = false
				m.busyTitle = "adding marketplace"
				m.screen = scRunning
				return m, opCmd("added marketplace", "mkt", "add", v)
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
			m.denyCmd = nil
			return m, m.confirmCmd(m)
		case "n", "N":
			// an explicit "no" runs the deny action when one is set
			// (e.g. install plugins without enabling codex hooks)
			if deny := m.denyCmd; deny != nil {
				m.denyCmd = nil
				return m, deny(m)
			}
			m.screen = m.prev
		case "esc", "q":
			m.denyCmd = nil
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

// hooksDenyArgs is the deny path of the codex-hooks confirm: the same install,
// but the engine must leave features.hooks untouched.
func hooksDenyArgs(args []string) []string {
	return append(append([]string(nil), args...), "--no-hooks-enable")
}

func removeSourcesCmd(args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := core(args...)
		body := stripANSI(out)
		if err == nil {
			if pruneErr := pruneStarsForSources(args[1:]); pruneErr != nil {
				err = pruneErr
				if body != "" {
					body += "\n"
				}
				body += "failed to prune starred skills: " + pruneErr.Error()
			}
		}
		return opDoneMsg{title: "removed sources", output: body, err: err}
	}
}

func (m *model) onFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		m.input.Blur()
		m.layout()
		return m, nil
	case "esc":
		m.filtering = false
		m.input.SetValue("")
		m.input.Blur()
		m.pick.setFilter("")
		m.layout()
		return m, nil
	case "up", "k":
		m.filtering = false
		m.input.Blur()
		m.layout()
		m.pick.move(-1)
		return m, nil
	case "down", "j":
		m.filtering = false
		m.input.Blur()
		m.layout()
		m.pick.move(1)
		return m, nil
	case "home":
		m.filtering = false
		m.input.Blur()
		m.layout()
		m.pick.home()
		return m, nil
	case "end":
		m.filtering = false
		m.input.Blur()
		m.layout()
		m.pick.end()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.pick.setFilter(m.input.Value())
	return m, cmd
}

func (m *model) toggleCurrentStar() tea.Cmd {
	it, ok := m.pick.current()
	if !ok {
		return nil
	}
	on, err := toggleStar(it)
	if err != nil {
		return flashFor("could not update starred skills: "+err.Error(), 0)
	}
	it.star = on
	m.pick.replaceCurrent(it)
	if on {
		return flashFor("starred "+it.title, 0)
	}
	return flashFor("unstarred "+it.title, 0)
}
