# Concepts

Grove has a small vocabulary. Learn these five terms and the rest of the
documentation reads easily.

## Grove

A **grove** is a directory containing several git worktrees checked out
together, plus the metadata and generated context that tie them into one unit of
work. It is the thing you create, work inside, and tear down.

A grove lives at `<grove_root>/<slug>/` — for example `~/grove/add-payments/`.
Inside it, each subdirectory is a worktree of a *separate* repository. They do
not share git history; a commit in one is unrelated to a commit in another.

A grove is **ephemeral**. You create one for a task, work in it, and finish it.
Its lifetime is days, not months.

## Repo and role

A grove is built from **repos** you already have cloned. Each repo enters a
grove with a **role**, and the role is the single most important concept in
grove:

| Role | What it gets | Use it for |
| --- | --- | --- |
| `write` | A worktree on a new branch `grove/<slug>` | Repos you intend to commit to |
| `read` | A **detached** worktree at a fixed commit | Repos you only need for context |

The `read` role is what makes grove more than a loop around `git worktree add`.
A detached worktree has no branch, so there is nothing to push and nothing left
in `refs/heads` when you are done. Any edit you make in a `read` repo is
discardable by construction. The dominant real-world case — editing one repo
while needing three others *on hand* so the agent knows what it is calling — is
exactly a single `write` beside several `read` repos.

## Manifest

The **manifest** is the file `grove.toml` at the root of every grove. It is the
only durable state grove keeps. It records what you asked for: which repos, which
roles, which branch, the task description, and which profile (if any) it came
from.

Two rules govern it, and both matter:

- **Derive, don't store.** The manifest holds only what you asked for. Anything
  git already knows — whether a repo is dirty, how far ahead of its upstream it
  is, which branch is checked out — is recomputed on read, never written down.
  Stored derived state drifts; git is the source of truth.
- **Snapshot, don't reference.** The manifest embeds the *fully resolved* repo
  set at the moment of creation. It does not point back at the profile it was
  built from. This is why editing a profile can never break the teardown of a
  grove created weeks earlier.

## Profile

A **profile** is a durable, reusable template for a repo set. Where a grove is
ephemeral, a profile is permanent. When you find yourself picking the same three
repos for the third time, you save them as a profile and stop picking.

A profile is a directory under `<config>/profiles/<name>/`:

| File | Purpose | Exported? |
| --- | --- | --- |
| `profile.toml` | The repo set: names, roles, pins | yes |
| `CONTEXT.md` | Durable notes about the *combination* — shareable structure like "changes flow shared → api → web" | yes |
| `LOCAL.md` | Machine-specific notes — key paths, ports, credentials | **never** |

The `CONTEXT.md` / `LOCAL.md` split exists because sharing forces it. Durable
notes always mix shareable structure with unshareable local detail; two files
mean the export default cannot leak a secret.

## How they fit together

```
profile  (durable template)          grove  (ephemeral instance)
─────────────────────────            ──────────────────────────
fullstack/                           add-payments/
  profile.toml     ── grove new ──▶    grove.toml   (snapshot of the profile)
  CONTEXT.md          --profile        CONTEXT.md   (copied in)
  LOCAL.md            fullstack         AGENTS.md    (generated)
                                        PLAN.md      (the task's working memory)
                                        backend/     (write worktree)
                                        client-sdk/  (read worktree)
```

You author a profile once. You spin up many groves from it. Each grove is a
self-contained snapshot that never looks back at the profile again.
