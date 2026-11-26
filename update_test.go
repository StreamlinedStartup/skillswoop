package main

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestMenuTabDoesNothing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	before := stripANSI(mm.View())

	mm, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = mm.(*model)
	if cmd != nil {
		t.Fatal("tab returned a command")
	}
	if m.screen != scMenu || m.domain != 0 || m.menu.cursor != 0 {
		t.Fatalf("tab changed menu state to (%d, %d, %d)", m.screen, m.domain, m.menu.cursor)
	}
	if after := stripANSI(m.View()); after != before {
		t.Fatalf("tab changed the view:\n%s", after)
	}
}

func TestMenuUpdateEntriesRunTheirCommands(t *testing.T) {
	stub := filepath.Join(t.TempDir(), "swoop-core")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho \"args: $*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SWOOP_CORE", stub)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := []struct{ title, args string }{
		{"Update this folder", "args: update"},
		{"Update global", "args: -g update"},
		{"Update everything", "args: update --all"},
	}
	var got []menuEntry
	for _, e := range menuDomains()[0].entries {
		if strings.HasPrefix(e.title, "Update") {
			got = append(got, e)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("skills section has %d update entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].title != w.title {
			t.Fatalf("update entry %d = %q, want %q", i, got[i].title, w.title)
		}
		m := newModel()
		_, cmd := got[i].act(m)
		if m.screen != scRunning {
			t.Fatalf("%s did not start running", w.title)
		}
		msg, ok := cmd().(opDoneMsg)
		if !ok || msg.err != nil || strings.TrimSpace(msg.output) != w.args {
			t.Fatalf("%s ran %q (err %v), want %q", w.title, msg.output, msg.err, w.args)
		}
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
	for _, want := range []string{"KEYS · SKILLS", "change section", "quit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("help view missing %q", want)
		}
	}
	for _, gone := range []string{"update scope", "PROJECT", "GLOBAL"} {
		if strings.Contains(view, gone) {
			t.Fatalf("help view still mentions %q", gone)
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

			mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
			m = mm.(*model)

			if cmd != nil || m.screen != screen || m.install != nil {
				t.Fatalf("tab changed state on screen %d", screen)
			}
		})
	}
}

