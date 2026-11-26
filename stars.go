package main

import (
	"os"
	"path/filepath"
	"strings"
)

func starsPath() string { return filepath.Join(configDir(), "stars") }

func loadStars() []item {
	lines := readLines(starsPath())
	out := make([]item, 0, len(lines))
	seen := map[string]bool{}
	for _, ln := range lines {
		f := strings.SplitN(ln, "\t", 3)
		if len(f) < 2 {
			continue
		}
		src := canonSource(strings.TrimSpace(f[0]))
		skill := strings.TrimSpace(f[1])
		if src == "" || skill == "" {
			continue
		}
		key := starKey(src, skill)
		if seen[key] {
			continue
		}
		seen[key] = true
		desc := ""
		if len(f) == 3 {
			desc = strings.TrimSpace(f[2])
		}
		out = append(out, item{id: skill, title: skill, desc: desc, source: src, star: true})
	}
	return out
}

func saveStars(stars []item) error {
	seen := map[string]bool{}
	var b strings.Builder
	for _, st := range stars {
		src := strings.TrimSpace(st.source)
		skill := strings.TrimSpace(st.id)
		if src == "" || skill == "" {
			continue
		}
		key := starKey(src, skill)
		if seen[key] {
			continue
		}
		seen[key] = true
		b.WriteString(src)
		b.WriteByte('\t')
		b.WriteString(skill)
		b.WriteByte('\t')
		b.WriteString(strings.TrimSpace(st.desc))
		b.WriteByte('\n')
	}
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	tmp := starsPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, starsPath())
}

func applyStars(items []item, src string) []item {
	stars := loadStars()
	for i := range items {
		items[i].source = src
		items[i].star = hasStar(stars, src, items[i].id)
	}
	return items
}

func toggleStar(it item) (bool, error) {
	if it.source == "" || it.id == "" {
		return false, nil
	}
	stars := loadStars()
	for i, st := range stars {
		if st.source == it.source && st.id == it.id {
			stars = append(stars[:i], stars[i+1:]...)
			return false, saveStars(stars)
		}
	}
	stars = append(stars, item{id: it.id, title: it.id, desc: it.desc, source: it.source, star: true})
	return true, saveStars(stars)
}

// starsForSources returns the Stars that belong to any of sources.
func starsForSources(sources []string) []item {
	in := map[string]bool{}
	for _, src := range sources {
		in[src] = true
	}
	var out []item
	for _, st := range loadStars() {
		if in[st.source] {
			out = append(out, st)
		}
	}
	return out
}

func hasStar(stars []item, src, skill string) bool {
	for _, st := range stars {
		if st.source == src && st.id == skill {
			return true
		}
	}
	return false
}

func starKey(src, skill string) string { return src + "\t" + skill }
