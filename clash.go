package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// A Name clash: a skill entering the Library (by add or Tidy) has the same
// folder name as a different skill already there. The engine refuses without
// a terminal; this screen asks the person instead, one clash at a time.
type clash struct {
	mode string // "add" (copy in) or "stash" (move in)
	path string // incoming skill folder
	name string // folder name shared with the Library skill
}

var clashChoices = []string{"Rename the incoming skill", "Rename the Library skill", "Skip"}

// clashMsg hands the TUI the clashes left after an add or Tidy run, plus the
// engine output so far. tidy holds Tidy's counts with its summary lines taken
// out of log, so the final summary can include the person's choices.
type clashMsg struct {
	title   string
	log     string
	clashes []clash
	tidy    *tidyCounts
}

type tidyCounts struct{ moved, unlinked, notMoved, failed int }

// tidySummaryRE matches the engine's stash summary line.
var tidySummaryRE = regexp.MustCompile(`^Done\. moved (\d+) · unlinked (\d+) · not moved (\d+) · failed (\d+)$`)

// splitTidySummary takes the summary and Name clash lines out of Tidy's output
// and returns the counts; ok is false when the summary line is missing.
func splitTidySummary(out string) (string, tidyCounts, bool) {
	var keep []string
	var c tidyCounts
	ok := false
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if mm := tidySummaryRE.FindStringSubmatch(ln); mm != nil {
			c.moved, _ = strconv.Atoi(mm[1])
			c.unlinked, _ = strconv.Atoi(mm[2])
			c.notMoved, _ = strconv.Atoi(mm[3])
			c.failed, _ = strconv.Atoi(mm[4])
			ok = true
			continue
		}
		if strings.HasPrefix(ln, "Name clash: ") || strings.HasPrefix(ln, "Run `swoop stash` in a terminal") {
			continue
		}
		keep = append(keep, ln)
	}
	return strings.Join(keep, "\n"), c, ok
}

func (c tidyCounts) String() string {
	return "Done. moved " + itoa(c.moved) + " · unlinked " + itoa(c.unlinked) +
		" · not moved " + itoa(c.notMoved) + " · failed " + itoa(c.failed)
}

// clashStepMsg reports one _resolve run.
type clashStepMsg struct {
	output string
	err    error
}

func parseClashes(mode, out string) []clash {
	var cs []clash
	for _, ln := range strings.Split(out, "\n") {
		f := strings.SplitN(ln, "\t", 2)
		if len(f) == 2 && f[0] != "" && f[1] != "" {
			cs = append(cs, clash{mode: mode, path: f[0], name: f[1]})
		}
	}
	return cs
}

// addSourceCmd asks the engine for a Name clash before adding, so a clash
// opens the choice screen instead of the engine's refusal.
func addSourceCmd(src string) tea.Cmd {
	return func() tea.Msg {
		out, err := core("_clashes", "add", src)
		if err == nil {
			if cs := parseClashes("add", out); len(cs) > 0 {
				return clashMsg{title: "added source", clashes: cs}
			}
		}
		out, err = core("add", src)
		return opDoneMsg{title: "added source", output: stripANSI(out), err: err}
	}
}

// tidyCmd runs Tidy (which moves everything that does not clash), then
// collects the clashing skills it left in place.
func tidyCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := core("stash")
		if err != nil {
			return opDoneMsg{title: "tidy global skills", output: stripANSI(out), err: err}
		}
		cl, cerr := core("_clashes", "stash")
		if cerr != nil {
			return opDoneMsg{title: "tidy global skills", output: stripANSI(out + cl), err: cerr}
		}
		if cs := parseClashes("stash", cl); len(cs) > 0 {
			log, counts, ok := splitTidySummary(stripANSI(out))
			if !ok {
				return opDoneMsg{title: "tidy global skills", output: stripANSI(out), err: errors.New("tidy summary line missing from engine output")}
			}
			return clashMsg{title: "tidy global skills", log: log, clashes: cs, tidy: &counts}
		}
		return opDoneMsg{title: "tidy global skills", output: stripANSI(out)}
	}
}

func resolveClashCmd(c clash, how, name string) tea.Cmd {
	return func() tea.Msg {
		out, err := core("_resolve", c.mode, how, name, c.path)
		return clashStepMsg{output: stripANSI(out), err: err}
	}
}

var newNameRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// newNameError mirrors the engine's name_ok so the person is asked again
// before anything runs (the engine checks again).
func newNameError(name string) string {
	switch {
	case name == "":
		return "the new name cannot be empty"
	case !newNameRE.MatchString(name):
		return "use only lowercase letters, digits, and hyphens"
	}
	if _, err := os.Lstat(filepath.Join(libraryDir(), name)); err == nil {
		return "the Library already has a skill named " + name
	}
	return ""
}

