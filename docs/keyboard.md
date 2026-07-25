# Keyboard reference

Grove's interactive surfaces (the repo picker and the `grove ls -i` dashboard)
are built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Repo picker (`grove new` with no `-w`/`-r`)

| Key | Action |
| --- | --- |
| *any character* | Fuzzy-filter the repo list |
| `Tab` | Cycle highlighted repo: unselected → write → read → unselected |
| `Shift+Tab` | Cycle the other way |
| `↑` / `↓` (or `Ctrl+p` / `Ctrl+n`) | Move the cursor |
| `Enter` | Confirm selection (requires at least one repo picked) |
| `Esc` / `Ctrl+c` | Cancel — creates nothing |

Filtering never disturbs your selections: a repo marked `write` stays marked
even while filtered out of view, because selection state is keyed to the repo,
not to its position in the filtered list.

`Tab` cycles roles rather than `Space`, because `Space` is a legitimate filter
character.

## Dashboard (`grove ls -i`)

| Key | Action |
| --- | --- |
| `↑` / `↓` (or `k` / `j`) | Move between groves |
| `q` / `Esc` / `Ctrl+c` | Quit |

The dashboard is read-only: it shows each grove's repos, roles, and derived git
status (dirty count, ahead/behind). It does not mutate anything.
