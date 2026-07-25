# Contributing

Thanks for looking at grove.

## Before you start

Grove is deliberately small and unproven — read [TRIAL.md](TRIAL.md) first.
New surface area (a map subcommand, a picker, forge integration) has a high bar
until the core is shown to be used. Bug fixes and correctness improvements to the
existing surface are always welcome.

## Ground rules

The design invariants in [CLAUDE.md](CLAUDE.md) are not preferences. A change
that violates one is wrong even if it compiles and passes — most are guarded by
tests, and `TestManifestHoldsNoDerivedState` will fail the build if you try to
persist derived state.

## Working locally

```sh
go test -race ./...     # unit + lifecycle tests against real git repos
go vet ./...
gofmt -l .              # must print nothing
golangci-lint run       # optional locally; required in CI
```

New behaviour needs a test in the same package. The lifecycle tests
(`internal/workspace`) are the model: assert on real git state, not on grove's
own claims about what it did.

## Commits and releases

- Label PRs so [release-drafter](.github/release-drafter.yml) can categorise
  them: `feature`/`enhancement`, `fix`/`bug`, `dependencies`, and
  `major`/`minor`/`patch` to steer the next version (default: patch).
- Releases are cut by pushing an `X.Y.Z` tag, which runs goreleaser.
