package main

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuDomainsExposeApprovedActions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()

	if got, want := len(m.domains), 3; got != want {
		t.Fatalf("domain count = %d, want %d", got, want)
	}
	want := map[string][]string{
		"SKILLS": {
			"Install skills",
			"Starred skills",
			"Update this folder",
			"Update every folder",
			"Browse skills.sh",
			"Sources…",
		},
		"PLUGINS": {
			"Install plugins",
			"Remove plugins",
			"Add a marketplace",
			"Marketplaces…",
		},
		"SETTINGS": {
			"Default agents",
			"Tidy global skills",
		},
	}
	for _, domain := range m.domains {
		var titles []string
		for _, entry := range domain.entries {
			titles = append(titles, entry.title)
			if entry.title == "Quit" {
				t.Fatal("Quit must not remain a menu action")
			}
		}
		if !reflect.DeepEqual(titles, want[domain.label]) {
			t.Fatalf("%s actions = %v, want %v", domain.label, titles, want[domain.label])
		}
	}
}

func TestMenuDomainNavigationRemembersEachCursor(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = mm.(*model)
	if m.domain != 1 || m.menu.cursor != 0 {
		t.Fatalf("plugin domain state = (%d, %d), want (1, 0)", m.domain, m.menu.cursor)
	}

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = mm.(*model)
	if m.domain != 0 || m.menu.cursor != 2 {
		t.Fatalf("restored skills state = (%d, %d), want (0, 2)", m.domain, m.menu.cursor)
	}

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = mm.(*model)
	if m.domain != 1 || m.menu.cursor != 1 {
		t.Fatalf("restored plugins state = (%d, %d), want (1, 1)", m.domain, m.menu.cursor)
	}
}

func TestMenuTabStillTogglesScope(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = mm.(*model)
	if !m.global {
		t.Fatal("tab did not enable global scope")
	}
	if m.domain != 0 {
		t.Fatalf("tab changed domain to %d", m.domain)
	}
}

func TestQuestionMarkOpensAndEscapeClosesContextualHelp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = mm.(*model)
	if !m.showHelp {
		t.Fatal("question mark did not open help")
	}
	view := stripANSI(m.View())
	for _, want := range []string{"KEYS · SKILLS", "change section", "toggle scope", "quit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("help view missing %q", want)
		}
	}

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(*model)
	if m.showHelp {
		t.Fatal("escape did not close help")
	}
	if m.screen != scMenu {
		t.Fatalf("help changed screen to %d", m.screen)
	}
}

func TestHelpIsContextualForSkillPicker(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.curSource = "owner/repo"
	m.enterPicker(newPicker([]item{{id: "tdd", title: "tdd"}}, true))
	m.screen = scSkills
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	view := stripANSI(mm.View())
	for _, want := range []string{"mark skill", "star skill", "filter skills", "install marked"} {
		if !strings.Contains(view, want) {
			t.Fatalf("skill help missing %q", want)
		}
	}
}

func TestSourcesActionOpensManagementMenu(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	for i := 0; i < 5; i++ {
		mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(*model)
	if m.screen != scSourceActions {
		t.Fatalf("screen = %d, want scSourceActions", m.screen)
	}
	if got, want := m.pick.len(), 2; got != want {
		t.Fatalf("source action count = %d, want %d", got, want)
	}
	if current, ok := m.pick.current(); !ok || !strings.Contains(current.title, "Add a source") {
		t.Fatalf("first source action = %#v", current)
	}
}

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
