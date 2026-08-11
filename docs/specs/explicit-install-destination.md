# Spec: Explicit Install Destination

Status: Approved

## Objective

Add an explicit destination step to the SkillSwoop terminal UI for every install flow:

- Skills selected from a saved source
- Starred skills
- Plugins selected from a marketplace

After marking items, Enter opens a destination screen instead of installing immediately. The screen names the selected item type, count, source when applicable, selected names, and the two destinations. `This project` is highlighted every time. The user may move to `Globally`, then press Enter to install.

The feature succeeds when users no longer need to discover or interpret the hidden Tab scope toggle to install skills or plugins globally, and can clearly verify what will be installed and where before the command runs.

## Assumptions

1. This changes the TUI flow only. The existing CLI `-g` behavior remains intact.
2. Global keeps its current meaning: available user-wide across projects for the configured agents.
3. The existing terminal theme remains intact. This is an interaction improvement, not a visual redesign.
4. One installation request sends every marked item to the same destination.
5. Project scope is always the safer initial selection and is never remembered between installs.
6. Update scope remains separate from install destination state.

## User Experience

### Selection screen

```text
INSTALL · mattpocock/skills

  ◉ diagnose         Disciplined diagnosis for hard bugs
▌ ○ tdd              Test-driven development
  ◉ grill-with-docs  Challenge a plan against the domain model

  2 skills selected

↑↓ move   space mark   / filter   enter continue   esc back
```

### Destination screen

```text
INSTALL 2 SKILLS

  From mattpocock/skills

  Selected
  diagnose, grill-with-docs

  Where should these skills be installed?

▌ ● This project
    Only in /Users/example/dev/current-project

  ○ Globally
    Available in every project for Claude Code and Codex

↑↓ choose   enter install   esc change selection
```

The plugin flow uses the same destination screen with the heading changed to `INSTALL <N> PLUGIN(S)`. For long selections, the summary shows the first three names followed by `+ N more`.

### Flow

```text
Selection list
    Enter
      |
      v
Destination screen, default: This project
    | Esc                         | Enter
    v                             v
Original selection list     Existing install execution
                                  |
                                  v
                       Running and result screens
```

## Functional Requirements

### Destination content

- Show `INSTALL <N> SKILL(S)` or `INSTALL <N> PLUGIN(S)` as the heading.
- Show the source for single-source skill and plugin installs.
- Summarize up to three selected names, followed by `+ N more` when needed.
- For starred skills, include each summarized skill's source because one selection may span repositories.
- Show the absolute current working directory below `This project`.
- Describe `Globally` as available in every project and name the configured agents.

### Interaction

- Project is selected whenever the destination screen opens, regardless of update scope or any prior install.
- Up/Down and `j`/`k` move between the two destinations.
- Enter begins installation in the selected scope.
- Escape returns to the original marked list with marks, filter, scroll position, and cursor preserved.
- Tab does nothing on source, marketplace, skill, starred-skill, and plugin install screens.
- Enter with no marked items keeps the user on the list and shows the existing mark-first guidance.
- The footer reads `↑↓ choose`, `enter install`, and `esc change selection` on the destination screen.

### Execution

- Project installs omit `-g`.
- Global installs prepend `-g` to the existing engine command.
- Running and result titles include the item count, item type, and destination.
- Existing plugin hooks confirmation remains after destination selection when required.
- The selected destination and result title survive the plugin hooks check and confirmation flow.
- Existing plugin scope mapping, engine commands, CLI flags, and update behavior remain unchanged.

### Scope indicator

- Remove the unrelated persistent scope indicator from install screens.
- Keep existing main-menu update scope behavior.
- Label the main-menu indicator as `update scope` so it is not mistaken for the install destination.

## Tech Stack

- Go 1.23
- Bubble Tea 1.3.4 for state and update flow
- Bubbles 0.20.0 for terminal components
- Lip Gloss 1.0.0 for rendering
- Go's built-in testing package

No new dependencies, environment variables, configuration files, engine subcommands, or persisted preferences are required.

## Commands

```sh
gofmt -w <changed-go-files>
go vet ./...
go test ./...
go build -o /tmp/swoop .
go run .
```

The first four commands are required automated verification. `go run .` is used for the manual TUI walkthrough.

## Project Structure

```text
model.go                  TUI state, screen types, and install request values
update.go                 State transitions and keyboard behavior
menu.go                   Install request creation and scoped command execution
view.go                   Destination screen and status-bar rendering
update_test.go            State transition and command routing tests
render_test.go            Destination screen and terminal-width tests
README.md                 User-facing keyboard and install-flow documentation
docs/ARCHITECTURE.md      TUI state-machine documentation
docs/specs/               Approved feature specifications
```

The embedded Bash engine is outside this feature's implementation scope because its project and global behavior already exists.

## Code Style

Follow the existing small-value-type and switch-based Bubble Tea style. Keep install intent explicit rather than storing opaque callbacks:

```go
type installRequest struct {
	kind   installKind
	items  []item
	source string
	origin screen
	global bool
}
```

Use existing naming conventions, `gofmt`, explicit empty-selection handling, and pure helpers for selection summaries and scope-specific argument construction. Do not add logging for ordinary UI transitions.

## Testing Strategy

### State transition tests

- Enter from skills, starred skills, and plugins opens the destination screen without executing an install.
- Every destination screen starts on project scope, even when update scope is global.
- Escape restores the exact prior picker state.
- Project execution omits `-g`; global execution includes it.
- Hook-bearing plugins retain the selected destination through the Codex hooks flow.
- Empty selections never open the destination screen.
- Tab no longer changes install scope within installer screens.

### Render tests

- Skill and plugin screens show source, selected names, count, project path, configured agents, and active destination.
- Starred skills from multiple sources remain identifiable.
- Long selections collapse to `+ N more`.
- Narrow terminals truncate safely without overflowing.

### Regression and manual verification

- Run `go vet ./...`, `go test ./...`, and `go build -o /tmp/swoop .`.
- Walk project and global installation for skills, starred skills, and plugins with `go run .`.
- Confirm the existing update and plugin hooks flows still behave as documented.

## Boundaries

### Always

- Preserve existing engine scope semantics.
- Keep selection state when backing out of the destination screen.
- Use configured agent names in destination copy.
- Keep every interaction keyboard accessible.
- Update affected user and architecture documentation when implementation begins.

### Ask first

- Change update behavior.
- Persist the last install scope.
- Change CLI flags or engine subcommands.
- Change plugin scope semantics.
- Add dependencies or configuration files.

### Never

- Default an installation to global scope.
- Install before the destination is explicitly confirmed.
- Expose global installation through another hidden shortcut.
- Log secrets, API keys, paths containing sensitive data, or personally identifiable information.
- Remove or weaken existing tests to make the feature pass.

## Success Criteria

- All three install entry points use the same explicit destination screen.
- The user sees what is selected and whether the installation is project-local or global before execution.
- Project is always the initial destination.
- Global installs pass `-g`; project installs do not.
- Escape returns to the original selection without losing state.
- Existing CLI `-g`, update behavior, plugin hooks behavior, and install destinations remain compatible.
- Automated checks pass and the manual TUI walkthrough confirms both destinations for every install entry point.

## Open Questions

None. The approved product choices are the final destination step, project as the default, coverage for every install entry point, and removal of Tab as an install-scope shortcut.
