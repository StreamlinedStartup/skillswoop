# Plugin support for skillswoop (Claude Code + Codex, incl. plugin-bundled hooks)

## Context

skillswoop currently installs only **skills** (SKILL.md directories) by delegating to `npx skills`. Both target agents now have native plugin systems: `claude` 2.1.198 (`claude plugin marketplace add / install --scope / list --json / uninstall`) and `codex` 0.142.2 (`codex plugin marketplace add / add / list --json / remove`) — verified on this machine. Plugins bundle skills, hooks, MCP servers, and (Claude only) commands/agents; **hooks auto-wire on install**, so supporting plugins gives hook support for free. Goal: let users add plugin marketplaces, browse/install/remove plugins for both agents through the existing TUI and CLI, mirroring the skills flow.

Approved decisions:
1. **Plugin-bundled hooks only** — no standalone hooks.json installer. Surface hook-bearing plugins with a badge.
2. **Cross-agent fan-out**: detect which marketplace format(s) a source repo provides (`.claude-plugin/marketplace.json` vs `.agents/plugins/marketplace.json`), install to each compatible configured agent via its native CLI, warn clearly about skipped agents.
3. **Codex `features.hooks` flag**: when installing a hook-bearing plugin for codex and the flag is off, prompt first, then run `codex features enable hooks`.
4. **TUI**: new main-menu entries reusing the source→picker→confirm→running→result flow.

## Design

### Engine (`engine/swoop-core`) — where all real logic goes

**Config**: new `~/.config/swoop/marketplaces` file (do NOT reuse `sources` — those feed `npx skills` and the skills picker). TAB-separated like `stars`:
```
source<TAB>claude_name<TAB>codex_name
```
Names are the `name` fields from each format's marketplace.json, cached at add time (solves "what name did the CLI register?" once). Add `MARKETPLACES` constant next to `STARS` (swoop-core:51) + `save_marketplace`/`remove_marketplace` helpers modeled on `save_star`/`remove_star`.

**Refactor**: extract the gh→git clone fallback from `list_skills_in` (swoop-core:274–288) into `clone_source <src> <tmpdir>`, shared by skills and plugin listing.

**`NODE_PLUGINS` inline Node script** (same pattern as `NODE_WALK` :201): given a checkout, parse both marketplace.json formats, resolve relative-path plugin sources, detect components (`hooks/hooks.json` or `hooks` key ⇒ `hooks`; `.mcp.json` or `mcpServers` key ⇒ `mcp`; plus `commands`/`agents`/`skills`). Emit header line `@marketplace<TAB>claude_name<TAB>codex_name`, then per plugin: `name<TAB>desc<TAB>flags` (flags = comma list, e.g. `claude,codex,hooks,mcp`; dedupe by name across formats, union flags). External-repo plugin sources: badge only what the marketplace entry declares (no second clone) — note this in ENGINE.md.

**New machine-readable subcommands** (add to arg allowlist :625 and dispatch :633):
- `_mkts` — dump the marketplaces file.
- `_plugins <source>` — clone (or walk local path), run `NODE_PLUGINS`.
- `_plugins_installed` — merge `claude plugin list --json` + `codex plugin list --json` via a small inline Node parser → `name@marketplace<TAB>agents<TAB>desc`; tolerate either CLI missing (warn to stderr, emit what we have).
- `_codex_hooks` — `codex features list | awk '$1=="hooks"{print $3}'` → prints `on`/`off`/`n/a`; any parse failure ⇒ `n/a`, never blocks installs.

**New user-facing subcommands** (`cmd_mkt`, `cmd_plugin` with sub-verbs; aliases `marketplace`/`plugins`):
- `mkt add <source>...` — detect formats; per configured agent (`load_agents`): run `run_skills claude plugin marketplace add <source>` / `run_skills codex plugin marketplace add <source>` where the format exists; `warn: skipped <agent> — <reason>` otherwise; then `save_marketplace`.
- `mkt list | remove <source> | update` — remove uses cached names; update = `claude plugin marketplace update` + `codex plugin marketplace upgrade`.
- `plugin install <source> <plugin>... [--no-hooks-enable]` — per agent: claude-code → `claude plugin install "<p>@<claude_name>" --scope <user|project>` (swoop `-g` ⇒ `user`, project ⇒ `project`); codex → `codex plugin add "<p>@<codex_name>"` (no project scope: print note in project mode). Warn skipped agents; exit 1 only if ALL targets failed. Unknown source ⇒ offer `mkt add` first via `ui_confirm`.
- `plugin remove <plugin[@mkt]>...`, `plugin list`.

**Codex hooks flow**: before a codex install of a hook-flagged plugin, if `_codex_hooks` = `off` and no `--no-hooks-enable`: `ui_confirm "Enable Codex hooks (features.hooks) so this plugin's hooks run?"` → yes: `run_skills codex features enable hooks`; no: warn and continue install. CLI gets a real prompt; `SWOOP_ASSUME_YES` ⇒ yes.

Dry-run comes free: all external CLI calls go through the existing `run_skills` wrapper (swoop-core:156).

### Go / TUI

