package main

import (
	"sort"
	"strings"
)

type filterMatch struct {
	index int
	score int
}

func rankedMatches(items []item, query string) []int {
	q := strings.TrimSpace(query)
	out := make([]filterMatch, 0, len(items))
	for i, it := range items {
		score, ok := fuzzyScore(itemSearchText(it), q)
		if ok {
			out = append(out, filterMatch{index: i, score: score})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score == out[j].score {
			return out[i].index < out[j].index
		}
		return out[i].score > out[j].score
	})
	idx := make([]int, len(out))
	for i, m := range out {
		idx[i] = m.index
	}
	return idx
}

func itemSearchText(it item) string {
	if it.source == "" {
		return it.title + " " + it.desc + " " + it.id
	}
	return it.title + " " + it.desc + " " + it.id + " " + it.source
}

func fuzzyScore(text, query string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	t := strings.ToLower(text)
	if q == "" {
		return 0, true
	}
	if t == "" {
		return 0, false
	}
	if pos := strings.Index(t, q); pos >= 0 {
		return 10000 - pos - len(q), true
	}
	score := 0
	last := -1
	run := 0
	for _, qr := range q {
		found := -1
		for i, tr := range t {
			if i <= last {
				continue
			}
			if tr == qr {
				found = i
				break
			}
		}
		if found < 0 {
			return 0, false
		}
		gap := found - last - 1
		if last < 0 {
			gap = found
		}
		if gap == 0 {
			run++
			score += 40 + run*5
		} else {
			run = 0
			score += 15 - min(gap, 12)
		}
		if found == 0 || strings.ContainsRune(" -_/.", rune(t[found-1])) {
			score += 20
		}
		last = found
	}
	return score, true
}
