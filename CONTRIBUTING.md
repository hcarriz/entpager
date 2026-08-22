# Contributing to Entpager

Thanks for helping improve Entpager. Contributions should keep the package
small, predictable, and dependency-free.

## Development requirements

- Go 1.21 or newer
- Git

Go 1.21 is the source-compatibility floor. Run security scans and production
software with a currently supported Go release containing the latest security
patches.

Clone the repository, make the smallest focused change that solves the problem,
and add or update tests for observable behavior.

## Design expectations

- Use only the Go standard library in production and tests.
- Do not import Ent into this module. Generated Ent queries integrate through
  the package's structural `Ent` interface.
- Prefer a flat package layout while the project remains small.
- Keep exported APIs and behavior documented.
- Keep `Pagination` as the dedicated input to `Paginate`. Request and URL
  adapters return pagination values; options are reserved for explicit
  exceptional behavior such as `UnsafeLimit`.
- Leave schema-specific filtering and sorting to callers and Ent's generated
  predicates.
- Use deterministic query ordering in examples.
- Update the README when public behavior changes.

The module is pre-v1, so a well-justified breaking change is possible. Explain
the motivation, migration path, and security or usability benefit in the change
description. Avoid unrelated API cleanup in the same change.

## Pagination safety

Pagination values often originate from untrusted HTTP requests. Changes must
keep ordinary limits bounded and must not let request parameters select an
unsafe, unbounded, or above-maximum limit.

An API that bypasses the maximum limit must require an explicit opt-in and must
make the resource-consumption risk obvious in its name and documentation. Tests
should cover defaulting, clamping, the unsafe opt-in boundary, and offset
overflow behavior.

Keep `DefaultLimit`, `MaximumLimit`, `MaximumOffset`, parameter names, and custom
parameter-name values immutable. Validate offset arithmetic before modifying or
executing a query. See the README for the complete public contract.

## Reporting vulnerabilities

Do not open a public issue for a suspected vulnerability. Follow
[SECURITY.md](SECURITY.md) and use GitHub private vulnerability reporting so the
maintainer can investigate and coordinate a fix before disclosure.

## Tests

Write table-driven tests with descriptive case names. Prefer black-box tests in
package `entpager_test` and small hand-written fakes over mocks or test
frameworks. Tests should include errors, nil inputs, boundary values, and
behavioral invariants where they apply.

Fuzz tests should seed representative defaults, malformed values, boundaries,
and fixed regressions. A fuzz target should assert properties that remain true
across implementations instead of reproducing the implementation's exact
steps. When fixing a fuzz-discovered failure, preserve the minimized input as a
regression seed or corpus entry.

Run these checks before submitting a change:

```sh
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go test -shuffle=on -count=1 ./...
go test -cover ./...
git diff --check
```

Run each fuzz target affected by the change separately. The package currently
has this target:

```sh
go test -fuzz=FuzzPaginationFromValues -fuzztime=30s .
```

Longer fuzzing is encouraged before releases and after changes to parsing,
normalization, limit enforcement, or offset arithmetic.

For a documentation-only change, `git diff --check` is sufficient, but ensure
all examples still match the public API.

## Commit messages

All commits must follow
[Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

```text
<type>[optional scope][!]: <description>
```

Use lowercase types and a concise imperative description. Use `feat` for new
user-visible behavior and `fix` for bug fixes. Common additional types are
`docs`, `test`, `refactor`, `perf`, `build`, `ci`, and `chore`.

Examples:

```text
docs: explain unsafe limit opt-in
test: add malformed value fuzz seeds
fix: preserve the first page for negative input
feat!: enforce the maximum page size
```

Breaking changes must use `!` before the colon or an uppercase
`BREAKING CHANGE:` footer. Add a footer with migration details when the subject
alone does not fully explain the impact:

```text
feat!: bound request-derived limits by default

BREAKING CHANGE: zero and negative HTTP limits now use DefaultLimit instead of
disabling pagination.
```

## Change and review scope

Keep commits focused and explain user-visible behavior changes. Reviewers should
be able to determine:

- what changed and why;
- whether the change is breaking;
- how limits from untrusted input remain bounded;
- which tests cover the behavior; and
- which documentation was updated.
