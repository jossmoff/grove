# grove

[![CI](https://github.com/jossmoff/grove/actions/workflows/ci.yaml/badge.svg)](https://github.com/jossmoff/grove/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jossmoff/grove.svg)](https://pkg.go.dev/github.com/jossmoff/grove)
[![Go Report Card](https://goreportcard.com/badge/github.com/jossmoff/grove)](https://goreportcard.com/report/github.com/jossmoff/grove)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Multi-repo agent workspaces backed by git worktrees.

Every worktree tool in the ecosystem is *one repo, N agents*. Grove is the
inversion: **N repos, one task**. It materialises a directory where the repos a
task touches sit as siblings — each a git worktree with a role — compiles the
context an agent needs to work across them, and gets out of the way.

```
~/grove/add-payments/
├── grove.toml       the manifest — a snapshot, the only durable state
├── AGENTS.md        generated: routes, never inlines
├── CONTEXT.md       durable notes, copied from the profile
├── PLAN.md          the task's working memory
├── backend/         worktree on grove/add-payments       (write)
└── client-sdk/      detached worktree                    (read)
```

The dominant case is cross-repo *reads*: you're editing `backend` and need
`client-sdk` on hand so the agent knows what it's actually calling. `read` repos get
a **detached** worktree — no branch, nothing to push, nothing left in
`refs/heads` afterwards. Edits there are discardable by construction.

## Install

```sh
brew install jossmoff/tap/grove          # or grab a release binary
go install github.com/jossmoff/grove/cmd/grove@latest
eval "$(grove init zsh)"             # gcd / gfin shell wrappers
```

## Usage

```sh
grove repo clone git@github.com:acme/backend.git   # into <root>/<host>/<owner>/<repo>

grove new add-payments -w backend -r client-sdk -t "Extend the payments API"
cd "$(grove path add-payments)" && claude          # grove's job is done

grove ls                                           # derived status, all groves
grove finish                                       # from inside; guards, prunes
```

Pin a read repo: `-r client-sdk@v2.1`. Reads fetch by default (a stale reference
hands the agent an interface that no longer exists); writes don't (branching
from what you last pulled is often deliberate). `--no-fetch` opts out.

### Profiles

Repo sets are stable; tasks aren't. When you've picked the same repos twice:

```sh
cd "$(grove path add-payments)"
grove profile save fullstack        # promote the grove you're standing in
grove new next-task -p fullstack    # no re-picking
```

A profile is a directory: `profile.toml` (the repo set, shorthand
`"client-sdk:read"` or longhand tables with `base`/`dir`/`depends_on`), `CONTEXT.md`
(durable notes about the combination), and `LOCAL.md` (machine paths,
credentials). Share one:

```sh
grove profile export fullstack > fullstack.toml   # inlines CONTEXT
grove profile import fullstack fullstack.toml     # or  ... -  for stdin
```

**`LOCAL.md` is never exported.** Locals hold credentials; a default that can
leak is not a default.

## Design

- **Derive, don't store.** The manifest holds what was asked for. Dirty state,
  ahead/behind, branch status are recomputed from git on every read — anything
  persisted that git already knows will drift.
- **Snapshot, don't reference.** `grove.toml` embeds the fully-resolved repo
  set. Profiles get edited; teardown must not break because a profile changed
  three weeks ago. `finish` reads the manifest and nothing else.
- **Route, don't inline.** `AGENTS.md` points at each repo's own instructions,
  namespaces colliding skills (`backend:test`, `client-sdk:test`), states dependency
  direction, and routes to your notes. Inlining five CLAUDE.mds produces a doc
  that's mostly irrelevant and stale on arrival.
- **The slug lives in the branch.** Git refuses one branch in two worktrees;
  `grove/<slug>` is what lets two live groves share a repo.
- **`finish` never deletes branches.** The work may be pushed but unmerged.
  It refuses on dirty or unpushed work unless `--force`.
- **Not an orchestrator.** Grove doesn't run agents, pick models, or manage
  sessions. `claude`, `aider`, `codex` all point at the same grove; the
  moment grove spawns agents it becomes the single-repo tools it was defined
  against.

## Documentation

Full docs live in [`docs/`](docs/) and render on GitHub:

- [Concepts](docs/concepts.md) — the vocabulary: grove, role, manifest, profile
- [Quick start](docs/quick-start.md) — install and first grove
- [CLI reference](docs/cli-reference.md) — every command and flag
- [Configuration reference](docs/config-reference.md) — the file formats
- [Architecture](docs/architecture.md) — how it's built
- [Design decisions](docs/decisions.md) — why it's built that way, including what was cut
- [Working with agents](docs/agents.md) — what grove generates and how agents use it
- [Keyboard reference](docs/keyboard.md) — interactive picker and dashboard keys

## Development

> **First build:** the interactive layer (`internal/tui`) pulls in Bubble Tea.
> Run `go mod download && go build ./...` once to fetch it — see
> [BUILD_NOTES.md](BUILD_NOTES.md). All other packages build with no extra step.


```sh
go test -race ./...   # unit + lifecycle tests against real git repos
go vet ./...
```

The lifecycle suite (`internal/workspace`) drives real repositories: roles,
branch-vs-detached, two groves sharing a repo, rollback on partial failure,
finish guards, base pinning, and a test that fails if anyone adds derived
state to the manifest.

Package docs are written for `go doc`: start with
`go doc github.com/jossmoff/grove/internal/workspace`.
