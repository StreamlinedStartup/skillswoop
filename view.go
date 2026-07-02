package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) View() string {
	if m.width < 24 || m.height < 12 {
		return "swoop — terminal too small (resize to at least 24x12)"
	}

	header := clampBlock(banner(m.width-2), m.width)
	body := m.buildBody()
	content := lipgloss.NewStyle().Width(m.innerW).Render(padLines(body, m.innerH))
	panel := panelStyle.Render(content)
	status := m.statusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, "", panel, "", status)
}

func clampBlock(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if lipgloss.Width(ln) > w {
			lines[i] = truncate(stripANSI(ln), w)
		}
	}
	return strings.Join(lines, "\n")
}

func heading(label, sub string) string {
	h := titleStyle.Render("▸ " + label)
	if sub != "" {
		h += rowDesc.Render("   " + sub)
	}
	return h
}

func (m *model) buildBody() string {
	// defensive: list screens must never render before their picker exists
	switch m.screen {
	case scSources, scSkills, scStarred, scBrowseResults, scRemove,
		scMarkets, scPlugins, scPluginRemove:
		if m.pick == nil {
			return heading("LOADING", "") + "\n\n" + rowDesc.Render("  …")
		}
	}

	switch m.screen {

	case scMenu:
		return heading("MAIN", "what do you want to do?") + "\n\n" +
			m.menu.view() + "\n" + m.menu.scrollFooter()

	case scSources:
		return heading("INSTALL", "pick a source · ctrl+r renames") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scRename:
		return heading("RENAME source", short(m.renameURL)) + "\n\n" +
			m.input.View() + "\n\n" +
			rowDesc.Render("ENTER save · ESC cancel · blank = show the repo URL")

	case scSkills:
		sub := "SPACE marks · s stars · / filters · ENTER installs marked"
		if m.filtering {
			return heading("INSTALL · "+short(m.curSource), sub) + "\n" +
				m.input.View() + "\n" +
				m.pick.view() + "\n" + m.pick.scrollFooter()
		}
		return heading("INSTALL · "+short(m.curSource), sub) + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scStarred:
		return heading("STARRED skills", "SPACE marks · ENTER installs marked") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scBrowseResults:
		return heading("BROWSE", "SPACE marks repos to remember · ENTER saves") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scRemove:
		return heading("REMOVE", "SPACE marks sources to forget · ENTER removes") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scMarkets:
		return heading("PLUGINS", "pick a marketplace · x removes · u updates") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scPlugins:
		sub := "SPACE marks · / filters · ENTER installs marked"
		if m.filtering {
			return heading("PLUGINS · "+short(m.curMarket), sub) + "\n" +
				m.input.View() + "\n" +
				m.pick.view() + "\n" + m.pick.scrollFooter()
		}
		return heading("PLUGINS · "+short(m.curMarket), sub) + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scPluginRemove:
		return heading("REMOVE plugins", "SPACE marks plugins to uninstall · ENTER removes") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scBrowseInput:
		return m.inputBody("BROWSE skills.sh", "type a keyword, then ENTER")

	case scAdd:
		if m.addMarketplace {
			return m.inputBody("ADD a marketplace", "a repo with a plugin marketplace manifest")
		}
		return m.inputBody("ADD a source", "owner/repo · git URL · local path")

	case scAgents:
		return m.inputBody("DEFAULT agents", "which agents to target on install")

	case scConfirm:
		return heading("CONFIRM", "") + "\n\n" +
			rowCursor.Render(m.confirmMsg) + "\n\n" +
			helpKey.Render("[y]") + helpDesc.Render(" yes   ") +
			helpKey.Render("[n]") + helpDesc.Render(" no")

	case scRunning:
		dots := m.spin.View()
		return "\n\n" + lipgloss.NewStyle().Width(m.innerW).Align(lipgloss.Center).Render(
			titleStyle.Render(dots+"  "+m.busyTitle+"  "+dots)+"\n\n"+
				rowDesc.Render("running the engine…"))

	case scResult:
		var head string
		if m.resultErr {
			head = errStyle.Render("✖ " + m.resultTitle)
		} else {
			head = okStyle.Render("✓ " + m.resultTitle)
		}
		return head + "\n\n" + m.vp.View()
	}
	return ""
}

func (m *model) inputBody(label, sub string) string {
	return heading(label, sub) + "\n\n" +
		m.input.View() + "\n\n" +
		rowDesc.Render("ENTER confirm · ESC cancel")
}

func (m *model) statusBar() string {
	scope := scopeProj.Render("PROJECT")
	if m.global {
		scope = scopeGlob.Render("GLOBAL")
	}
	agents := chipStyle.Render(strings.ReplaceAll(m.agents, " ", " ⋅ "))

	var keys string
	switch m.screen {
	case scMenu:
		keys = key("↑↓", "move") + key("⏎", "select") + key("tab", "scope") + key("q", "quit")
	case scSources:
		keys = key("↑↓", "move") + key("⏎", "open") + key("ctrl+r", "rename") + key("esc", "back")
	case scRename:
		keys = key("⏎", "save") + key("esc", "cancel")
	case scSkills:
		if m.filtering {
			keys = key("type", "filter") + key("⏎", "list") + key("esc", "clear")
		} else {
			keys = key("↑↓", "move") + key("space", "mark") + key("s", "star") + key("/", "filter") + key("a", "all") + key("⏎", "install") + key("esc", "back")
		}
	case scStarred:
		keys = key("↑↓", "move") + key("space", "mark") + key("a", "all") + key("⏎", "install") + key("esc", "menu")
	case scBrowseResults, scRemove, scPluginRemove:
		keys = key("↑↓", "move") + key("space", "mark") + key("⏎", "go") + key("esc", "back")
	case scMarkets:
		keys = key("↑↓", "move") + key("⏎", "open") + key("x", "remove") + key("u", "update") + key("esc", "back")
	case scPlugins:
		if m.filtering {
			keys = key("type", "filter") + key("⏎", "list") + key("esc", "clear")
		} else {
			keys = key("↑↓", "move") + key("space", "mark") + key("/", "filter") + key("a", "all") + key("⏎", "install") + key("esc", "back")
		}
	case scBrowseInput, scAdd, scAgents:
		keys = key("⏎", "confirm") + key("esc", "cancel")
	case scConfirm:
		keys = key("y", "yes") + key("n", "no")
	case scResult:
		keys = key("↑↓", "scroll") + key("esc", "menu")
	case scRunning:
		keys = helpDesc.Render("working…")
	}

	left := keys + helpDesc.Render(" │ ") + helpDesc.Render("scope ") + scope
	right := helpDesc.Render("agents ") + agents
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		// drop the agents chip on narrow terminals
		return truncate(stripANSI(left), m.width)
	}
	return left + strings.Repeat(" ", gap) + right
}

func key(k, d string) string {
	return helpKey.Render(k) + helpDesc.Render(" "+d+"  ")
}
