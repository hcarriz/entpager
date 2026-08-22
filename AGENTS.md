# Repository guide

## Purpose

Entpager is a small, dependency-free Go library that adds offset pagination to
Ent queries. Keep the package focused: it should depend on the behavior exposed
by the `Ent` interface, not on Ent itself.

This file applies to the entire repository.

## Repository layout

- `query.go` contains package documentation, the public API, and pagination
  implementation.
- `query_test.go` contains black-box tests in package `entpager_test`.
- `fuzz_test.go` contains fuzz tests for parsing and pagination invariants.
- `README.md` is user-facing documentation.
- `CONTRIBUTING.md` describes the contributor workflow.

Prefer this flat layout while the package remains small. Add a package or
directory only when it has a distinct, independently useful responsibility.

## Working agreements

- Support Go 1.21 and newer.
- Keep production and test code dependency-free; use the standard library.
- Do not add a direct dependency on Ent. Preserve the small structural `Ent`
  interface so generated Ent query types satisfy it implicitly.
- Keep public APIs idiomatic, small, and documented with Go doc comments.
- Treat API compatibility thoughtfully, but breaking changes are allowed before
  v1 when they make the API safer or clearer. Call out every breaking change in
  the README and handoff summary.
- Update tests and user-facing documentation whenever behavior changes.
- Preserve unrelated user changes in the working tree.
- Write every commit message using Conventional Commits 1.0.0. Use the form
  `<type>[optional scope][!]: <description>` with a lowercase type and a concise
  imperative description.

## Pagination and security invariants

Current behavior and intended behavior are documented separately in the README.
Do not describe an intended API as already available.

When implementing the planned bounded-limit behavior:

- Keep `DefaultLimit` at 25 unless the change is deliberate and documented.
- Bound ordinary and HTTP-derived limits to `MaximumLimit`, initially 100.
- Treat missing, malformed, zero, and negative HTTP limit values as
  `DefaultLimit`.
- Clamp page values below 1 to page 1.
- Permit limits above the maximum only through an explicit API whose name and
  documentation make the unsafe resource-consumption tradeoff obvious.
- Never enable the unsafe limit implicitly from untrusted query parameters.
- Guard page/limit offset arithmetic against integer overflow.
- Continue fetching one extra record for bounded queries to determine
  `NextPage` without a count query.

## Implementation style

- Favor direct control flow and useful zero values over abstractions.
- Handle errors explicitly and wrap them only when added context helps callers.
- Accept `context.Context` as the first parameter for operations that perform a
  query.
- Use table-driven tests with descriptive subtest names.
- Keep black-box tests in `entpager_test` unless white-box access is essential.
- Use `t.Parallel` only when test data and collaborators are isolated.
- Avoid generated mocks and assertion frameworks; small fakes are preferred.
- Add fuzz seeds for boundary values and previously discovered regressions.
- Express fuzz checks as stable invariants rather than duplicating the
  implementation. Keep inputs bounded only where an explicitly documented,
  not-yet-fixed limitation would otherwise make the suite permanently fail.
- When a fuzz failure is fixed, retain its minimized input as a regression seed
  or corpus entry.

## Required verification

After changing Go code, run:

```sh
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go test -shuffle=on -count=1 ./...
```

Run each affected fuzz target for at least 30 seconds. For example:

```sh
go test -fuzz=FuzzValues -fuzztime=30s .
```

Use `go test -cover ./...` to find meaningful gaps, but do not add tests that
only execute lines without asserting behavior.

After documentation-only changes, inspect rendered structure and run:

```sh
git diff --check
```

Before finishing any change, verify that examples match the public API and that
the README clearly distinguishes current behavior from planned behavior.

## Code review rules

- Flag any path by which request-controlled input can disable or bypass the
  pagination limit without an explicit unsafe opt-in.
- Flag unbounded or overflowing offset calculations.
- Flag new dependencies, including test-only dependencies.
- Flag direct coupling to generated Ent packages or schemas.
- Flag public behavior changes that lack tests and documentation.
- Flag commit messages that do not follow Conventional Commits.

## Commit messages

Use `feat` for new user-visible behavior and `fix` for bug fixes. Other useful
types include `docs`, `test`, `refactor`, `perf`, `build`, `ci`, and `chore`.
Scopes are optional and should name a meaningful part of this small codebase.

Examples:

```text
docs: document pagination safety policy
test: add fuzz coverage for query values
feat!: bound request-derived limits by default
```

Mark breaking changes with `!` before the colon or an uppercase
`BREAKING CHANGE:` footer. Include a footer when callers need migration details,
even though breaking changes are currently allowed before v1.
