# CLAUDE.md

Agent instructions for working on grove itself.

Grove is a CLI that materialises multi-repo git-worktree workspaces and compiles
agent context for them. This file is deliberately short and routes to the code —
which is the same philosophy grove applies to the workspaces it generates. If you
find yourself wanting to paste code in here, put it in a doc comment instead.

## Where things are

The `cmd/grove` binary is a thin wire; all behaviour is in `internal/`. Read the
package doc comment before the code — each one states the *why*, and the why is
load-bearing here.

- `internal/gitx` — every git call. The one rule: **shell out, never a git
  library.** `git worktree add` carries branch/tracking/sparse-checkout/repair
  semantics not worth reimplementing.
- `internal/manifest` — `grove.toml`. Read its doc comment before touching it.
- `internal/profile` — durable templates; the shorthand/longhand TOML decoder is
  the trickiest code in the repo.
- `internal/workspace` — `Create` (with rollback) and `Finish` (with guards).
  The lifecycle tests live beside it and are the real spec.
- `internal/contextgen` — generates `AGENTS.md`. Write-only output.
- `internal/index`, `internal/status` — repo discovery and derived state.
- `internal/cmd` — flags → behaviour. The only package that imports `cli`.

## Invariants — do not break these

These are the design, not preferences. A change that violates one is wrong even
if it compiles and passes.

1. **Derive, don't store.** The manifest holds only what the user asked for.
   Anything git knows (dirty, ahead/behind, branch state) is recomputed on read.
   `TestManifestHoldsNoDerivedState` fails the build if you add a derived field —
   that test is a feature, not an obstacle.
2. **Snapshot, don't reference.** `grove.toml` embeds the fully-resolved repo
   set. `Finish` reads the manifest and nothing else — never the profile it came
   from. Profiles change; teardown must not.
3. **`read` repos are detached.** No branch, nothing in `refs/heads`. If you find
   yourself giving a read repo a branch, you have misunderstood the role.
4. **`finish` never deletes branches.** Work may be pushed but unmerged.
5. **Route, don't inline.** `contextgen` emits pointers to per-repo files, never
   their contents.
6. **Not an orchestrator.** Grove never spawns an agent. The moment it does, it
   becomes the single-repo tools it exists to counter.

## Working here

- `go test -race ./...` before anything is considered done. The lifecycle suite
  drives real git repositories; if it is slow, that is why.
- `gofmt` and `go vet` are clean and stay clean.
- New behaviour needs a test in the same package. The lifecycle tests are the
  model: assert on real git state (`git rev-parse`, `git branch --list`), not on
  grove's own claims about what it did.
- Keep dependencies minimal — currently three (toml, cli, go-cmp). Adding one is
  a decision, not a reflex.
- stdout is the answer, stderr is the narration. Every command prints exactly one
  machine-readable thing to stdout; progress and confirmations go to stderr.

## Status

Grove is unproven — see `TRIAL.md`. Do not add speculative features (a map
subcommand, a picker, forge integration) until the trial says the core is used.
The bar for new surface area is high on purpose.
