package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDownArrowLeavesFilterAndMovesList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	m.curSource = "owner/repo"
	m.enterPicker(newPicker([]item{
		{id: "alpha", title: "alpha"},
		{id: "bravo", title: "bravo"},
		{id: "charlie", title: "charlie"},
	}, true))
	m.screen = scSkills
	m.filtering = true
	m.input.Focus()
	m.pick.setFilter("a")

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = mm.(*model)

	if m.filtering {
		t.Fatal("expected down arrow to leave filter mode")
	}
	if m.pick.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.pick.cursor)
	}
}
