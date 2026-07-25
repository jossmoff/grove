# Design decisions

This is the reasoning behind grove's shape, including the ideas that were
designed in detail and then cut. It is written as a decision log because the
*rejected* options are as instructive as the chosen ones — grove is a tool that
got smaller the longer it was designed, and that is the most important fact about
it.

Each entry: the decision, the alternatives, and why.

---

## Roles: `write` and `read`, and `read` is detached

**Decision.** Two roles. `write` gets a worktree on a branch; `read` gets a
*detached* worktree with no branch.

**Why.** The dominant real-world case is not editing several repos at once — that
is rare. It is editing *one* repo while needing several others on hand for
context. A detached worktree is the exact right primitive for context: no branch
means nothing to push and nothing left in `refs/heads` at teardown, and any edit
is discardable by construction. This is the single design idea that makes grove
more than a loop around `git worktree add`.

**Rejected: a third role.** "Write onto an existing branch" (for review fixups)
and "pinned read" both looked like they wanted new roles. They didn't. The first
is a `branch` field on a `write` repo; the second is a `base` field on a `read`
repo. Two roles plus two optional fields express everything a third role would
have, without the combinatorial surface. See [config-reference](config-reference.md).

## Derive, don't store

**Decision.** The manifest holds only what the user asked for. Everything git
already knows — dirty state, ahead/behind, current branch — is recomputed on
every read.

**Why.** Any persisted copy of state git already owns will drift, and
reconciling that drift would become the tool's whole personality. This was
learned the expensive way: an earlier design had a lifecycle "board" with
webhooks pushing state into a store, which is precisely this trap with extra
infrastructure. `TestManifestHoldsNoDerivedState` enforces the rule in CI.

## Snapshot, don't reference

**Decision.** `grove.toml` embeds the fully-resolved repo set. It does not point
back at the profile it came from.

**Why.** Profiles are edited over time. If a grove referenced its profile,
editing that profile could break the teardown of a grove created weeks earlier.
A snapshot makes each grove self-contained: `finish` reads the manifest and
nothing else.

## Route, don't inline

**Decision.** The generated `AGENTS.md` points at each repo's own instruction
file rather than concatenating their contents.

**Why.** Inlining five repos' `CLAUDE.md` files produces a context document that
is mostly irrelevant to whatever you are actually doing and stale the moment any
of those files changes. Routing keeps the generated file small and always
current, and lets the agent load a repo's instructions only when it touches that
repo. (This is the same just-in-time principle that modern agent harnesses apply
to context generally.)

**Caveat, stated honestly.** This is the one load-bearing feature with no
empirical backing. Whether an agent actually reads `AGENTS.md` and behaves
differently is untested belief, not measured fact. See [../TRIAL.md](../TRIAL.md).

## The slug lives in the branch name

**Decision.** Write branches are named `grove/<slug>`.

**Why.** Git refuses to check out one branch in two worktrees. If the branch were
just named after the repo, two groves could never share a repo. Putting the slug
in the branch name is what lets `add-payments` and `add-auth` both hold a
worktree of `backend`.

## `finish` never deletes branches

**Decision.** Teardown removes worktrees and prunes, but leaves branches alone.

**Why.** The work may be pushed but unmerged. Deleting the branch at teardown
would be a silent way to lose committed work. The cost is that `refs/heads`
accumulates `grove/*` branches over time — an acceptable trade for never losing
work, and a future `grove gc` could offer opt-in cleanup.

## Not an orchestrator

**Decision.** Grove materialises the workspace and stops. It never runs an
agent, picks a model, or manages a session.

**Why.** The moment grove spawns agents, it owns model selection, session
lifecycle, output streaming, and a list of supported agent CLIs — and it becomes
the single-repo "one repo, N agents" tools it was defined against. *Not* owning
the agent is precisely what lets `claude`, `aider`, and `codex` all point at the
same grove. This is a feature disguised as a limitation.

---

## Ideas designed in full, then cut

These consumed real design effort before being rejected. They are recorded so
the reasoning is not lost and so they are not re-proposed.

### DAG workloads

**The idea.** Let users declare a dependency graph of CLI commands to run across
the grove — fmt, then test, ordered by repo dependency.

**Why cut.** The justifications collapsed under examination. Fan-out over repos
is a `for` loop, not a DAG. Role-awareness is a five-line guard. And ordering
from repo dependencies stops being useful exactly where it matters, because the
step *between* an upstream change and its downstream consumer is a
publish/version step a runner cannot execute — it is a human release process.
What survived was a single hypothetical `grove exec --scope writes --ordered`,
and even that died once the dominant case turned out to be a single `write` repo,
where there is nothing to order.

### Forge integration (issues/PRs needing review)

**The idea.** A command that pulls PRs needing review and groups them into a
grove.

**Why cut.** The grouping key — the shared head branch — only exists for PRs
grove itself created, which you already have a grove for. Other people's
cross-repo PRs, the ones you would actually want grouped, are not branch-named
that way and will not group. The feature works precisely where it is redundant
and fails precisely where it is needed. GitHub does not model cross-repo PR
linking, so there is no fallback key.

### A tree-sitter repo-map subsystem

**The idea.** Build a cross-repo symbol graph, rank it with personalized
PageRank, and feed the agent a budget-limited map of the most relevant symbols.

**Why cut as a *subsystem*.** The realisation that killed the custom build:
RepoMapper (the standalone port of Aider's repo map) takes a *directory*, not a
git repo. A grove is already a directory where several repos are siblings, so
pointing an existing single-tree tool at the grove root gives you a cross-repo
map for free. The novelty was never PageRank; it was that grove builds the
directory that makes an existing tool accidentally cross-repo. This remains a
possible future `grove map` — a ~30-line shell-out, not a subsystem — gated
behind the trial.

### Committed team config

**The idea.** A committed `env.json` (à la other multi-repo agent tools) so a
team shares one workspace definition.

**Why cut.** Grove's volatile, propagation-sensitive fields (pinned agent
versions, MCP routing) were themselves cut, leaving only the repo set, which
changes almost never. A point-in-time export/import is sufficient for a
near-static structure and avoids the live-propagation machinery. Profiles plus
`grove profile export`/`import` deliver the sharing without the commitment. See
[config-reference](config-reference.md#exporting-a-profile).

---

## The language pivot

Grove was first built in Rust — around 1,900 lines, fully tested, with a
six-target release pipeline — and then rebuilt in Go.

**Why the pivot was legitimate.** The tool does filesystem walks, git shell-outs,
TOML parsing, and (planned) a TUI. That is squarely Go's wheelhouse, with no
CPU-bound or memory-safety-critical surface to argue for Rust. The strongest Rust
argument — deployment — largely evaporated on inspection: grove has zero C
dependencies, so Rust cross-compilation was already easy, and Go's is easier
still.

**Why the pivot is also a caution.** The honest read is that the Rust version was
not hurting — no incident, no named friction, just "I think I'll deliver faster."
The thing that actually transferred was the *design*, which is why the Go rebuild
took a fraction of the time. The lesson worth keeping: **the design was the
expensive artefact; the code was cheap.** Which means the code is also cheap to
delete — see [../TRIAL.md](../TRIAL.md).

## The one thing to remember

Grove began as a DAG engine with forge integration and a tree-sitter map
subsystem, and ended as a worktree materialiser with a context generator and two
shell-outs. It did not shrink because the ideas were bad. It shrank because the
workspace *directory* turned out to be the entire invention, and everything else
was either already solved by an existing tool or better left to the agent.
