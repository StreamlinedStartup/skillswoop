package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
