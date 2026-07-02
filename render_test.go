package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
