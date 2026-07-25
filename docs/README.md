# grove documentation

Multi-repo agent workspaces backed by git worktrees.

Start here depending on what you need:

| If you want to… | Read |
|---|---|
| Understand the vocabulary — grove, workspace, role, profile | [Concepts](concepts.md) |
| Install and run your first grove | [Quick start](quick-start.md) |
| Know every command and flag | [CLI reference](cli-reference.md) |
| Know the interactive keybindings | [Keyboard reference](keyboard.md) |
| Understand the file formats | [Configuration reference](config-reference.md) |
| Understand *how* it's built and *why* those choices | [Architecture](architecture.md) |
| See the reasoning behind each major decision | [Design decisions](decisions.md) |
| Point an AI agent at a grove | [Working with agents](agents.md) |
| Contribute code | [../CONTRIBUTING.md](../CONTRIBUTING.md) and [../CLAUDE.md](../CLAUDE.md) |
| Know whether grove is worth using at all | [../TRIAL.md](../TRIAL.md) |

## The one-paragraph version

Every git-worktree tool in the ecosystem is *one repo, many agents* — parallel
sessions against a single codebase. Grove is the inversion: *many repos, one
task*. It materialises a directory where the repositories a task touches sit as
siblings, each a git worktree with a role (`write` or `read`), compiles the
context an agent needs to work across them, and then gets out of the way. It
does not run your agent. That last decision is what lets any agent CLI —
`claude`, `aider`, `codex` — point at the same grove.

## A note on maturity

Grove is deliberately small and, by its author's own estimate, **probably won't
survive its first month of use**. These docs are thorough not because the tool
is proven, but because the *design* is the artefact worth keeping — see
[Design decisions](decisions.md) and [../TRIAL.md](../TRIAL.md). If you are
evaluating grove, read TRIAL.md before you invest in it.
