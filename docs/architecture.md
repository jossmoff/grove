# Architecture

This document describes how grove is built: the package layout, the data flow,
and the load-bearing structural choices. For *why* each choice was made, see
[Design decisions](decisions.md); this document explains *what* the shape is.

## Shape at a glance

Grove is a single Go binary. `cmd/grove` is a thin wire; all behaviour lives in
`internal/` packages that never import the CLI framework. The dependency
direction is strictly downward — `cmd` depends on the internals, the internals
depend on each other in a DAG, and nothing depends on `cmd`.

```
cmd/grove/main.go          wires subcommands, stamps version
  └─ internal/cmd          flags → behaviour  (the only package importing urfave/cli)
       ├─ internal/workspace   Create (with rollback), Finish (with guards)
       │    ├─ internal/gitx        every git invocation
       │    ├─ internal/manifest    grove.toml — the snapshot
       │    ├─ internal/contextgen  generates AGENTS.md
       │    └─ internal/status      derived state (never stored)
       ├─ internal/profile      durable templates; shorthand/longhand TOML
       ├─ internal/index        repo discovery + TTL cache
       ├─ internal/repo         remote URL → canonical path
       └─ internal/config       ~/.config/grove/config.toml
```

Three dependencies total: `BurntSushi/toml`, `urfave/cli/v3`, `google/go-cmp`
(tests only). Adding a fourth is a decision, not a reflex.

## The four lifecycle phases

Grove's whole job is four phases, and only two of them are grove's code:

```
1 MATERIALISE   grove   index → resolve → worktrees + manifest + context
2 PREPARE       grove   (optional) generate a repo map into the context
3 WORK          NOT grove   cd into the grove; point any agent at it
4 TEARDOWN      grove   remove worktrees → prune → delete directory
```

Phase 3 is deliberately a hole. Grove never spawns an agent. See
[decisions §"Not an orchestrator"](decisions.md#not-an-orchestrator).

## Package responsibilities

### `internal/gitx`

Every git call in the program goes through here, and every one shells out to the
`git` binary rather than using a git library. `git worktree add` carries branch
creation, tracking setup, sparse-checkout and `repair` semantics that a library
would force us to reimplement. Read-side helpers (`DirtyCount`, `AheadBehind`)
also shell out; if that ever gets slow across many repos, the fix is a cache, not
a library.

Errors always carry git's stderr, because "exit status 128" alone is useless.

### `internal/repo`

Maps a remote URL to its canonical position under the root:
`git@github.com:acme/backend.git` → `github.com/acme/backend`. The hard case is
scp-style remotes, which are *not* URLs and which `net/url` silently mangles —
handled explicitly with a dedicated regex. Nested GitLab groups
(`group/subgroup/repo`) are preserved rather than flattened.

### `internal/index`

Discovers repos by walking the root and recording every git repo without
descending into it. The filesystem *is* the registry: a clone at
`<root>/<host>/<owner>/<repo>` is the record of that remote; there is no separate
registry file. Metadata (last-commit time, remote URL) is gathered on a bounded
worker pool — this is subprocess-bound work, so a semaphore of a few workers is
the right shape. Results are cached with a TTL, so the index is wrong for at most
`index_ttl_secs` rather than wrong forever.

### `internal/manifest`

Defines `grove.toml` and the cwd-discovery walk (`Discover`) that lets every
command find its enclosing grove by walking up for the manifest, the way git
walks up for `.git`. The manifest is a **snapshot**: fully resolved, never
referencing the profile it came from. See [Concepts](concepts.md#manifest).

### `internal/profile`

Durable templates. The interesting code is the TOML decoder that accepts both
shorthand (`"client-sdk:read"`) and longhand (`{ at = "...", role = "read", base =
"v2.1" }`) in the same array — the cargo-dependency pattern, terse by default,
expanding when needed. This is the trickiest code in the repo and has the
densest tests.

### `internal/workspace`

The lifecycle. `Create` validates everything before mutating anything, then
materialises worktrees, and on any failure rolls back every worktree it created
plus the directory — a half-built grove is worse than none. `Finish` refuses on
uncommitted or unpushed work unless forced, removes and prunes each worktree, and
deletes the directory — but never deletes branches, because the work may be
pushed-but-unmerged.

### `internal/contextgen`

Generates `AGENTS.md`. It **routes** rather than inlines: the generated file
points at each repo's own instruction file instead of reproducing its contents,
namespaces skills that would otherwise collide (`backend:test`, `client-sdk:test`),
states dependency direction, and points at your notes. Generated files are
write-only — grove never reads them back, so hand-editing them is pointless;
edit the profile or the manifest and re-run `grove sync`.

### `internal/status`

Derived state, computed in parallel across a grove's repos on every read.
Nothing here is ever persisted. If it becomes slow, it gets a TTL cache like the
index — it never gets written into the manifest.

## State model

Grove keeps exactly two tiers of state and one escape hatch:

| Tier | Where | Read back as truth? |
| --- | --- | --- |
| **Declared** | `config.toml`, `profile.toml`, `grove.toml` | Yes |
| **Rendered** | `AGENTS.md`, `PLAN.md` | Never — grove only writes these |
| **Escape hatch** | `CONTEXT.md`, `LOCAL.md` | Hand-written; grove routes to, never parses |

The one rule that ties it together: **grove never reads its own generated
output.** Anything grove needs to know is in a declared file or derivable from
git. This is what keeps the system from drifting.

## Concurrency

There is no async runtime and no goroutine-heavy design. Two places use bounded
parallelism, both subprocess-bound: index metadata gathering and status
derivation. Both use a `sync.WaitGroup` over a fixed set of work, with a
semaphore where the fan-out is large. This is deliberate — the workload is
`exec` and `stat`, not IO concurrency, so a thread pool is the correct shape and
an async runtime would be complexity for nothing.

## Testing strategy

Three levels, described fully in [CONTRIBUTING](../CONTRIBUTING.md):

1. **Table-driven unit tests** for pure logic — remote parsing, the profile
   decoder.
2. **Lifecycle integration tests** (`internal/workspace`) that drive *real* git
   repositories and assert on real git state (`git rev-parse`, `git branch
   --list`), never on grove's own claims about what it did. Mocking git would
   only test the mock.
3. **Invariant guards** — e.g. `TestManifestHoldsNoDerivedState` fails the build
   if anyone adds a derived field to the manifest. The test encodes the design
   rule so the rule cannot silently erode.
