package main

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestParsePlugins(t *testing.T) {
	out := "@marketplace\tcm\txm\n" +
		"hooky\tHooks and mcp.\tclaude,codex,hooks,mcp\tplugins/hooky\n" + // relpath col ignored
		"plain\t\tclaude\n" +
		"hooky\tdupe\tclaude\n" +
		"\n"
	cn, xn, items := parsePlugins(out)
	if cn != "cm" || xn != "xm" {
		t.Fatalf("names = %q, %q; want cm, xm", cn, xn)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2: %#v", len(items), items)
	}
	if items[0].id != "hooky" || items[0].flags != "claude,codex,hooks,mcp" {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[1].id != "plain" || items[1].desc != "" || items[1].flags != "claude" {
		t.Fatalf("second item = %#v", items[1])
	}
}

func TestParsePluginsHeaderOnly(t *testing.T) {
	cn, xn, items := parsePlugins("@marketplace\tcm\t\n")
	if cn != "cm" || xn != "" || len(items) != 0 {
		t.Fatalf("got %q, %q, %#v", cn, xn, items)
	}
}

func TestLoadMarketplaces(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, "swoop")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "owner/repo\tcm\txm\n" +
		"claude-only\tcm2\t\n" +
		"\t\t\n" // blank source is dropped
	if err := os.WriteFile(filepath.Join(cfg, "marketplaces"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	mkts := loadMarketplaces()
	if len(mkts) != 2 {
		t.Fatalf("got %d marketplaces, want 2: %#v", len(mkts), mkts)
	}
	if mkts[0] != (marketplace{source: "owner/repo", claude: "cm", codex: "xm"}) {
		t.Fatalf("first = %#v", mkts[0])
	}
	items := marketItems()
	if items[0].desc != "claude + codex" || items[1].desc != "claude only" {
		t.Fatalf("market items = %#v", items)
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag("claude,codex,hooks", "hooks") {
		t.Fatal("expected hooks flag")
	}
	if hasFlag("claude,codex", "code") || hasFlag("", "hooks") {
		t.Fatal("unexpected flag match")
	}
}
