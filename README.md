# skillswoop

A terminal UI (and CLI) for managing agent skills for Claude Code and Codex. It is a
front-end over [`npx skills`](https://github.com/vercel-labs/skills): remember skill
repositories, pick individual skills from a repo, install them into a project for one or
more agents, and update them from GitHub.

The command is `swoop`.

## Install

Homebrew:

```sh
brew install StreamlinedStartup/tap/swoop
```

go install (installs a binary named `skillswoop`):

```sh
go install github.com/StreamlinedStartup/skillswoop@latest
```

Prebuilt binaries: see [Releases](https://github.com/StreamlinedStartup/skillswoop/releases).

From source:

```sh
git clone https://github.com/StreamlinedStartup/skillswoop
cd skillswoop && go build -o swoop .
```

### Requirements

- `node` / `npx` — used to run `npx skills`, which performs the actual install/update.
- `gh` and/or `git` — used to list and clone repositories. `gh` (authenticated) is required for private repos. Optional otherwise.

## Usage

Run `swoop` with no arguments for the TUI.

| Key | Action |
| --- | --- |
| `↑`/`↓`, `j`/`k` | move |
| `space` | mark (multi-select) |
| `a` | mark all / none |
| `enter` | select / confirm |
| `ctrl+r` | rename a source (set a display alias) |
| `tab` | toggle project / global scope |
| `esc` | back · `q` quit |

Every action is also available as a non-interactive command:

```sh
swoop add owner/repo                              # remember a source
swoop use owner/repo -- --skill A --skill B -y    # install specific skills
swoop update                                      # update skills in the current folder
swoop update --all                                # update every folder you've installed into
swoop -g update                                   # update global skills
swoop browse <query>                              # search skills.sh
swoop list | swoop remove | swoop agents claude-code codex
swoop --version
```

## Where skills are installed

By default skills install into the current directory:

- Claude Code: `./.claude/skills/<name>`
- Codex: `./.agents/skills/<name>`

`-g` installs to the global agent directories instead. Updates are tracked per directory
via the `skills-lock.json` that `npx skills` writes.

## Files

- Config: `~/.config/swoop/` — `sources`, `agents`, `projects`, `aliases`
- Library (skills moved out by `swoop stash`): `~/.local/share/swoop/library/`
- Engine cache: `~/Library/Caches/swoop/` (macOS) or `$XDG_CACHE_HOME/swoop/`

Existing `~/.config/ccskill` configuration is copied to `~/.config/swoop` on first run.

## How it works

`swoop` is a single Go binary: a [Bubble Tea](https://github.com/charmbracelet/bubbletea)
TUI with a Bash engine embedded via `go:embed`. On first run the engine is written to the
cache directory and executed; it orchestrates `npx skills`, `gh`/`git`, and Node. Any
command-line arguments are passed straight through to the engine.

## License

[MIT](LICENSE)
