package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---- cyberpunk palette --------------------------------------------------
var (
	cPink   = lipgloss.Color("#ff2e97")
	cMag    = lipgloss.Color("#ff6ac1")
	cCyan   = lipgloss.Color("#00f0ff")
	cCyan2  = lipgloss.Color("#22d3ee")
	cPurple = lipgloss.Color("#b16cff")
	cGreen  = lipgloss.Color("#39ff14")
	cYellow = lipgloss.Color("#fde047")
	cText   = lipgloss.Color("#e8e6ff")
	cDim    = lipgloss.Color("#8a87b8")
	cFaint  = lipgloss.Color("#54527a")
	cRed    = lipgloss.Color("#ff4d6d")
)

// gradient stops used for the banner + rules (pink -> purple -> cyan)
var gradStops = []string{"#ff2e97", "#ff5fd0", "#b16cff", "#7b8bff", "#22d3ee", "#00f0ff"}

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(cPurple).
			Padding(0, 1)

	tagStyle    = lipgloss.NewStyle().Foreground(cDim).Italic(true)
	helpKey     = lipgloss.NewStyle().Foreground(cCyan)
	helpDesc    = lipgloss.NewStyle().Foreground(cFaint)
	chipStyle   = lipgloss.NewStyle().Foreground(cYellow)
	scopeProj   = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	scopeGlob   = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	rowCursor   = lipgloss.NewStyle().Foreground(cText).Bold(true)
	rowNormal   = lipgloss.NewStyle().Foreground(cCyan2)
	rowDesc     = lipgloss.NewStyle().Foreground(cFaint)
	rowDescCur  = lipgloss.NewStyle().Foreground(cDim)
	barStyle    = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	checkOn     = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	checkOff    = lipgloss.NewStyle().Foreground(cFaint)
	titleStyle  = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	okStyle     = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	errStyle    = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	inputPrompt = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	scrollHint  = lipgloss.NewStyle().Foreground(cPurple)
)

// ---- gradient helpers ---------------------------------------------------
func hexRGB(h string) (int, int, int) {
	h = strings.TrimPrefix(h, "#")
	var r, g, b int
	fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func lerp(a, b, t float64) int { return int(a + (b-a)*t + 0.5) }

// colorAt returns a hex color sampled from the gradient at ratio p in [0,1].
func colorAt(p float64) lipgloss.Color {
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	seg := p * float64(len(gradStops)-1)
	i := int(seg)
	if i >= len(gradStops)-1 {
		return lipgloss.Color(gradStops[len(gradStops)-1])
	}
	t := seg - float64(i)
	r1, g1, b1 := hexRGB(gradStops[i])
	r2, g2, b2 := hexRGB(gradStops[i+1])
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
		lerp(float64(r1), float64(r2), t),
		lerp(float64(g1), float64(g2), t),
		lerp(float64(b1), float64(b2), t)))
}

// gradient renders s with a per-rune horizontal gradient.
func gradient(s string) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return s
	}
	var b strings.Builder
	for i, r := range runes {
		p := 0.0
		if n > 1 {
			p = float64(i) / float64(n-1)
		}
		b.WriteString(lipgloss.NewStyle().Foreground(colorAt(p)).Bold(true).Render(string(r)))
	}
	return b.String()
}

// neonRule draws a full-width gradient rule line.
func neonRule(width int) string {
	if width < 1 {
		width = 1
	}
	return gradient(strings.Repeat("═", width))
}

// banner: the cyberpunk wordmark used in the title bar.
func banner(width int) string {
	mark := gradient("▓▒░  S W O O P  ░▒▓")
	tag := tagStyle.Render("// skillswoop · swoop skills into claude + codex")
	rule := neonRule(width)
	header := lipgloss.JoinVertical(lipgloss.Left, mark+"   "+tag, rule)
	return header
}

// truncate to a max display width (rune-based), adding an ellipsis.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

// padLines pads a block of text to exactly h lines (prevents panel jitter).
func padLines(s string, h int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}
