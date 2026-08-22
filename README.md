# Entpager

[![Go Reference](https://pkg.go.dev/badge/github.com/hcarriz/entpager.svg)](https://pkg.go.dev/github.com/hcarriz/entpager)
[![Go Report Card](https://goreportcard.com/badge/github.com/hcarriz/entpager)](https://goreportcard.com/report/github.com/hcarriz/entpager)

Entpager is a small, dependency-free Go package for paginating
[Ent](https://entgo.io) queries. It uses a structural interface rather than
importing Ent, so generated query types work without adding Ent as a dependency
of this module.

The package uses one extra result to determine whether another page exists. It
does not issue a count query.

## Requirements

- Go 1.21 or newer
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

## Current behavior

The current pre-v1 implementation behaves as follows:

- The default page is `1` and the default limit is `25`.
- Pages below `1` are clamped to `1`.
- A malformed or missing HTTP limit uses the default limit.
- A malformed or missing HTTP page uses page `1`.
- A limit of `0` or less disables the limit and returns all remaining results.
- Positive limits are not currently capped.
- `Request(nil)` is valid and uses the defaults.

The last two points mean applications must not pass untrusted numeric limits to
the current release without validating them first. An attacker-controlled
unbounded or very large result set can consume excessive database, memory, and
network resources. This is the class of risk described by
[OWASP API4: Unrestricted Resource Consumption](https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/).

## Pre-v1 direction

Before v1, the limit behavior is intended to become bounded by default:

- `DefaultLimit` remains `25`.
- `MaximumLimit` defaults to `100`.
- Missing, malformed, zero, and negative HTTP limits use `DefaultLimit`.
- Limits greater than `MaximumLimit` are clamped to the maximum.
- Page values below `1` continue to clamp to page `1`.
- Offset arithmetic is checked to prevent integer overflow.
- Callers may explicitly opt into a limit above the maximum through an API that
  clearly identifies the resource-consumption risk as unsafe.
- HTTP query parameters never enable that unsafe behavior implicitly.

These bullets describe planned behavior, not the API in the current release.
Because the module is not yet v1, changes that improve safety or clarity may be
breaking.

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

## License

Entpager is available under the [MIT License](LICENSE).
