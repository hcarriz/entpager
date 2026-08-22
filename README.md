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

- Source compatibility with Go 1.21 or newer
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
	entpager.Page(2),
	entpager.Limit(25),
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

`Request` reads the `page` and `limit` query parameters:

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
	result, err := entpager.Paginate(
		r.Context(),
		client.User.Query().Order(user.ByID()),
		entpager.Request(r),
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

Options can also be combined and reused:

```go
pagination := entpager.Options(entpager.Page(2), entpager.Limit(50))
result, err := entpager.Paginate(ctx, client.User.Query(), pagination)
```

### Custom query-parameter names

`RequestWithNames` and `ValuesWithNames` accept immutable custom names. Empty
fields use the standard `page` and `limit` names:

```go
names := entpager.ParameterNames{
	Page:  "p",
	Limit: "page_size",
}

result, err := entpager.Paginate(
	r.Context(),
	client.User.Query().Order(user.ByID()),
	entpager.RequestWithNames(r, names),
)
```

## Safe limits

Entpager bounds work caused by ordinary and HTTP-derived pagination options:

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
In particular, `Limit(0)` and negative limits now use `DefaultLimit` rather
than disabling pagination, positive limits are capped unless `UnsafeLimit` is
used, and the exported defaults and parameter names are constants rather than
mutable variables.

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and project
rules. The test suite includes table-driven unit tests and native Go fuzzing for
query-parameter parsing and pagination invariants.

Please report vulnerabilities according to [SECURITY.md](SECURITY.md), not in a
public issue.

## License

Entpager is available under the [MIT License](LICENSE).
