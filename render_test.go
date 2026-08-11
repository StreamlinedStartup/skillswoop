package main

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestMenuDisclosesOnlySelectedDescription(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	view := stripANSI(mm.View())
	if !strings.Contains(view, "pick a source and skills, then choose where to install") {
		t.Fatal("selected action description is missing")
	}
	if strings.Contains(view, "install skills you've starred for quick reuse") {
		t.Fatal("unselected action description should be hidden")
	}
	if !strings.Contains(view, "SKILLS") || !strings.Contains(view, "PLUGINS") || !strings.Contains(view, "SETTINGS") {
		t.Fatal("domain tabs are missing")
	}
	if got := strings.Count(view, "SKILLS"); got != 1 {
		t.Fatalf("active domain label appears %d times, want 1", got)
	}
}

func TestScreenInstructionsLiveInContextualHelp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.curSource = "owner/repo"
	m.enterPicker(newPicker([]item{{id: "tdd", title: "tdd"}}, true))
	m.screen = scSkills
	m.width, m.height = 96, 30
	m.layout()

	view := stripANSI(m.View())
	if strings.Contains(view, "SPACE marks") || strings.Contains(view, "/ filters") {
		t.Fatal("screen heading still repeats shortcut instructions")
	}
	status := stripANSI(m.statusBar())
	if !strings.Contains(status, "? keys") {
		t.Fatal("compact status bar does not advertise contextual help")
	}
	if strings.Contains(status, "s star") || strings.Contains(status, "/ filter") || strings.Contains(status, "a all") {
		t.Fatal("status bar still exposes secondary shortcuts")
	}
}

func TestInstallDestinationUsesRewrittenChrome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.width, m.height = 96, 30
	m.projectDir = "/workspace/skillswoop"
	m.agents = "claude-code codex"
	m.install = &installRequest{
		kind:   installSkills,
		source: "owner/skills",
		origin: scSkills,
		items: []item{
			{id: "alpha", title: "alpha"},
			{id: "bravo", title: "bravo"},
		},
	}
	m.screen = scInstallDestination
	m.layout()

	view := stripANSI(m.View())
	for _, want := range []string{
		"INSTALL 2 SKILLS",
		"From owner/skills",
		"alpha, bravo",
		"This project",
		"Only in /workspace/skillswoop",
		"Globally",
		"Available in every project for claude-code and codex",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("destination view missing %q", want)
		}
	}

	status := stripANSI(m.statusBar())
	for _, want := range []string{"↑↓ choose", "⏎ install", "esc selection", "? keys", "agents claude-code ⋅ codex"} {
		if !strings.Contains(status, want) {
			t.Fatalf("destination status missing %q: %q", want, status)
		}
	}
	if strings.Contains(status, "scope") {
		t.Fatalf("destination status exposes update scope: %q", status)
	}
}

func TestMenuLabelsScopeAsUpdateOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.width, m.height = 96, 30
	m.layout()

	status := stripANSI(m.statusBar())
	if !strings.Contains(status, "update scope PROJECT") {
		t.Fatalf("menu status does not label update scope: %q", status)
	}
}

func TestStarredInstallSummaryNamesSourcesAndCollapsesOverflow(t *testing.T) {
	req := installRequest{
		kind:   installSkills,
		origin: scStarred,
		items: []item{
			{id: "alpha", source: "one/skills"},
			{id: "bravo", source: "two/skills"},
			{id: "charlie", source: "three/skills"},
			{id: "delta", source: "four/skills"},
			{id: "echo", source: "five/skills"},
		},
	}

	got := selectionSummary(req)
	for _, want := range []string{"alpha (one/skills)", "bravo (two/skills)", "charlie (three/skills)", "+ 2 more"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "delta") || strings.Contains(got, "echo") {
		t.Fatalf("summary did not collapse overflow: %q", got)
	}
}

func TestInstallDestinationFitsSupportedTerminalWidths(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{
		{Width: 24, Height: 12},
		{Width: 40, Height: 14},
		{Width: 96, Height: 30},
	} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			m := newModel()
			m.projectDir = "/workspace/a-project-with-a-long-name"
			m.install = &installRequest{
				kind:   installPlugins,
				origin: scPlugins,
				source: "owner/a-marketplace-with-a-long-name",
				items:  []item{{id: "one"}, {id: "two"}, {id: "three"}, {id: "four"}},
			}
			m.screen = scInstallDestination
			var mm tea.Model = m
			mm, _ = mm.Update(size)

			view := stripANSI(mm.View())
			if !strings.Contains(view, "This project") || !strings.Contains(view, "Globally") {
				t.Fatalf("destination choices are clipped at %dx%d", size.Width, size.Height)
			}
			for i, line := range strings.Split(mm.View(), "\n") {
				if got := lipgloss.Width(line); got > size.Width {
					t.Fatalf("line %d width = %d, want <= %d", i+1, got, size.Width)
				}
			}
		})
	}
}

