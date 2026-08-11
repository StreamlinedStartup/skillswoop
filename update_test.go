package main

import (
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
	for _, domain := range m.domains {
		for _, entry := range domain.entries {
			if entry.title == "Quit" {
				t.Fatal("Quit must not remain a menu action")
			}
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
	for _, want := range []string{"KEYS · SKILLS", "change section", "toggle update scope", "quit"} {
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
	for _, want := range []string{"mark skill", "star skill", "filter skills", "choose install destination"} {
		if !strings.Contains(view, want) {
			t.Fatalf("skill help missing %q", want)
		}
	}
}

func TestInstallSelectionsOpenProjectDestination(t *testing.T) {
	tests := []struct {
		name   string
		screen screen
		kind   installKind
		source string
		item   item
	}{
		{name: "skills", screen: scSkills, kind: installSkills, source: "owner/skills", item: item{id: "tdd", title: "tdd", sel: true}},
		{name: "starred skills", screen: scStarred, kind: installSkills, item: item{id: "review", title: "review", source: "owner/skills", sel: true}},
		{name: "plugins", screen: scPlugins, kind: installPlugins, source: "owner/plugins", item: item{id: "hooky", title: "hooky", flags: "codex,hooks", sel: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			m := newModel()
			m.global = true
			m.screen = tt.screen
			m.curSource = "stale/source"
			if tt.screen == scSkills {
				m.curSource = tt.source
			}
			m.curMarket = tt.source
			m.enterPicker(newPicker([]item{tt.item}, true))

			mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = mm.(*model)

			if cmd != nil {
				t.Fatal("selection started an install before destination confirmation")
			}
			if m.screen != scInstallDestination {
				t.Fatalf("screen = %d, want scInstallDestination", m.screen)
			}
			if m.install == nil {
				t.Fatal("install request was not recorded")
			}
			if m.install.kind != tt.kind || m.install.origin != tt.screen || m.install.source != tt.source {
				t.Fatalf("install request = %#v", m.install)
			}
			if m.install.global {
				t.Fatal("destination inherited the global update scope")
			}
		})
	}
}

func TestEmptyInstallSelectionDoesNotOpenDestination(t *testing.T) {
	for _, screen := range []screen{scSkills, scStarred, scPlugins} {
		t.Run(itoa(int(screen)), func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			m := newModel()
			m.screen = screen
			m.enterPicker(newPicker([]item{{id: "one", title: "one"}}, true))

			mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = mm.(*model)

			if cmd == nil {
				t.Fatal("empty selection did not return mark-first guidance")
			}
			if m.screen != screen || m.install != nil {
				t.Fatalf("empty selection changed install state: screen=%d request=%#v", m.screen, m.install)
			}
		})
	}
}

func TestInstallDestinationEscapePreservesSelectionState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.screen = scSkills
	m.curSource = "owner/skills"
	m.enterPicker(newPicker([]item{
		{id: "alpha", title: "alpha", sel: true},
		{id: "bravo", title: "bravo"},
		{id: "charlie", title: "charlie", sel: true},
	}, true))
	m.pick.cursor = 2
	m.pick.top = 1
	m.pick.filter = "a"
	wantPicker := m.pick

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(*model)

	if m.screen != scSkills {
		t.Fatalf("screen = %d, want scSkills", m.screen)
	}
	if m.pick != wantPicker || m.pick.cursor != 2 || m.pick.top != 1 || m.pick.filter != "a" {
		t.Fatalf("picker state changed: %#v", m.pick)
	}
	if got := m.pick.selectedCount(); got != 2 {
		t.Fatalf("selected count = %d, want 2", got)
	}
}

func TestInstallDestinationChoosesGlobalAndStartsInstall(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.screen = scInstallDestination
	m.install = &installRequest{
		kind:   installSkills,
		origin: scSkills,
		source: "owner/skills",
		items:  []item{{id: "alpha"}, {id: "bravo"}},
	}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = mm.(*model)
	if !m.install.global {
		t.Fatal("down did not choose the global destination")
	}

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(*model)
	if cmd == nil {
		t.Fatal("enter did not start the confirmed install")
	}
	if m.screen != scRunning {
		t.Fatalf("screen = %d, want scRunning", m.screen)
	}
	if m.install != nil {
		t.Fatalf("confirmed request was not cleared: %#v", m.install)
	}
	if got, want := m.busyTitle, "installing 2 skills globally"; got != want {
		t.Fatalf("busy title = %q, want %q", got, want)
	}
}

func TestInstallArgsUseRequestDestination(t *testing.T) {
	tests := []struct {
		name string
		req  installRequest
		want []string
	}{
		{
			name: "project skills",
			req:  installRequest{kind: installSkills, source: "owner/skills", items: []item{{id: "alpha"}}},
			want: []string{"use", "owner/skills", "--", "--skill", "alpha", "-y"},
		},
		{
			name: "global skills",
			req:  installRequest{kind: installSkills, global: true, source: "owner/skills", items: []item{{id: "alpha"}, {id: "bravo"}}},
			want: []string{"-g", "use", "owner/skills", "--", "--skill", "alpha", "--skill", "bravo", "-y"},
		},
		{
			name: "project plugins",
			req:  installRequest{kind: installPlugins, source: "owner/plugins", items: []item{{id: "plain"}}},
			want: []string{"plugin", "install", "owner/plugins", "plain"},
		},
		{
			name: "global plugins",
			req:  installRequest{kind: installPlugins, global: true, source: "owner/plugins", items: []item{{id: "plain"}}},
			want: []string{"-g", "plugin", "install", "owner/plugins", "plain"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := installArgs(tt.req); strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
				t.Fatalf("install args = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestHookPluginRequestRetainsDestinationThroughCheck(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	m.agents = "claude-code codex"
	req := installRequest{
		kind:   installPlugins,
		origin: scPlugins,
		source: "owner/plugins",
		global: true,
		items:  []item{{id: "hooky", flags: "codex,hooks"}},
	}

	cmd := m.executeInstall(req)
	if cmd == nil {
		t.Fatal("hook check command is nil")
	}
	if m.pendingInstall == nil || !m.pendingInstall.global {
		t.Fatalf("pending install lost destination: %#v", m.pendingInstall)
	}
	if got, want := installTitle(*m.pendingInstall), "1 plugin globally"; got != want {
		t.Fatalf("install title = %q, want %q", got, want)
	}
}

func TestTabDoesNothingInInstallFlowScreens(t *testing.T) {
	for _, screen := range []screen{scSources, scSkills, scStarred, scMarkets, scPlugins} {
		t.Run(itoa(int(screen)), func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			m := newModel()
			m.screen = screen
			m.enterPicker(newPicker([]item{{id: "one", title: "one"}}, screen == scSkills || screen == scStarred || screen == scPlugins))

			mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
			m = mm.(*model)

			if m.global {
				t.Fatalf("tab enabled update scope on screen %d", screen)
			}
		})
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
	m.pendingInstall = &installRequest{kind: installPlugins, origin: scPlugins, source: "owner/mkt", items: []item{{id: "hooky"}}}

	mm, _ = m.Update(codexHooksMsg{state: "off"})
	m = mm.(*model)

	if m.screen != scConfirm {
		t.Fatalf("screen = %d, want scConfirm", m.screen)
	}
	if m.denyCmd == nil {
		t.Fatal("expected a deny action for the hooks confirm")
	}
	if m.pendingInstall != nil {
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
	m.pendingInstall = &installRequest{kind: installPlugins, origin: scPlugins, source: "owner/mkt", items: []item{{id: "hooky"}}}

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
