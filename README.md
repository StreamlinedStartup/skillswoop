<div align="center">

# ▓▒░ skillswoop ░▒▓

**A neon TUI for managing agent skills — jack them straight into Claude Code & Codex.**

`swoop` is a friendly cyberpunk front-end over [`npx skills`](https://github.com/vercel-labs/skills).
Remember your favorite skill repos, drill into individual skills, and install only the
ones you want — into the current project, for every agent you target.

</div>

```
▓▒░  S W O O P  ░▒▓   // skillswoop · jack skills into claude + codex
══════════════════════════════════════════════════════════════════════
▸ INSTALL · mattpocock/skills   SPACE marks · ENTER installs marked

  ◉ diagnose          Disciplined diagnosis loop for hard bugs…
▌ ○ tdd               Test-driven development with red-green-refactor…
  ◉ grill-with-docs   Grilling session that challenges your plan…
  ○ to-issues         Break a plan into independently-grabbable issues…
```

## Features

- **Cyberpunk Bubble Tea TUI** — gradient banner, neon panels, multi-select with checkboxes.
- **Per-skill install** — pick a repo, then mark exactly the skills you want (not the whole set).
- **Targets multiple agents** — installs to `./.claude/skills/<name>` and `./.agents/skills/<name>` (Codex) at once.
- **Project-local by default** — lands in the current directory; `-g` for global.
- **Updates from GitHub** — `swoop update` pulls the latest for skills in this folder; `swoop update --all` refreshes every folder you've installed into.
- **Browse skills.sh** — search the directory and remember new sources.
- **Scriptable** — every action also works non-interactively (great for CI / dotfiles).

## Install

### Homebrew (macOS / Linux)
```sh
brew install StreamlinedStartup/tap/swoop
```

### go install
```sh
go install github.com/StreamlinedStartup/skillswoop@latest
# installs a binary named `skillswoop`; symlink it to `swoop` if you like:
ln -sf "$(go env GOPATH)/bin/skillswoop" "$(go env GOPATH)/bin/swoop"
```

### Prebuilt binaries
Grab a tarball for your OS/arch from the [Releases](https://github.com/StreamlinedStartup/skillswoop/releases) page and put `swoop` on your `PATH`.

### From source
```sh
git clone https://github.com/StreamlinedStartup/skillswoop
cd skillswoop && go build -o swoop .
```

### Prerequisites
- **Node.js / npx** — used to run `npx skills` (the actual installer).
- **gh** and/or **git** — for listing/cloning repos (private repos work when `gh` is authenticated). Optional but recommended.

## Usage

Just run it:
```sh
swoop
```

| Key | Action |
| --- | --- |
| `↑ ↓` / `j k` | move |
| `space` | mark (multi-select) |
| `a` | mark all / none |
| `enter` | select / install |
| `tab` | toggle PROJECT ⟷ GLOBAL scope |
| `esc` | back · `q` quit |

### Scriptable subcommands
Everything the TUI does is also a plain command:
```sh
swoop add mattpocock/skills            # remember a source
swoop use mattpocock/skills -- --skill diagnose --skill tdd -y
swoop update                           # update skills in the current folder
swoop update --all                     # update every folder you've installed into
swoop -g update                        # update your global skills
swoop browse marketing                 # search skills.sh
swoop list | swoop remove | swoop agents claude-code codex
swoop --version
```

## How it works

`swoop` is a single self-contained binary: a Go/[Bubble Tea](https://github.com/charmbracelet/bubbletea)
TUI with a small Bash **engine embedded** inside it (via `go:embed`). On first run the engine is
extracted to your cache dir and executed; it orchestrates `npx skills`, `gh`/`git`, and Node for
listing. Any CLI arguments pass straight through to the engine, so scripted use behaves identically.

Config lives in `~/.config/swoop/` (`sources`, `agents`, `projects`) and copied skills in
`~/.local/share/swoop/library/`. Existing `ccskill` config is migrated automatically on first run.

## License

[MIT](LICENSE) © StreamlinedStartup
