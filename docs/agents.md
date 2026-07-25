# Working with agents

Grove's second job — after materialising the worktrees — is compiling the
context an agent needs to work across them. This page explains what it generates
and how an agent is meant to use it.

Grove does **not** run the agent. You `cd` into a grove and launch whatever agent
CLI you use; grove has already laid out the directory and written the context.

## What grove generates

At the root of every grove:

| File | Written by | Purpose |
| --- | --- | --- |
| `AGENTS.md` | grove (regenerable) | The routing context — read this first |
| `PLAN.md` | grove (seeded once) | The task's working memory; yours to maintain |
| `CONTEXT.md` | copied from profile | Durable notes about the repo combination |
| `LOCAL.md` | copied from profile | Machine-specific notes |

## AGENTS.md routes; it does not inline

The generated `AGENTS.md` deliberately does *not* contain the full instructions
for each repo. It contains:

- A table of the repos, their roles, their branches, and a pointer to each
  repo's own instruction file — read that file when you first touch that repo.
- Rules: which directories are writable, which are read-only context, and the
  hard git constraint that you cannot check out a branch already checked out in a
  sibling.
- Dependency direction, when `depends_on` is set — including the warning that an
  upstream interface change cannot simply be edited in both places, because a
  publish/version step sits between them.
- Namespaced skills, when repos define them, so `backend:test` and `client-sdk:test`
  do not collide.
- Pointers to `CONTEXT.md`, `LOCAL.md`, and `PLAN.md`.

The reason it routes rather than inlines: a document that concatenates five
repos' instructions is mostly irrelevant to any given task and stale the moment
any of those files change. Pointers stay small and current.

## Regenerating context

`AGENTS.md` is generated output — grove never reads it back, so editing it by
hand is pointless. To change it, edit the source (the manifest, or a profile's
notes) and run:

```sh
grove sync
```

## Configuring for a specific agent CLI

Different agent tools look for different filenames. If yours reads something
other than `AGENTS.md`, set `agent_file` in `config.toml`:

```toml
agent_file = "CLAUDE.md"
```

That single setting is the whole integration surface. Grove writes the routing
context to whatever filename you name; the agent finds it because it is the file
the agent already looks for.

## PLAN.md and long tasks

`PLAN.md` is seeded once and is yours thereafter. It exists so that a long task
has memory outside the conversation: if the agent's context window compacts or
the session restarts, the plan on disk survives. Keep it updated as you go — it
is the task's memory, not a grove-managed file.

## A candid note

Whether an agent actually reads `AGENTS.md` and changes its behaviour is, at time
of writing, an untested assumption — the most load-bearing untested assumption in
the whole tool. If you use grove, the single most useful thing you can report is
whether the agent visibly followed a rule the generated context stated. See
[../TRIAL.md](../TRIAL.md).
