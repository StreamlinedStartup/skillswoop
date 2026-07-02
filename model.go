package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	scMenu          screen = iota
	scSources              // install: pick a source
	scSkills               // install: multi-select skills
	scStarred              // install: multi-select starred skills
	scBrowseInput          // browse: query
	scBrowseResults        // browse: multi-select repos
	scAdd                  // add a source
	scAgents               // edit default agents
	scRemove               // remove sources (multi)
	scRename               // give a source a friendly display name
	scConfirm              // generic yes/no
	scRunning              // spinner while an op runs
	scResult               // scrollable output of an op
	scMarkets              // plugins: pick a marketplace
	scPlugins              // plugins: multi-select plugins
	scPluginRemove         // plugins: multi-select installed plugins to remove
)

// menu entries
type menuEntry struct {
	icon  string
	title string
	desc  string
	act   func(m *model) (tea.Model, tea.Cmd)
}

type model struct {
	width, height  int
	innerW, innerH int

	screen screen
	prev   screen

	menu    *picker
	pick    *picker // active picker for sources/skills/browse/remove
	input   textinput.Model
	spin    spinner.Model
	vp      viewport.Model
	vpReady bool

	entries []menuEntry

	global         bool     // scope toggle (project default)
	curSource      string   // source being drilled into
	curMarket      string   // marketplace being drilled into (scPlugins)
	renameURL      string   // source whose alias is being edited (scRename)
	filtering      bool     // true while slash-filtering repo skills/plugins
	addMarketplace bool     // scAdd doubles as the add-marketplace input
	pendingInstall []string // plugin-install args parked while _codex_hooks runs
	busyTitle      string
	resultTitle    string
	resultErr      bool
	confirmMsg     string
	confirmCmd     func(m *model) tea.Cmd
	denyCmd        func(m *model) tea.Cmd // optional "no" action in scConfirm (nil = just go back)
	flash          string                 // transient status note

	agents string
}

// ---- messages -----------------------------------------------------------
type sourcesMsg struct{ items []item }
type skillsMsg struct {
	items []item
	err   error
}
type searchMsg struct {
	items []item
	err   error
}
type starredMsg struct{ items []item }
type marketsMsg struct{ items []item }
type pluginsMsg struct {
	items []item
	err   error
}
type installedPluginsMsg struct {
	items []item
	err   error
}
type codexHooksMsg struct{ state string }
type opDoneMsg struct {
	title  string
	output string
	err    error
}
type flashMsg string
type clearFlashMsg struct{}

func newModel() *model {
	sp := spinner.New()
	sp.Spinner = spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    time.Second / 12,
	}
	sp.Style = titleStyle

	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.PromptStyle = inputPrompt
	ti.TextStyle = rowCursor
	ti.Cursor.Style = barStyle
	ti.CharLimit = 200

	m := &model{
		screen: scMenu,
		spin:   sp,
		input:  ti,
		agents: loadAgents(),
	}
	m.entries = menuEntries()
	items := make([]item, len(m.entries))
	for i, e := range m.entries {
		// pad the icon to a uniform 2 cells so the titles line up regardless of glyph width
		items[i] = item{id: e.title, title: padRight(e.icon, 2) + "  " + e.title, desc: e.desc}
	}
	m.menu = newPicker(items, false)
	return m
}

func (m *model) Init() tea.Cmd { return m.spin.Tick }

// ---- commands -----------------------------------------------------------
func loadSourcesCmd() tea.Cmd {
	return func() tea.Msg { return sourcesMsg{sourceItems()} }
}

func loadSkillsCmd(src string) tea.Cmd {
	return func() tea.Msg {
		items, err := listSkills(src)
		return skillsMsg{items, err}
	}
}

func loadStarredCmd() tea.Cmd {
	return func() tea.Msg { return starredMsg{starredItems()} }
}

func loadMarketsCmd() tea.Cmd {
	return func() tea.Msg { return marketsMsg{marketItems()} }
}

func loadPluginsCmd(src string) tea.Cmd {
	return func() tea.Msg {
		items, err := listPlugins(src)
		return pluginsMsg{items, err}
	}
}

func loadInstalledPluginsCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := listInstalledPlugins()
		return installedPluginsMsg{items, err}
	}
}

func codexHooksCmd() tea.Cmd {
	return func() tea.Msg { return codexHooksMsg{codexHooksState()} }
}

func searchCmd(q string) tea.Cmd {
	return func() tea.Msg {
		items, err := searchSkills(q)
		return searchMsg{items, err}
	}
}

// opCmd runs an engine command and reports the captured output.
func opCmd(title string, args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := core(args...)
		return opDoneMsg{title: title, output: stripANSI(out), err: err}
	}
}

func flashFor(s string, d time.Duration) tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return flashMsg(s) },
		tea.Tick(d, func(time.Time) tea.Msg { return clearFlashMsg{} }),
	)
}