func TestSourcesActionOpensManagementMenu(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})

	for i := 0; i < 6; i++ {
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

func TestLibraryShownOnceAndNotRemovable(t *testing.T) {
	config, data := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_DATA_HOME", data)
	lib := filepath.Join(data, "swoop", "library")
	if err := os.MkdirAll(filepath.Join(config, "swoop"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(lib, "my-skill") + "\n" + filepath.Join(lib, "other") + "\nowner/repo\n"
	if err := os.WriteFile(filepath.Join(config, "swoop", "sources"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	items := sourceItems()
	if len(items) != 2 || items[0].id != lib || items[0].title != "Library" || items[1].id != "owner/repo" {
		t.Fatalf("install source list = %#v, want Library then owner/repo", items)
	}

	m := newModel()
	actRemove(m)
	for _, it := range m.pick.items {
		if inLibrary(it.id) {
			t.Fatalf("remove list offers the Library: %#v", m.pick.items)
		}
	}
	if len(m.pick.items) != 1 {
		t.Fatalf("remove list = %#v, want only owner/repo", m.pick.items)
	}
	if got := short(filepath.Join(lib, "my-skill")); got != "Library" {
		t.Fatalf("legacy star source label = %q, want Library", got)
	}
}

func TestLegacyLibraryStarIsTheLibraryStar(t *testing.T) {
	config, data := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_DATA_HOME", data)
	lib := filepath.Join(data, "swoop", "library")
	if err := os.MkdirAll(filepath.Join(lib, "foo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "foo", "SKILL.md"), []byte("---\nname: foo\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(lib, "collections", "team")
	if err := os.MkdirAll(filepath.Join(config, "swoop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(starsPath(), []byte(filepath.Join(lib, "foo")+"\tfoo\tdemo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "swoop", "sources"), []byte(nested+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items := applyStars([]item{{id: "foo", title: "foo"}}, libraryDir())
	if !items[0].star {
		t.Fatalf("legacy star should show on the Library skill")
	}
	if on, err := toggleStar(items[0]); err != nil || on {
		t.Fatalf("toggle should unstar the legacy star: on=%v err=%v", on, err)
	}
	if got := loadStars(); len(got) != 0 {
		t.Fatalf("stars after unstar = %#v, want none", got)
	}

	src := sourceItems()
	if len(src) != 2 || src[1].id != nested {
		t.Fatalf("nested folder under the Library should stay a source: %#v", src)
	}
	m := newModel()
	actRemove(m)
	if len(m.pick.items) != 1 || m.pick.items[0].id != nested {
		t.Fatalf("nested source should be removable: %#v", m.pick.items)
	}
}

// clashModel points the TUI at the repo engine with a temp config, data and HOME.
func clashModel(t *testing.T) (*model, string, string) {
	t.Helper()
	root := t.TempDir()
	eng, err := filepath.Abs("engine/swoop-core")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SWOOP_CORE", eng)
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	writeSkill(t, filepath.Join(root, "lib"), "notes", "Library copy.")
	if out, err := core("add", filepath.Join(root, "lib", "skills", "notes")); err != nil {
		t.Fatalf("seed add failed: %v\n%s", err, out)
	}
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	return mm.(*model), root, libraryDir()
}

// step feeds a message and runs any returned command once, like the runtime.
func step(t *testing.T, m *model, msg tea.Msg) *model {
	t.Helper()
	mm, cmd := m.Update(msg)
	m = mm.(*model)
	if m.screen == scRunning && cmd != nil { // only engine runs, never input blink ticks
		mm, _ = m.Update(cmd())
		m = mm.(*model)
	}
	return m
}

func typeText(t *testing.T, m *model, s string) *model {
	for _, r := range s {
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) // skip the cursor-blink cmd
		m = mm.(*model)
	}
	return m
}

func TestAddClashScreenRenameIncoming(t *testing.T) {
	m, root, lib := clashModel(t)
	writeSkill(t, filepath.Join(root, "other"), "notes", "Incoming copy.")
	incoming := filepath.Join(root, "other", "skills", "notes")

	msg := addSourceCmd(incoming)()
	cm, ok := msg.(clashMsg)
	if !ok || len(cm.clashes) != 1 || cm.clashes[0].name != "notes" {
		t.Fatalf("add should report a Name clash, got %#v", msg)
	}
	m = step(t, m, cm)
	if m.screen != scClash {
		t.Fatalf("screen = %d, want scClash", m.screen)
	}
	m.layout()
	body := m.buildBody()
	for _, want := range clashChoices {
		if !strings.Contains(body, want) {
			t.Fatalf("clash screen missing %q:\n%s", want, body)
		}
	}

	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // rename incoming
	if !m.clashNaming {
		t.Fatal("expected the name input")
	}
	m = typeText(t, m, "notes")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != scClash || !strings.Contains(m.clashErr, "already has") {
		t.Fatalf("taken name should be rejected, err=%q screen=%d", m.clashErr, m.screen)
	}
	m.input.SetValue("")
	m = typeText(t, m, "notes-two")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.screen != scResult || m.resultErr {
		t.Fatalf("screen = %d err = %v, want a clean result", m.screen, m.resultErr)
	}
	if b, _ := os.ReadFile(filepath.Join(lib, "notes-two", "SKILL.md")); !strings.Contains(string(b), "name: notes-two\n") {
		t.Fatalf("incoming skill not renamed:\n%s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(lib, "notes", "SKILL.md")); !strings.Contains(string(b), "Library copy.") {
		t.Fatal("Library skill must keep its name")
	}
}

func TestAddClashScreenSkipChangesNothing(t *testing.T) {
	m, root, lib := clashModel(t)
	writeSkill(t, filepath.Join(root, "other"), "notes", "Incoming copy.")
	m = step(t, m, addSourceCmd(filepath.Join(root, "other", "skills", "notes"))())
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != scResult {
		t.Fatalf("screen = %d, want scResult", m.screen)
	}
	ents, _ := os.ReadDir(lib)
	if len(ents) != 1 {
		t.Fatalf("skip must not add anything; Library has %d entries", len(ents))
	}
}

func TestTidyClashScreenRenameLibrarySkill(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required to read the Library skill name for Stars")
	}
	m, root, lib := clashModel(t)
	global := filepath.Join(root, "home", ".claude")
	writeSkill(t, global, "notes", "Global copy.")
	writeSkill(t, global, "fresh", "No clash.")

	msg := tidyCmd()()
	cm, ok := msg.(clashMsg)
	if !ok || len(cm.clashes) != 1 || cm.clashes[0].mode != "stash" {
		t.Fatalf("tidy should report one Name clash, got %#v", msg)
	}
	m = step(t, m, cm)
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown}) // rename the Library skill
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = typeText(t, m, "notes-lib")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.screen != scResult || m.resultErr {
		t.Fatalf("screen = %d err = %v, want a clean result", m.screen, m.resultErr)
	}
	if b, _ := os.ReadFile(filepath.Join(lib, "notes-lib", "SKILL.md")); !strings.Contains(string(b), "name: notes-lib\n") || !strings.Contains(string(b), "Library copy.") {
		t.Fatalf("Library skill not renamed:\n%s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(lib, "notes", "SKILL.md")); !strings.Contains(string(b), "Global copy.") {
		t.Fatal("global skill should take the original name")
	}
	if _, err := os.Stat(filepath.Join(global, "skills", "notes")); !os.IsNotExist(err) {
		t.Fatal("Tidy should move the global skill out")
	}
	if _, err := os.Stat(filepath.Join(lib, "fresh")); err != nil {
		t.Fatal("non-clashing skill should be moved")
	}
}

func TestClashNameWithSpacesIsRejected(t *testing.T) {
	m, root, lib := clashModel(t)
	writeSkill(t, filepath.Join(root, "other"), "notes", "Incoming copy.")
	m = step(t, m, addSourceCmd(filepath.Join(root, "other", "skills", "notes"))())
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.input.SetValue(" notes-two ")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != scClash || !m.clashNaming || m.clashErr == "" {
		t.Fatalf("spaced name should keep the prompt open: screen=%d naming=%v err=%q", m.screen, m.clashNaming, m.clashErr)
	}
	if _, err := os.Stat(filepath.Join(lib, "notes-two")); err == nil {
		t.Fatal("nothing should be renamed")
	}
}

func TestTidyClashSummaryCountsChoices(t *testing.T) {
	m, root, _ := clashModel(t)
	writeSkill(t, filepath.Join(root, "lib2"), "memo", "Library memo.")
	if out, err := core("add", filepath.Join(root, "lib2", "skills", "memo")); err != nil {
		t.Fatalf("seed add failed: %v\n%s", err, out)
	}
	global := filepath.Join(root, "home", ".claude")
	writeSkill(t, global, "memo", "Global memo.")
	writeSkill(t, global, "notes", "Global notes.")
	writeSkill(t, global, "fresh", "No clash.")

	m = step(t, m, tidyCmd()())
	if len(m.clashes) != 2 {
		t.Fatalf("want 2 clashes, got %d", len(m.clashes))
	}
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // memo: rename incoming
	m = typeText(t, m, "memo-global")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEsc}) // notes: skip
	if m.screen != scResult {
		t.Fatalf("screen = %d, want scResult", m.screen)
	}
	body := m.vp.View()
	if !strings.Contains(body, "Done. moved 2 · unlinked 0 · not moved 1 · failed 0") {
		t.Fatalf("final summary should count the rename and the skip:\n%s", body)
	}
	if strings.Count(body, "Done.") != 1 || strings.Contains(body, "Run `swoop stash`") || !strings.Contains(body, "skipped: notes") {
		t.Fatalf("stale first-pass summary in result:\n%s", body)
	}
}

// VRTS-7: Enter on "Remove sources" asks first when a marked Source has Stars.
func removeScreenWith(t *testing.T, stars []item, sources ...string) *model {
	t.Helper()
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	if err := os.MkdirAll(filepath.Join(config, "swoop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "swoop", "sources"), []byte(strings.Join(sources, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveStars(stars); err != nil {
		t.Fatal(err)
	}
	m := newModel()
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 96, Height: 30})
	m = mm.(*model)
	actRemove(m)
	return m
}

func pressKey(m *model, k string) (*model, tea.Cmd) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	}
	mm, cmd := m.Update(msg)
	return mm.(*model), cmd
}

func TestRemoveSourcesWithStarsConfirmsFirst(t *testing.T) {
	stars := []item{
		{id: "tdd", source: "owner/starred"},
		{id: "review", source: "owner/kept"},
	}
	m := removeScreenWith(t, stars, "owner/starred", "owner/kept")
	m, _ = pressKey(m, " ") // mark owner/starred
	m, cmd := pressKey(m, "enter")
	if m.screen != scConfirm || cmd != nil {
		t.Fatalf("screen = %d (cmd %v), want scConfirm and no command yet", m.screen, cmd != nil)
	}
	if !strings.Contains(m.confirmMsg, "tdd  (owner/starred)") || strings.Contains(m.confirmMsg, "review") {
		t.Fatalf("confirm should list only the marked source's stars:\n%s", m.confirmMsg)
	}

	for _, k := range []string{"esc", "n"} {
		m, cmd = pressKey(m, k)
		if m.screen != scRemove || cmd != nil {
			t.Fatalf("%q: screen = %d, want scRemove", k, m.screen)
		}
		if got := len(loadStars()); got != 2 {
			t.Fatalf("%q: stars changed, got %d", k, got)
		}
		if len(m.pick.selected()) != 1 {
			t.Fatalf("%q: marks lost", k)
		}
		m, _ = pressKey(m, "enter")
	}

	m, cmd = pressKey(m, "y")
	if m.screen != scRunning || cmd == nil {
		t.Fatalf("yes: screen = %d, want scRunning with a command", m.screen)
	}
}

func TestRemoveSourcesWithoutStarsRemovesAtOnce(t *testing.T) {
	m := removeScreenWith(t, []item{{id: "review", source: "owner/kept"}}, "owner/plain", "owner/kept")
	m, _ = pressKey(m, " ") // mark owner/plain
	m, cmd := pressKey(m, "enter")
	if m.screen != scRunning || cmd == nil {
		t.Fatalf("screen = %d, want scRunning with a command", m.screen)
	}
}