func (m *model) startClashes(msg clashMsg) {
	m.clashes = msg.clashes
	m.clashTitle = msg.title
	m.clashLog = strings.TrimRight(msg.log, "\n")
	m.clashFailed = false
	m.clashTidy = msg.tidy
	m.clashSkipped = nil
	m.beginClash()
}

func (m *model) beginClash() {
	m.screen = scClash
	m.clashChoice = 0
	m.clashNaming = false
	m.clashErr = ""
	m.input.Blur()
}

func (m *model) logClash(s string) {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return
	}
	if m.clashLog != "" {
		m.clashLog += "\n"
	}
	m.clashLog += s
}

// nextClash drops the current clash and shows the next one, or the result.
func (m *model) nextClash() {
	m.clashes = m.clashes[1:]
	if len(m.clashes) > 0 {
		m.beginClash()
		return
	}
	if t := m.clashTidy; t != nil {
		m.logClash(t.String())
		if len(m.clashSkipped) > 0 {
			m.logClash("Name clash, skipped: " + strings.Join(m.clashSkipped, " "))
		}
		m.clashTidy = nil
	}
	m.screen = scResult
	m.resultTitle = m.clashTitle
	m.resultErr = m.clashFailed
	out := m.clashLog
	if out == "" {
		out = "done."
	}
	m.setResult(out)
}

func (m *model) onClashKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.clashes) == 0 {
		m.screen = scMenu
		return m, nil
	}
	c := m.clashes[0]
	k := msg.String()
	if m.clashNaming {
		switch k {
		case "esc":
			m.clashNaming = false
			m.clashErr = ""
			m.input.Blur()
			return m, nil
		case "enter":
			name := m.input.Value() // validated as typed: spaces are not trimmed away
			if e := newNameError(name); e != "" {
				m.clashErr = e
				return m, nil
			}
			how := "incoming"
			if m.clashChoice == 1 {
				how = "library"
			}
			m.busyTitle = "renaming " + c.name
			m.screen = scRunning
			return m, resolveClashCmd(c, how, name)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch k {
	case "up", "k":
		if m.clashChoice > 0 {
			m.clashChoice--
		}
	case "down", "j":
		if m.clashChoice < len(clashChoices)-1 {
			m.clashChoice++
		}
	case "esc", "q":
		m.clashChoice = 2
		fallthrough
	case "enter", " ":
		if m.clashChoice == 2 {
			m.logClash("  skipped " + c.name + " (Name clash - nothing changed)")
			m.clashSkipped = append(m.clashSkipped, c.name)
			m.nextClash()
			return m, nil
		}
		m.clashNaming = true
		m.clashErr = ""
		m.input.Placeholder = "new name: lowercase letters, digits, hyphens"
		m.input.SetValue("")
		m.input.Focus()
	}
	return m, nil
}

func (m *model) onClashStep(msg clashStepMsg) {
	if t := m.clashTidy; t != nil {
		t.notMoved--
		if msg.err != nil {
			t.failed++
		} else {
			t.moved++
		}
	}
	if msg.err != nil {
		m.clashFailed = true
		if strings.TrimSpace(msg.output) == "" {
			msg.output = "  failed: " + msg.err.Error()
		}
	}
	m.logClash(msg.output)
	m.nextClash()
}

func (m *model) clashBody() string {
	if len(m.clashes) == 0 {
		return heading("NAME CLASH", "")
	}
	c := m.clashes[0]
	sub := ""
	if len(m.clashes) > 1 {
		sub = itoa(len(m.clashes)) + " left"
	}
	lines := []string{
		heading("NAME CLASH", sub),
		"",
		rowCursor.Render(c.name) + rowDesc.Render(": the Library already has a different skill with this name."),
		rowDesc.Render(truncate("incoming: "+c.path, m.innerW)),
		"",
	}
	if m.clashNaming {
		what := "incoming skill"
		if m.clashChoice == 1 {
			what = "Library skill"
		}
		lines = append(lines, rowDescCur.Render("New name for the "+what+" "+c.name), m.input.View())
		if m.clashErr != "" {
			lines = append(lines, "", errStyle.Render(m.clashErr))
		}
		return strings.Join(lines, "\n")
	}
	for i, ch := range clashChoices {
		lines = append(lines, destinationRow(i == m.clashChoice, ch))
	}
	return strings.Join(lines, "\n")
}