func TestMenuFitsSupportedTerminalSizes(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{
		{Width: 24, Height: 12},
		{Width: 40, Height: 14},
		{Width: 96, Height: 30},
	} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			m := newModel()
			var mm tea.Model = m
			mm, _ = mm.Update(size)
			view := mm.View()
			lines := strings.Split(view, "\n")
			if len(lines) > size.Height {
				t.Fatalf("rendered %d lines at height %d", len(lines), size.Height)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got > size.Width {
					t.Fatalf("line %d width = %d, want <= %d", i+1, got, size.Width)
				}
			}
		})
	}
}

func TestSourceActionsDiscloseOnlySelectedDescription(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	mm, _ = actSourceActions(m)

	view := stripANSI(mm.View())
	if !strings.Contains(view, "owner/repo · git URL · local path") {
		t.Fatal("selected source action description is missing")
	}
	if strings.Contains(view, "forget one or more saved sources") {
		t.Fatal("unselected source action description should be hidden")
	}
}

// TestRender drives the model without a TTY to prove screens render w/o panic.
func TestRender(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	dump := func(label string) {
		fmt.Println("\n================= " + label + " =================")
		fmt.Println(mm.View())
	}

	dump("MENU")

	// populate a skills picker
	items := []item{
		{id: "diagnose", title: "diagnose", desc: "Disciplined diagnosis loop for hard bugs and performance regressions."},
		{id: "tdd", title: "tdd", desc: "Test-driven development with a red-green-refactor loop."},
		{id: "grill-with-docs", title: "grill-with-docs", desc: "Grilling session that challenges your plan against the domain model."},
		{id: "to-issues", title: "to-issues", desc: "Break a plan into independently-grabbable issues."},
	}
	model := mm.(*model)
	model.curSource = "https://github.com/mattpocock/skills"
	model.enterPicker(newPicker(applyStars(items, model.curSource), true))
	model.pick.items[0].sel = true
	model.pick.items[2].sel = true
	model.pick.cursor = 1
	model.screen = scSkills
	dump("SKILLS (multi-select)")

	model.filtering = true
	model.input.Placeholder = "filter skills"
	model.input.SetValue("tdd")
	model.pick.setFilter("tdd")
	dump("SKILLS (filtered)")
	model.filtering = false
	model.pick.setFilter("")

	model.enterPicker(newPicker([]item{
		{id: "tdd", title: "tdd", desc: "mattpocock/skills  ·  Test-driven development.", source: "https://github.com/mattpocock/skills", star: true},
	}, true))
	model.screen = scStarred
	dump("STARRED")

	model.screen = scRunning
	model.busyTitle = "installing 2 skill(s) from mattpocock/skills"
	dump("RUNNING")

	model.screen = scResult
	model.resultTitle = "install"
	model.setResult(">> installing from mattpocock/skills into /tmp/proj\n  ✓ diagnose (copied)\n  ✓ tdd (copied)\nDone!")
	dump("RESULT")

	model.screen = scAdd
	model.input.Placeholder = "owner/repo | https://… | ~/path/to/skill"
	model.input.SetValue("vercel-labs/agent-skills")
	dump("ADD INPUT")
}

// TestRenderPluginScreens proves the plugin screens render without panic,
// including badge rows at a narrow width.
func TestRenderPluginScreens(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	model := mm.(*model)

	dump := func(label string) {
		fmt.Println("\n================= " + label + " =================")
		fmt.Println(mm.View())
	}

	model.enterPicker(newPicker([]item{
		{id: "anthropics/claude-plugins-official", title: "anthropics/claude-plugins-official", desc: "claude + codex"},
		{id: "owner/mkt", title: "owner/mkt", desc: "claude only"},
	}, false))
	model.screen = scMarkets
	dump("MARKETS")

	plugins := []item{
		{id: "hooky", title: "hooky", desc: "A plugin with hooks and mcp.", flags: "claude,codex,hooks,mcp"},
		{id: "plain", title: "plain", desc: "Nothing fancy. · claude only", flags: "claude"},
	}
	model.curMarket = "anthropics/claude-plugins-official"
	model.enterPicker(newPicker(plugins, true))
	model.pick.items[0].sel = true
	model.screen = scPlugins
	dump("PLUGINS (badges)")

	// narrow terminal: badges must not overflow the row budget
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	dump("PLUGINS (narrow)")
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	model.enterPicker(newPicker([]item{
		{id: "hooky@test-mkt", title: "hooky@test-mkt", desc: "claude,codex  ·  A plugin."},
	}, true))
	model.screen = scPluginRemove
	dump("REMOVE PLUGINS")

	model.addMarketplace = true
	model.screen = scAdd
	model.input.Placeholder = "owner/repo | https://… | ~/path/to/marketplace"
	model.input.SetValue("anthropics/claude-plugins-official")
	dump("ADD MARKETPLACE INPUT")
}

func TestBadgeRowFitsWidth(t *testing.T) {
	p := newPicker([]item{
		{id: "hooky", title: "a-rather-long-plugin-name", desc: "some description here", flags: "claude,codex,hooks,mcp"},
	}, true)
	for _, w := range []int{20, 28, 40, 80} {
		p.setSize(w, 5)
		row := p.renderRow(0)
		if got := lipgloss.Width(row); got > w {
			t.Fatalf("row width %d exceeds picker width %d: %q", got, w, row)
		}
	}
}
