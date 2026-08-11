package main

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	scMenu               screen = iota
	scSourceActions             // source management actions
	scSources                   // install: pick a source
	scSkills                    // install: multi-select skills
	scStarred                   // install: multi-select starred skills
	scBrowseInput               // browse: query
	scBrowseResults             // browse: multi-select repos
	scAdd                       // add a source
	scAgents                    // edit default agents
	scRemove                    // remove sources (multi)
	scRename                    // give a source a friendly display name
	scInstallDestination        // install: confirm project or global destination
	scConfirm                   // generic yes/no
	scRunning                   // spinner while an op runs
	scResult                    // scrollable output of an op
	scMarkets                   // plugins: pick a marketplace
	scPlugins                   // plugins: multi-select plugins
	scPluginRemove              // plugins: multi-select installed plugins to remove
)

// menu entries
type menuEntry struct {
	icon  string
	title string
	desc  string
	act   func(m *model) (tea.Model, tea.Cmd)
}

type menuDomain struct {
	label   string
	entries []menuEntry
}

type installKind int

const (
	installSkills installKind = iota
	installPlugins
)

type installRequest struct {
	kind   installKind
	items  []item
	source string
	origin screen
	global bool
}

type model struct {
	width, height  int
	innerW, innerH int

	screen screen
	prev   screen

	menu          *picker
	pick          *picker // active picker for sources/skills/browse/remove
	domains       []menuDomain
	domain        int
	domainCursors []int
	input         textinput.Model
	spin          spinner.Model
	vp            viewport.Model
	vpReady       bool

	global         bool // update scope toggle (project default)
	install        *installRequest
	projectDir     string
	curSource      string          // source being drilled into
	curMarket      string          // marketplace being drilled into (scPlugins)
	renameURL      string          // source whose alias is being edited (scRename)
	filtering      bool            // true while slash-filtering repo skills/plugins
	showHelp       bool            // contextual keyboard help overlay
	addMarketplace bool            // scAdd doubles as the add-marketplace input
	pendingInstall *installRequest // plugin request parked while _codex_hooks runs
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
	projectDir, err := os.Getwd()
	if err != nil {
		projectDir = "current directory unavailable"
	}

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
		screen:     scMenu,
		spin:       sp,
		input:      ti,
		agents:     loadAgents(),
		projectDir: projectDir,
	}
	m.domains = menuDomains()
	m.domainCursors = make([]int, len(m.domains))
	m.setDomain(0)
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
