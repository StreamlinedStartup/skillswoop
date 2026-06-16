package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type item struct {
	id     string
	title  string
	desc   string
	source string
	sel    bool
	star   bool
}

// picker is a self-contained, windowed, optionally multi-select list.
// It owns its own scroll math so long lists never smear (the bug we hit in gum).
type picker struct {
	items   []item
	visible []int
	filter  string
	cursor  int
	top     int // index of first visible row
	width   int // inner content width
	height  int // visible rows
	multi   bool
	titleW  int // widest title (display cells) — used to align the description column
}

func newPicker(items []item, multi bool) *picker {
	p := &picker{items: items, multi: multi}
	p.rebuildVisible()
	return p
}

func (p *picker) rebuildVisible() {
	p.visible = rankedMatches(p.items, p.filter)
	p.titleW = 0
	for _, idx := range p.visible {
		it := p.items[idx]
		if w := lipgloss.Width(it.title); w > p.titleW {
			p.titleW = w
		}
	}
	if p.cursor >= len(p.visible) {
		p.cursor = len(p.visible) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.clampWindow()
}

func (p *picker) setFilter(q string) {
	p.filter = q
	p.cursor = 0
	p.top = 0
	p.rebuildVisible()
}

func (p *picker) setSize(w, h int) {
	p.width = w
	if h < 1 {
		h = 1
	}
	p.height = h
	p.clampWindow()
}

func (p *picker) len() int { return len(p.visible) }

func (p *picker) move(d int) {
	if len(p.visible) == 0 {
		return
	}
	p.cursor += d
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.visible) {
		p.cursor = len(p.visible) - 1
	}
	p.clampWindow()
}

func (p *picker) home() { p.cursor = 0; p.clampWindow() }
func (p *picker) end()  { p.cursor = len(p.visible) - 1; p.clampWindow() }

// clampWindow keeps the cursor inside the visible window [top, top+height).
func (p *picker) clampWindow() {
	if p.height < 1 {
		p.height = 1
	}
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+p.height {
		p.top = p.cursor - p.height + 1
	}
	if p.top < 0 {
		p.top = 0
	}
	max := len(p.visible) - p.height
	if max < 0 {
		max = 0
	}
	if p.top > max {
		p.top = max
	}
}

func (p *picker) toggle() {
	if p.multi && p.cursor >= 0 && p.cursor < len(p.visible) {
		idx := p.visible[p.cursor]
		p.items[idx].sel = !p.items[idx].sel
	}
}

func (p *picker) selectAll(v bool) {
	for _, idx := range p.visible {
		p.items[idx].sel = v
	}
}

func (p *picker) current() (item, bool) {
	if p.cursor < 0 || p.cursor >= len(p.visible) {
		return item{}, false
	}
	return p.items[p.visible[p.cursor]], true
}

func (p *picker) replaceCurrent(it item) {
	if p.cursor < 0 || p.cursor >= len(p.visible) {
		return
	}
	p.items[p.visible[p.cursor]] = it
	p.rebuildVisible()
}

func (p *picker) selected() []item {
	var out []item
	for _, it := range p.items {
		if it.sel {
			out = append(out, it)
		}
	}
	return out
}

func (p *picker) selectedCount() int {
	n := 0
	for _, it := range p.items {
		if it.sel {
			n++
		}
	}
	return n
}

func (p *picker) visibleSelectedCount() int {
	n := 0
	for _, idx := range p.visible {
		if p.items[idx].sel {
			n++
		}
	}
	return n
}

// view renders exactly p.height rows (padded), with scroll indicators.
func (p *picker) view() string {
	if len(p.visible) == 0 {
		return rowDesc.Render("  (empty)")
	}
	var b strings.Builder
	end := p.top + p.height
	if end > len(p.visible) {
		end = len(p.visible)
	}
	for i := p.top; i < end; i++ {
		b.WriteString(p.renderRow(i))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	out := b.String()
	// scroll affordances on the last line region (kept inside height by caller padding)
	return out
}

func (p *picker) renderRow(i int) string {
	it := p.items[p.visible[i]]
	cur := i == p.cursor

	bar := "  "
	if cur {
		bar = barStyle.Render("▌ ")
	}

	box := ""
	if p.multi {
		if it.sel {
			box = checkOn.Render("◉ ")
		} else {
			box = checkOff.Render("○ ")
		}
	}
	star := ""
	if it.star {
		star = starStyle.Render("★ ")
	}

	titleStyle := rowNormal
	descStyle := rowDesc
	if cur {
		titleStyle, descStyle = rowCursor, rowDescCur
	}

	// width budget: width - bar(2) - box(2 if multi) - star marker
	avail := p.width - 2
	if p.multi {
		avail -= 2
	}
	if it.star {
		avail -= 2
	}
	if avail < 6 {
		avail = 6
	}

	// single-column rows (no description): just the title
	if it.desc == "" {
		return bar + box + star + titleStyle.Render(truncate(it.title, avail))
	}

	// two-column rows: pad every title to a shared column so descriptions align.
	const gap = 2 // spaces between the title column and the description
	col := p.titleW
	if max := avail - 8 - gap; col > max { // always leave room for some description
		col = max
	}
	if col < 1 {
		col = 1
	}
	t := truncate(it.title, col)
	pad := col - lipgloss.Width(t)
	if pad < 0 {
		pad = 0
	}
	descW := avail - col - gap
	desc := ""
	if descW > 1 {
		desc = strings.Repeat(" ", gap) + descStyle.Render(truncate(it.desc, descW))
	}
	return bar + box + star + titleStyle.Render(t) + strings.Repeat(" ", pad) + desc
}

// scrollFooter is a 1-line indicator the view can append (e.g. "  3/29 ▾").
func (p *picker) scrollFooter() string {
	if len(p.visible) == 0 {
		return ""
	}
	pos := scrollHint.Render
	left := ""
	if p.top > 0 {
		left = "▲ "
	}
	right := ""
	if p.top+p.height < len(p.visible) {
		right = " ▼"
	}
	frac := rowDesc.Render(itoa(p.cursor+1) + "/" + itoa(len(p.visible)))
	sel := ""
	if p.multi {
		sel = "   " + checkOn.Render("◉ "+itoa(p.selectedCount())+" marked")
	}
	if p.filter != "" {
		sel += "   " + rowDesc.Render("/"+p.filter)
	}
	return pos(left) + frac + pos(right) + sel
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
