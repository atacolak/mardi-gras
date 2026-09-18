# Keybindings

Press `?` from anywhere to open the full help overlay. Inside it, `h` / `l` (or `←` / `→`) page through the sections and `esc`, `q`, or `?` closes it.

Keys marked **(orch)** need a live orchestrator — Gas Town (`gt`) or Gas City. Without one they are inert no-ops rather than errors. A few are narrower still and say so.

## Global

| Key          | Action                     |
| ------------ | -------------------------- |
| `q`          | Quit application           |
| `ctrl+c`     | Quit from anywhere, including forms, dialogs and input bars |
| `tab`        | Switch active pane         |
| `esc`        | Clear scope first, then existing focus behavior |
| `?`          | Toggle help overlay        |
| `: / Ctrl+K` | Open command palette      |
| `/`          | Enter filter mode          |
| `f`          | Toggle focus mode (my work + top priority) |
| `E`          | Scope the parade to the selected issue's epic subtree |
| `ctrl+g`     | Toggle Gas Town panel **(orch)** |
| `o`          | Toggle the actor society pane **(actor CLI)** |
| `p`          | Toggle problems view **(orch)** |
| `D`          | Toggle doctor diagnostics overlay |
| `M`          | Toggle Codex (MCP) live transcript |

`ctrl+g`, `o`, `p`, `D` and `M` all render into the same right-hand panel, but opening one does not always close the others: `o` displaces nothing, so pressing it while the Gas Town panel is open leaves Gas Town holding the shared slot, and the actors pane becomes visible once Gas Town is closed. `o` is progressive-hide like `ctrl+g`: without the `actor` CLI on `PATH` the pane, its key and its help section do not exist (see [Actors Pane](#actors-pane-o)), and pressing `o` is a no-op rather than an error.

## Parade

| Key          | Action                                    |
| ------------ | ----------------------------------------- |
| `j` / `k`    | Navigate down/up                         |
| `g` / `G`    | Jump to top / bottom                     |
| `enter`      | Focus detail pane                         |
| `>`          | Collapse or expand the selected branch    |
| `click`      | Select a bead and focus the pane under the pointer |
| `wheel`       | Move the pane under the pointer          |
| `/`          | Enter filter mode                         |
| `f`          | Toggle focus mode (my work + top priority)|
| `E`          | Scope the parade to the selected issue's epic subtree |
| `S`          | Cycle sibling sort: attention (default) / priority / chronological (bead ID) |
| `a`          | Launch agent (tmux: new pane; orchestrator: sling) |
| `A`          | Stop the active agent on the issue         |

`a` picks its dispatch path from the environment: with an orchestrator it slings, on the Gas City backend it first prompts for a target agent, in tmux without an orchestrator it opens an agent pane, and outside tmux it suspends the TUI. Pressing `a` on an issue that already has a tmux agent switches to that pane instead of launching a second one. `A` asks the orchestrator to unsling when one is present and only falls back to killing the tmux pane when there is none.

### Epic scope (`E`)

`E` scopes the parade to the epic above the cursor — the epic itself plus every descendant, walked through real `parent-child` edges rather than dotted IDs. The epic is the selected issue if it is one, otherwise its nearest epic ancestor. An issue with no epic above it has nothing to scope to, and `E` leaves the view untouched.

Scoping narrows rather than resets, so it composes with everything else: a fuzzy query, `--exclude-type` / `--exclude-label` and focus mode all still apply inside the scope, and the scope pass runs last, after all of them.

`esc` unwinds one layer at a time, and the scope always goes first:

1. The first press clears the epic scope — focus mode and the detail pane both survive it.
2. The next press drops focus mode.
3. The next press leaves the detail pane for the parade.

Because the scope is the last pass, you can scope to an epic that an earlier pass already removed: `E` on a child while `--exclude-type epic` is hiding the epic, or under a focus mode that never picked the epic, or with a filter query that matches the child but not the epic's own title. The scope root is then not in the loaded set, so there is no subtree left to show and the parade renders empty with the header's counts at zero. That is the honest answer rather than a bug — and rather than silently widening back to the whole board — so the footer keeps a `⚜ SCOPE <epic-id>` chip lit for as long as the scope is active, including when it has no rows to show. The footer is not always the bottom bar, though: a filter query, a bulk-selection prompt, or an open input bar takes that row instead, and the chip is not on screen while one of them is up. In the filter case the `N/M match` count on the filter bar is the cue. Whenever the chip is visible, the empty parade plus the chip is the operator's cue to press `esc`.

### Operator Attention

Operator Attention is an explicit stored Beads state. The lead transition is `br update <epic> --status review`; `review` is absent from Ready. Closing children does not automatically enter review.

## Quick Actions

| Key           | Action                                   |
| ------------- | ---------------------------------------- |
| `1` / `2` / `3` | Set status: in_progress / open / closed |
| `!` / `@` / `#` / `$` | Set priority: P1 / P2 / P3 / P4 |
| `b`           | Copy branch name to clipboard            |
| `B`           | Create + checkout git branch             |
| `N`           | Create new issue                         |
| `e`           | Edit selected issue (title, priority)    |
| `r`           | Add comment to selected issue            |
| `y`           | Assign selected issue                    |
| `t`           | Add label to selected issue              |
| `l`           | Add dependency link                      |
| `s`           | Pick a formula and sling the issue **(orch)** |
| `n`           | Nudge the agent working the issue **(orch)** |
| `C`           | Create a convoy from the selection **(orch)** |

`3` on a single issue closes it and atomically claims the next ready bead. `n` only fires when an agent is actually active on the selected issue. `C` on an epic builds the convoy from that epic's tree; on any other issue (or a multi-selection) it builds from the selected IDs.

## Multi-select

| Key           | Action                              |
| ------------- | ----------------------------------- |
| `space` / `x` | Toggle select on cursor issue      |
| `Shift+J/K`   | Select and move down/up            |
| `X`           | Clear all selections                |
| `1/2/3`       | Bulk set status on selected         |
| `!/@/#/$`     | Bulk set priority on selected       |
| `a`           | Sling all selected issues           |
| `s`           | Pick formula and sling all selected |
| `C`           | Create a convoy from all selected   |

## Detail Pane

| Key          | Action                     |
| ------------ | -------------------------- |
| `j` / `k`    | Scroll down/up            |
| `esc`        | Back to parade pane        |
| `/`          | Enter filter mode          |
| `a`          | Launch agent               |
| `A`          | Stop the active agent      |
| `m`          | Mark active molecule step done |

Any other navigation key (`pgup`, `pgdn`, `home`, `end`, …) is passed straight to the viewport.

## Gas Town Panel (`ctrl+g`)

The panel takes over the detail pane; give it focus with `tab` (or `enter` from the parade) and these keys route to it instead of the global handlers.

| Key          | Action                          |
| ------------ | ------------------------------- |
| `j` / `k`    | Navigate agents/convoys/mail   |
| `g` / `G`    | Jump to first/last             |
| `tab`        | Switch section (agents/convoys/mail) |
| `n`          | Nudge selected agent (agents)   |
| `h`          | Handoff work from agent (agents) |
| `K`          | Decommission polecat (agents; polecat role only) |
| `enter`      | Expand/collapse convoy or message |
| `l`          | Land convoy (convoys)           |
| `x`          | Close convoy (convoys)          |
| `w`          | Watch convoy (convoys) / compose message (agents, mail) |
| `W`          | Unwatch convoy (convoys)        |
| `r`          | Reply to selected message (mail) |
| `d`          | Archive selected message (mail) |
| `R`          | Mark all mail read (mail)       |

`tab` skips sections with nothing in them, so a town with no convoys cycles agents → mail → agents. Expanding an unread message also marks it read. Note that `l`, `x`, `r`, `d` and `w` mean something different here than they do in the parade — the panel claims them while it has focus.

## Problems View (`p`)

| Key          | Action                          |
| ------------ | ------------------------------- |
| `j` / `k`    | Navigate problems              |
| `g` / `G`    | Jump to first/last             |
| `n`          | Nudge agent on selected problem |
| `h`          | Handoff from agent              |
| `K`          | Decommission polecat (polecat role only) |
| `R`          | Recover dead rig — opens a confirmation dialog, then releases + re-slings orphans |

`R` only applies to a `dead_rig` problem; on any other row it does nothing. Recovery shells out to `gt` directly, so it is offered on the Gas Town backend only.

## Actors Pane (`o`)

The read-only window onto the actor society: who is live, what they own, and what they are doing. It takes over the detail pane and stays a pure observation surface — mg never restores, recycles, or messages an actor from here.

It is fed by the `actor` CLI's societyRow projection (`actor project <id-or-root> --json`, or `actor list --json` village-wide), re-read every 10 seconds while the pane is open, with a fetch kicked immediately when the pane opens. A failed fetch keeps the last roster on screen and reports the failure under it in a toast; polling continues.

Each row reads `name  kind  lifecycle/readiness  sprint — title  asked: …  now: …`, with `-` where the sprint is null.

| Key          | Action                          |
| ------------ | ------------------------------- |
| `o`          | Open or close the pane          |
| `v`          | Switch scope: this project ↔ the whole village (pane focused) |
| `j` / `k`    | Scroll the roster               |

`v` refetches as it flips, so the village listing appears without waiting out the poll interval. Without the `actor` CLI on `PATH`, the pane, the `o` key, the footer hint and the help section are all absent.

## Doctor Overlay (`D`)

| Key          | Action                          |
| ------------ | ------------------------------- |
| `j` / `k`    | Scroll diagnostics             |
| `g` / `G`    | Jump to first/last             |
| `R`          | Re-run `bd doctor`              |

## Codex Transcript (`M`)

| Key          | Action                          |
| ------------ | ------------------------------- |
| `M`          | Close the transcript            |
| `r`          | Reply to the live Codex session |
| `esc`        | Dismiss the reply input         |

Approval requests from Codex surface as a modal dialog: `j`/`k` to choose, `enter` to confirm, `esc` to cancel.

## Command Palette (`:` / `Ctrl+K`)

| Key                | Action                     |
| ------------------ | -------------------------- |
| type               | Fuzzy-filter the actions   |
| `up` / `ctrl+p`    | Previous match             |
| `down` / `ctrl+n`  | Next match                 |
| `enter`            | Run selected action        |
| `esc`              | Close the palette          |

The palette also carries actions with no key of their own — add note, claim next ready, prune preview/prune closed, create & assign to crew, cascade close, cycle layout, resume last Codex session, recover dead rigs. See [filtering.md](filtering.md#command-palette).

## Filter Mode (`/`)

| Key          | Action                          |
| ------------ | ------------------------------- |
| `enter`      | Keep the query applied and return to list navigation |
| `esc`        | Clear the query and exit        |

Every other printable key is a literal, so `q` and `?` type rather than quit or open help. `ctrl+c` still quits.
