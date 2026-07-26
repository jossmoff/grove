# Acceptance tests

Cucumber-style end-to-end tests for the grove CLI, implemented with
[godog](https://github.com/cucumber/godog).

Each scenario gets an isolated world — fresh temp directories, a dedicated
`GROVE_CONFIG`, and a dedicated cache dir — so no scenario can pollute another
and nothing leaks into the developer's own `~/src` or `~/grove`.

## Running locally

Build and run in one step (grove is compiled from source automatically):

```sh
go test -v -count=1 ./acceptance/...
```

Or point at a pre-built binary to skip the compile step:

```sh
GROVE_BINARY=$(go env GOPATH)/bin/grove go test -v -count=1 ./acceptance/...
```

Filter to a single feature file or scenario:

```sh
go test -v -count=1 ./acceptance/... -args --godog.paths=acceptance/features/grove_new.feature
go test -v -count=1 ./acceptance/... -args --godog.tags='@wip'
```

## Running in Docker

Docker gives full isolation: no host git config, no host grove config, a
deterministic Go toolchain. This is the same environment CI uses.

```sh
# Build the image (re-runs the Go build inside Docker)
make acceptance-docker

# Or run a single-feature subset
docker run --rm grove-acceptance \
  go test -v -count=1 ./acceptance/... \
    -args --godog.paths=acceptance/features/grove_finish.feature
```

## Structure

```
acceptance/
  features/          Gherkin feature files — living documentation
  suite_test.go      TestMain (binary resolution) + TestAcceptance
  world_test.go      World type: per-scenario temp dirs and helpers
  git_steps_test.go  Given/Then steps that seed repos and assert on git state
  cli_steps_test.go  When/Then steps that invoke grove and assert on output
  Dockerfile         Multi-stage: build grove → slim test runner with git
  README.md          This file
```

## Writing new scenarios

1. Add or extend a `.feature` file in `features/`.
2. Run the tests; godog prints `undefined` for any unmatched step.
3. Implement the missing step in `git_steps_test.go` or `cli_steps_test.go`
   and register it in the appropriate `register*Steps` function.

### Conventions

- **Given** steps seed state (repos, groves, profiles) using helpers, never
  by running grove commands — so that setup failures are clearly separated
  from the behaviour under test.
- **When** steps invoke exactly one grove command via `I run "grove ..."`.
- **Then** steps assert on stdout, stderr, exit code, git state, or the
  filesystem — never on grove's internal state directly.
- Avoid `--force` in assertions you didn't set up; prefer explicit Given steps.

## What is not covered

- **Interactive TUI** (`grove new` with no flags, `grove ls -i`): these open a
  terminal UI that cannot be driven headlessly. They are integration-tested
  manually and by the picker unit tests in `internal/tui`.
- **`grove repo clone`**: clone requires a URL with a host component
  (`git@host:owner/repo` or `https://host/…`). Local file paths are
  intentionally rejected by `repo.Parse`. Clone correctness is covered by
  `internal/workspace` lifecycle tests, which seed local origins the same way
  these tests do.
