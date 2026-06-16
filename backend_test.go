package main

import "testing"

func TestStripANSICollapsesCarriageReturnProgress(t *testing.T) {
	raw := "Cloning repository◐\rCloning repository◓\rRepository cloned\nFound 1 skill\n"
	got := stripANSI(raw)
	want := "Repository cloned\nFound 1 skill\n"
	if got != want {
		t.Fatalf("stripANSI() = %q, want %q", got, want)
	}
}

func TestListParsersDedupeItems(t *testing.T) {
	items := dedupeItems(parseTabbed("tdd\tone\ntdd\ttwo\nreview\tthree\n", 2))
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2: %#v", len(items), items)
	}
	if items[0].id != "tdd" || items[0].desc != "one" {
		t.Fatalf("first item = %#v", items[0])
	}
}
