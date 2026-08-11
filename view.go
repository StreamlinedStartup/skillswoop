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
	if m.showHelp {
		return m.helpBody()
	}

	// defensive: list screens must never render before their picker exists
	switch m.screen {
	case scSourceActions, scSources, scSkills, scStarred, scBrowseResults, scRemove,
		scMarkets, scPlugins, scPluginRemove:
		if m.pick == nil {
			return heading("LOADING", "") + "\n\n" + rowDesc.Render("  …")
		}
	}

	switch m.screen {

	case scMenu:
		return m.menuBody()

	case scSourceActions:
		return m.sourceActionsBody()

	case scSources:
		return heading("INSTALL", "pick a source") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scRename:
		label := "RENAME source"
		if m.prev == scMarkets {
			label = "RENAME marketplace"
		}
		return heading(label, short(m.renameURL)) + "\n\n" +
			m.input.View() + "\n\n" + rowDesc.Render("blank shows the repository URL")

	case scSkills:
		if m.filtering {
			return heading("INSTALL · "+short(m.curSource), "") + "\n" +
				m.input.View() + "\n" +
				m.pick.view() + "\n" + m.pick.scrollFooter()
		}
		return heading("INSTALL · "+short(m.curSource), "") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scStarred:
		return heading("STARRED skills", "") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scBrowseResults:
		return heading("BROWSE", "results from skills.sh") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scRemove:
		return heading("REMOVE", "saved skill sources") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scMarkets:
		return heading("MARKETPLACES", "pick a marketplace") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scPlugins:
		if m.filtering {
			return heading("PLUGINS · "+short(m.curMarket), "") + "\n" +
				m.input.View() + "\n" +
				m.pick.view() + "\n" + m.pick.scrollFooter()
		}
		return heading("PLUGINS · "+short(m.curMarket), "") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scPluginRemove:
		return heading("REMOVE plugins", "installed for configured agents") + "\n\n" +
			m.pick.view() + "\n" + m.pick.scrollFooter()

	case scInstallDestination:
		return m.installDestinationBody()

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

func (m *model) menuBody() string {
	domain := m.domains[m.domain]
	desc := m.flash
	if desc == "" && m.menu.cursor < len(domain.entries) {
		desc = domain.entries[m.menu.cursor].desc
	}
	compact := m.innerH < 12
	separator := "\n\n"
	if compact {
		separator = "\n"
	}
	return m.domainTabs() + separator + m.disclosureBody("", "", m.menu, desc, compact)
}

func (m *model) sourceActionsBody() string {
	entries := sourceActionEntries()
	desc := ""
	if m.pick.cursor < len(entries) {
		desc = entries[m.pick.cursor].desc
	}
	return m.disclosureBody("SOURCES", "manage saved skill sources", m.pick, desc, m.innerH < 9)
}

func (m *model) disclosureBody(label, sub string, p *picker, desc string, compact bool) string {
	desc = truncate(desc, m.innerW)
	prefix := ""
	if label != "" {
		if compact {
			prefix = heading(label, "") + "\n"
		} else {
			prefix = heading(label, sub) + "\n\n"
		}
	}
	if compact {
		return prefix + p.view() + "\n" + rowDescCur.Render(desc)
	}
	rule := rowDesc.Render(strings.Repeat("─", m.innerW))
	return prefix + p.view() + "\n\n" + rule + "\n" + rowDescCur.Render(desc)
}

func (m *model) domainTabs() string {
	if m.innerW < 38 {
		return rowDesc.Render("‹ ") + titleStyle.Render(m.domains[m.domain].label) +
			rowDesc.Render("  "+itoa(m.domain+1)+"/"+itoa(len(m.domains))+" ›")
	}
	var tabs []string
	for i, domain := range m.domains {
		label := "  " + domain.label + "  "
		if i == m.domain {
			tabs = append(tabs, titleStyle.Render("["+label+"]"))
		} else {
			tabs = append(tabs, rowDesc.Render(" "+label+" "))
		}
	}
	return strings.Join(tabs, "   ")
}

type keyHint struct {
	keys string
	desc string
}

func (m *model) helpBody() string {
	hints := m.contextKeys()
	lines := []string{heading("KEYS · "+m.contextLabel(), "")}
	for _, hint := range hints {
		lines = append(lines, helpKey.Render(padRight(hint.keys, 18))+rowDescCur.Render(hint.desc))
	}
	lines = append(lines, helpKey.Render(padRight("? / esc", 18))+rowDescCur.Render("close help"))
	return strings.Join(lines, "\n")
}

func (m *model) contextLabel() string {
	switch m.screen {
	case scMenu:
		return m.domains[m.domain].label
	case scSourceActions, scSources:
		return "SOURCES"
	case scSkills:
		return "SKILLS"
	case scStarred:
		return "STARRED"
	case scBrowseInput, scBrowseResults:
		return "BROWSE"
	case scAdd:
		if m.addMarketplace {
			return "ADD MARKETPLACE"
		}
		return "ADD SOURCE"
	case scAgents:
		return "DEFAULT AGENTS"
	case scRemove:
		return "REMOVE SOURCES"
	case scRename:
		return "RENAME"
	case scConfirm:
		return "CONFIRM"
	case scResult:
		return "RESULT"
	case scMarkets:
		return "MARKETPLACES"
	case scPlugins:
		return "PLUGINS"
	case scInstallDestination:
		return "INSTALL DESTINATION"
	case scPluginRemove:
		return "REMOVE PLUGINS"
	}
	return "SWOOP"
}

func (m *model) contextKeys() []keyHint {
	move := keyHint{"↑ / ↓ · j / k", "move selection"}
	back := keyHint{"esc", "go back"}
	switch m.screen {
	case scMenu:
		return []keyHint{
			{"← / → · h / l", "change section"},
			move,
			{"enter / space", "select action"},
			{"tab", "toggle update scope"},
			{"q", "quit"},
		}
	case scSourceActions:
		return []keyHint{move, {"enter / space", "select action"}, back, {"q", "return to menu"}}
	case scSources:
		return []keyHint{move, {"enter", "open source"}, {"ctrl+r", "rename source"}, back}
	case scSkills:
		if m.filtering {
			return []keyHint{{"type", "filter skills"}, {"enter", "return to skill list"}, {"esc", "clear filter"}}
		}
		return []keyHint{move, {"space", "mark skill"}, {"s", "star skill"}, {"/", "filter skills"}, {"a", "mark all or none"}, {"enter", "choose install destination"}, back}
	case scStarred:
		return []keyHint{move, {"space", "mark skill"}, {"a", "mark all or none"}, {"enter", "choose install destination"}, back}
	case scBrowseResults:
		return []keyHint{move, {"space", "mark repository"}, {"a", "mark all or none"}, {"enter", "remember marked sources"}, back}
	case scRemove:
		return []keyHint{move, {"space", "mark source"}, {"enter", "remove marked sources"}, back}
	case scMarkets:
		return []keyHint{move, {"enter", "open marketplace"}, {"ctrl+r", "rename marketplace"}, {"x", "remove marketplace"}, {"u", "update marketplaces"}, back}
	case scPlugins:
		if m.filtering {
			return []keyHint{{"type", "filter plugins"}, {"enter", "return to plugin list"}, {"esc", "clear filter"}}
		}
		return []keyHint{move, {"space", "mark plugin"}, {"/", "filter plugins"}, {"a", "mark all or none"}, {"enter", "choose install destination"}, back}
	case scInstallDestination:
		return []keyHint{move, {"enter", "install here"}, {"esc", "change selection"}}
	case scPluginRemove:
		return []keyHint{move, {"space", "mark plugin"}, {"enter", "remove marked plugins"}, back}
	case scRename:
		return []keyHint{{"type", "edit display name"}, {"enter", "save"}, {"esc", "cancel"}}
	case scBrowseInput:
		return []keyHint{{"type", "enter search keywords"}, {"enter", "search skills.sh"}, {"esc", "cancel"}}
	case scAdd, scAgents:
		return []keyHint{{"type", "edit value"}, {"enter", "confirm"}, {"esc", "cancel"}}
	case scConfirm:
		return []keyHint{{"y / enter", "confirm"}, {"n", "decline"}, {"esc", "cancel"}}
	case scResult:
		return []keyHint{{"↑ / ↓", "scroll output"}, {"esc / enter", "return to menu"}}
	}
	return nil
}

func (m *model) inputBody(label, sub string) string {
	return heading(label, sub) + "\n\n" + m.input.View()
}

func (m *model) installDestinationBody() string {
	if m.install == nil {
		return heading("INSTALL", "destination unavailable")
	}
	req := *m.install
	compact := m.innerH < 14
	agents := strings.ReplaceAll(m.agents, " ", " and ")
	globalDescription := "Every project for " + agents
	if !compact {
		globalDescription = "Available in every project for " + agents
	}
	lines := []string{heading(installHeading(req), "")}
	if !compact && req.source != "" {
		lines = append(lines, "", rowDesc.Render("From ")+rowCursor.Render(short(req.source)))
	}
	if !compact {
		lines = append(lines,
			"",
			rowDesc.Render("Selected")+"\n"+
				rowCursor.Render(truncate(selectionSummary(req), m.innerW))+"\n\n"+
				rowDescCur.Render(truncate(installQuestion(req), m.innerW)),
			"",
		)
	} else if m.innerH >= 7 {
		lines = append(lines,
			rowDesc.Render("Selected ")+rowCursor.Render(truncate(selectionSummary(req), m.innerW-9)),
			rowDescCur.Render(truncate(installQuestion(req), m.innerW)),
		)
	}
	lines = append(lines,
		destinationRow(!req.global, "This project"),
		rowDesc.Render("    "+truncate("Only in "+m.projectDir, m.innerW-4)),
	)
	if !compact {
		lines = append(lines, "")
	}
	lines = append(lines,
		destinationRow(req.global, "Globally"),
		rowDesc.Render("    "+truncate(globalDescription, m.innerW-4)),
	)
	return strings.Join(lines, "\n")
}

func destinationRow(active bool, label string) string {
	marker := "  ○ "
	style := rowNormal
	if active {
		marker = barStyle.Render("▌ ") + checkOn.Render("● ")
		style = rowCursor
	}
	return marker + style.Render(label)
}

func installHeading(req installRequest) string {
	return "INSTALL " + itoa(len(req.items)) + " " + strings.ToUpper(installNoun(req.kind, len(req.items) != 1))
}

func installNoun(kind installKind, plural bool) string {
	noun := "skill"
	if kind == installPlugins {
		noun = "plugin"
	}
	if plural {
		noun += "s"
	}
	return noun
}

func installQuestion(req installRequest) string {
	items := "this " + installNoun(req.kind, false)
	if len(req.items) != 1 {
		items = "these " + installNoun(req.kind, true)
	}
	return "Where should " + items + " be installed?"
}

func selectionSummary(req installRequest) string {
	limit := len(req.items)
	if limit > 3 {
		limit = 3
	}
	names := make([]string, 0, limit+1)
	for _, it := range req.items[:limit] {
		name := it.id
		if req.origin == scStarred && it.source != "" {
			name += " (" + short(it.source) + ")"
		}
		names = append(names, name)
	}
	if remaining := len(req.items) - limit; remaining > 0 {
		names = append(names, "+ "+itoa(remaining)+" more")
	}
	return strings.Join(names, ", ")
}

func (m *model) statusBar() string {
	scope := scopeProj.Render("PROJECT")
	if m.global {
		scope = scopeGlob.Render("GLOBAL")
	}
	agents := chipStyle.Render(strings.ReplaceAll(m.agents, " ", " ⋅ "))

	var keys string
	if m.showHelp {
		keys = key("?", "close") + key("esc", "close")
	} else {
		keys = m.compactKeys()
	}

	left := keys
	if m.screen == scMenu {
		left += helpDesc.Render(" │ ") + helpDesc.Render("update scope ") + scope
	}
	right := helpDesc.Render("agents ") + agents
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncate(stripANSI(left), m.width)
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *model) compactKeys() string {
	switch m.screen {
	case scMenu:
		return key("←→", "section") + key("↑↓", "move") + key("⏎", "select") + key("?", "keys")
	case scSourceActions:
		return key("↑↓", "move") + key("⏎", "select") + key("esc", "back") + key("?", "keys")
	case scSources:
		return key("↑↓", "move") + key("⏎", "open") + key("esc", "back") + key("?", "keys")
	case scRename:
		return key("⏎", "save") + key("esc", "cancel") + key("?", "keys")
	case scSkills:
		if m.filtering {
			return key("type", "filter") + key("⏎", "list") + key("esc", "clear") + key("?", "keys")
		}
		return key("↑↓", "move") + key("space", "mark") + key("⏎", "continue") + key("?", "keys")
	case scStarred:
		return key("↑↓", "move") + key("space", "mark") + key("⏎", "continue") + key("?", "keys")
	case scBrowseResults, scRemove, scPluginRemove:
		return key("↑↓", "move") + key("space", "mark") + key("⏎", "confirm") + key("?", "keys")
	case scMarkets:
		return key("↑↓", "move") + key("⏎", "open") + key("esc", "back") + key("?", "keys")
	case scPlugins:
		if m.filtering {
			return key("type", "filter") + key("⏎", "list") + key("esc", "clear") + key("?", "keys")
		}
		return key("↑↓", "move") + key("space", "mark") + key("⏎", "continue") + key("?", "keys")
	case scInstallDestination:
		return key("↑↓", "choose") + key("⏎", "install") + key("esc", "selection") + key("?", "keys")
	case scBrowseInput, scAdd, scAgents:
		return key("⏎", "confirm") + key("esc", "cancel") + key("?", "keys")
	case scConfirm:
		return key("y", "yes") + key("n", "no") + key("?", "keys")
	case scResult:
		return key("↑↓", "scroll") + key("esc", "menu") + key("?", "keys")
	case scRunning:
		return helpDesc.Render("working…")
	}
	return key("?", "keys")
}

func key(k, d string) string {
	return helpKey.Render(k) + helpDesc.Render(" "+d+"  ")
}
