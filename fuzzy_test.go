package main

import "testing"

func TestFuzzyScoreMatchesOrderedCharacters(t *testing.T) {
	if _, ok := fuzzyScore("test-driven development", "tdd"); !ok {
		t.Fatal("expected ordered character match")
	}
	if _, ok := fuzzyScore("test-driven development", "zzz"); ok {
		t.Fatal("did not expect unrelated query to match")
	}
}

func TestRankedMatchesPreferSubstring(t *testing.T) {
	items := []item{
		{id: "alpha", title: "alpha", desc: "test driven docs"},
		{id: "tdd", title: "tdd", desc: "short name"},
	}
	got := rankedMatches(items, "tdd")
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2", len(got))
	}
	if got[0] != 1 {
		t.Fatalf("first match index = %d, want 1", got[0])
	}
}

func TestPickerFilterPreservesMarkedState(t *testing.T) {
	p := newPicker([]item{
		{id: "tdd", title: "tdd", desc: "test driven"},
		{id: "review", title: "review", desc: "code review"},
	}, true)
	p.toggle()
	p.setFilter("review")
	if p.len() != 1 {
		t.Fatalf("filtered len = %d, want 1", p.len())
	}
	p.toggle()
	p.setFilter("")
	if got := p.selectedCount(); got != 2 {
		t.Fatalf("selected count = %d, want 2", got)
	}
}
