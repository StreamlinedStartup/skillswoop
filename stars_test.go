package main

import "testing"

func TestStarsRoundTripDedupesBySourceAndSkill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	err := saveStars([]item{
		{id: "tdd", desc: "one", source: "owner/repo"},
		{id: "tdd", desc: "two", source: "owner/repo"},
		{id: "review", desc: "three", source: "owner/repo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := loadStars()
	if len(got) != 2 {
		t.Fatalf("got %d stars, want 2", len(got))
	}
	if got[0].source != "owner/repo" || got[0].id != "tdd" || got[0].desc != "one" {
		t.Fatalf("first star = %#v", got[0])
	}
}

func TestToggleStarAddsAndRemoves(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	on, err := toggleStar(item{id: "tdd", desc: "desc", source: "owner/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if !on || len(loadStars()) != 1 {
		t.Fatalf("toggle on = %v, stars = %#v", on, loadStars())
	}
	on, err = toggleStar(item{id: "tdd", desc: "desc", source: "owner/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if on || len(loadStars()) != 0 {
		t.Fatalf("toggle on = %v, stars = %#v", on, loadStars())
	}
}
