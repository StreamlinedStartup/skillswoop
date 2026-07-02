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

func TestCodexHooksMsgOffRoutesToConfirm(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	m.screen = scRunning
	m.pendingInstall = []string{"plugin", "install", "owner/mkt", "hooky"}

	mm, _ = m.Update(codexHooksMsg{state: "off"})
	m = mm.(*model)

	if m.screen != scConfirm {
		t.Fatalf("screen = %d, want scConfirm", m.screen)
	}
	if m.denyCmd == nil {
		t.Fatal("expected a deny action for the hooks confirm")
	}
	if len(m.pendingInstall) != 0 {
		t.Fatalf("pendingInstall not cleared: %#v", m.pendingInstall)
	}
}

func TestCodexHooksMsgOnInstallsDirectly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	m.screen = scRunning
	m.pendingInstall = []string{"plugin", "install", "owner/mkt", "hooky"}

	mm, cmd := m.Update(codexHooksMsg{state: "on"})
	m = mm.(*model)

	if m.screen != scRunning {
		t.Fatalf("screen = %d, want scRunning", m.screen)
	}
	if cmd == nil {
		t.Fatal("expected an install command")
	}
	if m.denyCmd != nil {
		t.Fatal("denyCmd should stay nil on the direct path")
	}
}

func TestHooksDenyArgsAppendsFlag(t *testing.T) {
	args := []string{"plugin", "install", "owner/mkt", "hooky"}
	got := hooksDenyArgs(args)
	if got[len(got)-1] != "--no-hooks-enable" {
		t.Fatalf("deny args = %v", got)
	}
	if len(args) != 4 {
		t.Fatalf("original args mutated: %v", args)
	}
}

func TestMarketRenameRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	m.enterPicker(newPicker([]item{
		{id: "anthropics/claude-plugins-official", title: "anthropics/claude-plugins-official", desc: "claude only"},
	}, false))
	m.screen = scMarkets

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = mm.(*model)
	if m.screen != scRename || m.prev != scMarkets {
		t.Fatalf("screen = %d, prev = %d; want scRename from scMarkets", m.screen, m.prev)
	}
	m.input.SetValue("Official plugins")
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(*model)
	if m.screen != scMarkets {
		t.Fatalf("screen = %d, want scMarkets after save", m.screen)
	}
	if got := loadAliases()["anthropics/claude-plugins-official"]; got != "Official plugins" {
		t.Fatalf("alias = %q, want %q", got, "Official plugins")
	}
}

func TestConfirmNoWithoutDenyGoesBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	m.prev = scMenu
	m.screen = scConfirm
	m.confirmMsg = "sure?"

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = mm.(*model)

	if m.screen != scMenu {
		t.Fatalf("screen = %d, want scMenu", m.screen)
	}
}
