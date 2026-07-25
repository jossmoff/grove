# Quick start

## Install

```sh
# from source (until the first release is tagged)
go install github.com/jossmoff/grove/cmd/grove@latest

# or build a local checkout
go build -o grove ./cmd/grove && mv grove ~/go/bin/
```

Ensure `~/go/bin` is on your `PATH`. Verify:

```sh
grove --version
```

## One-time setup

Write the default config, then edit it if your repos live somewhere other than
`~/src`:

```sh
mkdir -p "$(dirname "$(grove config --path)")"
grove config > "$(grove config --path)"
$EDITOR "$(grove config --path)"      # check `root` and `grove_root`
```

Wire up the shell integration — this is what makes `cd`-into-a-grove work, since
a binary cannot move your shell:

```sh
eval "$(grove init zsh)"              # add to ~/.zshrc to persist (or bash/fish)
```

That gives you two functions: `gcd <slug>` (cd into a grove) and `gfin` (finish
a grove and escape the deleted directory).

## Your first grove

Grove works from repos already cloned under `root`. Clone through grove so they
land in the canonical `<root>/<host>/<owner>/<repo>` layout:

```sh
grove repo clone git@github.com:acme/backend.git
grove repo clone git@github.com:acme/client-sdk.git
grove repo ls                        # confirm both are indexed
```

Create a grove — `backend` to edit, `client-sdk` for read-only context:

```sh
grove new add-payments -w backend -r client-sdk -t "Extend the payments API"
gcd add-payments
```

You are now inside a directory with `backend/` (on branch `grove/add-payments`)
and `client-sdk/` (detached, read-only), plus a generated `AGENTS.md`. Point any agent
at it:

```sh
claude          # or aider, or codex — grove does not care which
```

## Finishing

From inside the grove:

```sh
gfin                 # refuses if you have uncommitted or unpushed work
```

Push first, or `grove finish --force` if you mean to discard. Branches are never
deleted, so your committed work is safe even after finishing.

## Next: profiles

When you have picked the same repos twice, stop picking:

```sh
gcd add-payments
grove profile save fullstack         # promote this grove's repo set
grove new next-task -p fullstack     # reuse it, no -w/-r needed
```

See [Concepts](concepts.md) for the model and
[CLI reference](cli-reference.md) for every command.