- **`item` struct** (picker.go:9): add `flags string`. Render `[hooks]` `[mcp]` badges (new dim `badgeStyle` in theme.go) between title and desc, subtracting width like the star marker. When a plugin lacks a configured agent's format, suffix desc with `· claude only` / `· codex only`.
- **New screens** (model.go:14 enum): `scMarkets` (single-select, mirrors `scSources`), `scPlugins` (multi-select + `/` filter, mirrors `scSkills`), `scPluginRemove` (mirrors `scRemove`). New model state: `curMarket string`, `pendingInstall []string`, `addMode` flag so `scAdd` doubles as the add-marketplace input.
- **New messages/commands** (model.go + backend.go): `marketsMsg`/`loadMarketsCmd` (reads marketplaces file directly via a `loadMarketplaces()` sibling of existing config readers), `pluginsMsg`/`loadPluginsCmd` (→ `core("_plugins", src)`, header line + `parsePlugins()` 3-column sibling of `parseTabbed` backend.go:224), `installedPluginsMsg`, `codexHooksMsg`/`core("_codex_hooks")`. Installs/removals reuse `opCmd`/`opDoneMsg` unchanged.
- **Menu entries** (menu.go:9): "Install plugins" → `scMarkets` → `scPlugins` (empty marketplaces ⇒ flash "Add a marketplace first"); "Add a marketplace" → `scAdd` in marketplace mode → `opCmd("added marketplace", "mkt", "add", v)`; "Remove plugins" → `scPluginRemove`. In `scMarkets`: `x` = remove (confirm), `u` = `mkt update` (status-bar hints).
- **Hooks confirm round-trip** (update.go): on enter in `scPlugins`, if any marked item has `hooks` flag and agents include codex → stash args in `pendingInstall`, fire `_codex_hooks`; `codexHooksMsg` `off` → `scConfirm` (yes ⇒ normal install, engine auto-enables under SWOOP_ASSUME_YES; no ⇒ same install + `--no-hooks-enable` — add optional `denyCmd func(*model) tea.Cmd` next to `confirmCmd`, nil keeps current behavior); `on`/`n/a` → install directly.
- Wire the three screens into `activeList()` (update.go:160), `buildBody()`/render cases (view.go:43,50), `statusBar()` hints, and the `layout()` filtering special case (update.go:23).

### CLI surface

No main.go changes — passthrough already execs the engine:
```
swoop mkt add|list|remove|update ...
swoop plugin install <source> <plugin>... | list | remove <plugin[@mkt]>...
```
Global flags (`-g`, `--dry-run`) apply; parse `--no-hooks-enable` in the engine flag loop.

## Implementation steps (each verifiable)

1. Engine plumbing: `MARKETPLACES` file + helpers, `clone_source` refactor (existing engine_test.go stays green), allowlist entries.
2. `NODE_PLUGINS` + `_plugins` + `_mkts` + `_codex_hooks`; verify against a local fixture marketplace and `anthropics/claude-plugins-official`.
3. `mkt add/list/remove/update` with skipped-agent warnings; verify with `SWOOP_DRYRUN=1`, then one live run.
4. `plugin install/remove/list` + `_plugins_installed` + hooks prompt/`--no-hooks-enable`.
5. Go backend: `loadMarketplaces`, `parsePlugins`, new cmds/msgs + backend tests.
6. Picker badges (`item.flags`, `badgeStyle`) + render test.
7. TUI screens, menu entries, hooks-confirm round-trip + update/render tests; manual TUI walk.
8. Docs (ENGINE.md: subcommand table, marketplaces file, scope mapping note, hooks-detection caveat; ARCHITECTURE.md: screens/messages; README: plugin quickstart) + `go test ./...`.

## Tests

- `engine_test.go`: `TestEnginePluginsLocalMarketplace` — temp repo with both manifest formats, one plugin with `hooks/hooks.json` + `.mcp.json`; assert `_plugins` header + `name<TAB>desc<TAB>claude,codex,hooks,mcp`. Claude-only repo ⇒ skip warning for codex. `mkt add`/`plugin install` under `SWOOP_DRYRUN=1` with stub `claude`/`codex` scripts on PATH (no real CLIs in CI).
- `backend_test.go`: `parsePlugins` edge cases, `loadMarketplaces`.
- `render_test.go`: new screens + badge rendering at narrow widths.
- `update_test.go`: `codexHooksMsg` routing (`off` → scConfirm; `on` → scRunning), deny path appends `--no-hooks-enable`.

## End-to-end verification

1. `go test ./...`
2. `SWOOP_DRYRUN=1 swoop mkt add anthropics/claude-plugins-official` → shows both native CLI command lines; live run once.
3. `swoop plugin install <mkt> <hook-bearing-plugin>` with codex `features.hooks` off → prompt appears; confirm → `codex features list` shows hooks true; `claude plugin list --json` and `codex plugin list --json` show the plugin.
4. TUI walk: menu → Add a marketplace → Install plugins (badges visible) → install → Remove plugins.
5. Start a Claude Code session and confirm the plugin's hook fires (e.g. SessionStart output).

## Risks

- Codex JSON output shape drift — parse via inline Node try/catch, warn + continue; gate on `have codex`.
- Marketplace `name` renamed upstream after caching — installs break until re-`mkt add`; `mkt list` shows cached names for debugging. Acceptable.
- Hooks badge is best-effort for plugins whose source is an external repo (only marketplace-entry declarations, no second clone).
