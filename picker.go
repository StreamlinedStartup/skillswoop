package main

import (
	"strings"
)

type item struct {
	id    string
	title string
	desc  string
	sel   bool
}

// picker is a self-contained, windowed, optionally multi-select list.
// It owns its own scroll math so long lists never smear (the bug we hit in gum).
type picker struct {
	items  []item
	cursor int
	top    int // index of first visible row
	width  int // inner content width
	height int // visible rows
	multi  bool
}

func newPicker(items []item, multi bool) *picker {
	return &picker{items: items, multi: multi}
}

func (p *picker) setSize(w, h int) {
	p.width = w
	if h < 1 {
		h = 1
	}
	p.height = h
	p.clampWindow()
}

func (p *picker) len() int { return len(p.items) }

func (p *picker) move(d int) {
	if len(p.items) == 0 {
		return
	}
	p.cursor += d
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.items) {
		p.cursor = len(p.items) - 1
	}
	p.clampWindow()
}

func (p *picker) home() { p.cursor = 0; p.clampWindow() }
func (p *picker) end()  { p.cursor = len(p.items) - 1; p.clampWindow() }

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
	max := len(p.items) - p.height
	if max < 0 {
		max = 0
	}
	if p.top > max {
		p.top = max
	}
}

func (p *picker) toggle() {
	if p.multi && p.cursor >= 0 && p.cursor < len(p.items) {
		p.items[p.cursor].sel = !p.items[p.cursor].sel
	}
}

func (p *picker) selectAll(v bool) {
	for i := range p.items {
		p.items[i].sel = v
	}
}

func (p *picker) current() (item, bool) {
	if p.cursor < 0 || p.cursor >= len(p.items) {
		return item{}, false
	}
	return p.items[p.cursor], true
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

// view renders exactly p.height rows (padded), with scroll indicators.
func (p *picker) view() string {
	if len(p.items) == 0 {
		return rowDesc.Render("  (empty)")
	}
	var b strings.Builder
	end := p.top + p.height
	if end > len(p.items) {
		end = len(p.items)
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
	it := p.items[i]
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

	// width budget: width - bar(2) - box(2 if multi)
	avail := p.width - 2
	if p.multi {
		avail -= 2
	}
	if avail < 6 {
		avail = 6
	}

	title := it.title
	// reserve room for an inline dim description after a separator
	descSep := ""
	if it.desc != "" {
		// give the title up to ~40% then description fills the rest
		tw := avail * 2 / 5
		if tw < len([]rune(title)) {
			// keep full short titles; only cap long ones
		}
		title = truncate(title, avail)
		used := len([]rune(title))
		rest := avail - used - 3
		if rest > 4 {
			d := truncate(it.desc, rest)
			if cur {
				descSep = "   " + rowDescCur.Render(d)
			} else {
				descSep = "   " + rowDesc.Render(d)
			}
		}
	} else {
		title = truncate(title, avail)
	}

	var ts string
	if cur {
		ts = rowCursor.Render(title)
	} else {
		ts = rowNormal.Render(title)
	}
	return bar + box + ts + descSep
}

// scrollFooter is a 1-line indicator the view can append (e.g. "  3/29 ▾").
func (p *picker) scrollFooter() string {
	if len(p.items) == 0 {
		return ""
	}
	pos := scrollHint.Render
	left := ""
	if p.top > 0 {
		left = "▲ "
	}
	right := ""
	if p.top+p.height < len(p.items) {
		right = " ▼"
	}
	frac := rowDesc.Render(itoa(p.cursor+1) + "/" + itoa(len(p.items)))
	sel := ""
	if p.multi {
		sel = "   " + checkOn.Render("◉ "+itoa(p.selectedCount())+" marked")
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
