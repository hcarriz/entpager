# Entpager

[![Go Reference](https://pkg.go.dev/badge/github.com/hcarriz/entpager.svg)](https://pkg.go.dev/github.com/hcarriz/entpager)
[![Go Report Card](https://goreportcard.com/badge/github.com/hcarriz/entpager)](https://goreportcard.com/report/github.com/hcarriz/entpager)
[![CI](https://github.com/hcarriz/entpager/actions/workflows/ci.yml/badge.svg)](https://github.com/hcarriz/entpager/actions/workflows/ci.yml)
[![Security](https://github.com/hcarriz/entpager/actions/workflows/security.yml/badge.svg)](https://github.com/hcarriz/entpager/actions/workflows/security.yml)

Entpager is a small, dependency-free Go package for paginating
[Ent](https://entgo.io) queries. It uses a structural interface rather than
importing Ent, so generated query types work without adding Ent as a dependency
of this module.

The package uses one extra result to determine whether another page exists. It
does not issue a count query.

## Requirements

- Source compatibility with Go 1.24 or newer, matching the minimum required by
  the latest stable Ent release
- A supported Go release with current security patches for production use
- An Ent-style query with `All`, `Offset`, and `Limit` methods

## Installation

```sh
go get github.com/hcarriz/entpager
```

## Usage

Pass a generated Ent query to `Paginate`:

```go
result, err := entpager.Paginate(
	ctx,
	client.User.Query().Order(user.ByName()),
	entpager.Pagination{Page: 2, Limit: 25},
)
if err != nil {
	return err
}

for _, u := range result.List {
	fmt.Println(u.Name)
}

if result.NextPage != 0 {
	fmt.Printf("next page: %d\n", result.NextPage)
}
```

`Response` contains the returned entities, the normalized page and limit, and
the next page number. `NextPage` is zero when there is no known next page.

### Read pagination from an HTTP request

`PaginationFromRequest` reads and normalizes the `page` and `limit` query
parameters before the query is built:

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
	pagination := entpager.PaginationFromRequest(r)
	result, err := entpager.Paginate(
		r.Context(),
		client.User.Query().Order(user.ByID()),
		pagination,
	)
	if errors.Is(err, entpager.ErrOffsetTooLarge) {
		http.Error(w, "page is too large", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "could not list users", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}
```

Use a deterministic order when paginating. Without one, database row order is
not guaranteed and records may be repeated or skipped between requests.

`Pagination` has a useful zero value. Constructing it directly with omitted,
zero, or negative fields uses page 1 and `DefaultLimit`; limits above
`MaximumLimit` are clamped:

```go
result, err := entpager.Paginate(ctx, client.User.Query(), entpager.Pagination{})
```

### Custom query-parameter names

`PaginationFromRequestWithNames` and `PaginationFromValuesWithNames` accept
immutable custom names. Empty fields use the standard `page` and `limit`
names:

```go
names := entpager.ParameterNames{
	Page:  "p",
	Limit: "page_size",
}

pagination := entpager.PaginationFromRequestWithNames(r, names)
result, err := entpager.Paginate(
	r.Context(),
	client.User.Query().Order(user.ByID()),
	pagination,
)
```

## Compose pagination with Ent filters

Embed `Pagination` in an application-specific parameter type. Parse and
validate filters in the application, apply Ent's generated predicates, and
then pass only the pagination value to Entpager:

```go
type UserParams struct {
	entpager.Pagination
	Name string
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	params := UserParams{
		Pagination: entpager.PaginationFromRequest(r),
		Name:       r.URL.Query().Get("name"),
	}

	query := client.User.Query()
	if params.Name != "" {
		query = query.Where(user.NameContainsFold(params.Name))
	}
	result, err := entpager.Paginate(
		r.Context(),
		query.Order(user.ByID()),
		params.Pagination,
	)
	// Handle err and encode result.
}
```

Keeping filters outside Entpager preserves Ent's generated, type-safe
predicates and avoids a dynamic string-to-query layer. It also lets the same
`Pagination` type compose with different schemas and filter models.

## Safe limits

Entpager bounds work caused by directly constructed and HTTP-derived pagination
values:

- `DefaultLimit` is `25`.
- `MaximumLimit` is `100`.
- Missing, malformed, zero, and negative limits use `DefaultLimit`.
- Limits greater than `MaximumLimit` are clamped to the maximum.
- Missing, malformed, zero, and negative pages use page `1`.
- `MaximumOffset` is `1,000,000`.
- `Paginate` returns `ErrOffsetTooLarge` before modifying or executing the query
  when the requested offset exceeds that maximum.
- Offset and lookahead arithmetic are checked before use.
- HTTP query parameters never bypass these limits.

These controls reduce the unrestricted resource-consumption risk described by
[OWASP API4:2023](https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/).
Applications should still enforce request rate limits, deadlines, authorization,
and database-specific resource controls.

### Explicit unsafe limit

Trusted callers can deliberately exceed `MaximumLimit` with `UnsafeLimit`:

```go
result, err := entpager.Paginate(
	ctx,
	client.User.Query().Order(user.ByID()),
	entpager.Pagination{},
	entpager.UnsafeLimit(500),
)
```

`UnsafeLimit` bypasses only `MaximumLimit`; it does not bypass
`MaximumOffset`. Non-positive values and the largest platform `int` return
`ErrInvalidLimit`. Never construct an unsafe limit directly from untrusted
request data.

## Errors

Use `errors.Is` to handle the exported sentinel errors:

- `ErrInvalidLimit` means an explicit `UnsafeLimit` cannot be executed safely.
- `ErrOffsetTooLarge` means the requested page and limit exceed
  `MaximumOffset`.

Errors returned by the underlying Ent query are returned unchanged.

## Pre-v1 compatibility

The package is not yet v1, so security and API improvements may be breaking.
The minimum Go version has changed from 1.21 to 1.24 to match the latest stable
Ent release. Callers using an older Go toolchain must upgrade it before using
the next Entpager release.

`Paginate` now requires a dedicated `Pagination` argument; the former
`Page`, `Limit`, `Values`, `Request`, and `Options` option helpers have been
removed. Migrate direct values and HTTP requests as follows:

```go
// Before:
entpager.Paginate(ctx, query, entpager.Page(2), entpager.Limit(25))

// Now:
entpager.Paginate(ctx, query, entpager.Pagination{Page: 2, Limit: 25})

// Before:
entpager.Paginate(ctx, query, entpager.Request(r))

// Now:
entpager.Paginate(ctx, query, entpager.PaginationFromRequest(r))
```

Custom-name callers should replace `RequestWithNames` or `ValuesWithNames`
with `PaginationFromRequestWithNames` or
`PaginationFromValuesWithNames`. `UnsafeLimit` remains an option and follows
the required pagination argument.

## How it integrates with Ent

Entpager accepts any query satisfying this interface:

```go
type Ent[Self, Entity any] interface {
	All(context.Context) ([]Entity, error)
	Offset(int) Self
	Limit(int) Self
}
```

Generated Ent query types already have this shape. The interface also makes the
package straightforward to test with a small fake and keeps this module free of
third-party dependencies.

## Versioning

Release Please creates the initial `v0.1.0` tag and subsequent releases from
Conventional Commits on `master`. During v0, `fix` and `perf` commits produce
patch releases; `feat` commits and approved breaking changes produce minor
releases. Breaking changes must include migration guidance. Beginning with v1,
Entpager will preserve source compatibility throughout the v1 module line.

GitHub Releases are the release-note source of truth; the repository does not
maintain a generated changelog. See
[GitHub Releases](https://github.com/hcarriz/entpager/releases) for published
versions.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and project
rules. The test suite includes table-driven unit tests and native Go fuzzing for
query-parameter parsing and pagination invariants.

Please report vulnerabilities according to [SECURITY.md](SECURITY.md), not in a
public issue.

## License

Entpager is available under the [MIT License](LICENSE).
