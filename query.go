// Package entpager provides dependency-free offset pagination for Ent-style
// queries.
//
// Paginate accepts any query with All, Offset, and Limit methods matching the
// Ent interface. Options can set the page and limit directly or derive them
// from URL values and HTTP requests. A bounded query fetches one extra entity
// to determine whether Response.NextPage should be set, avoiding a separate
// count query.
package entpager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
)

// Ent is the subset of an Ent query needed by Paginate.
type Ent[Self, Entity any] interface {
	All(context.Context) ([]Entity, error)
	Offset(int) Self
	Limit(int) Self
}

const (
	// DefaultLimit is used when a limit is absent, malformed, zero, or negative.
	DefaultLimit = 25
	// MaximumLimit is the largest limit accepted without UnsafeLimit.
	MaximumLimit = 100
	// MaximumOffset is the largest database offset Paginate will request.
	MaximumOffset = 1_000_000
	// ParameterPage is the default page query-parameter name.
	ParameterPage = "page"
	// ParameterLimit is the default limit query-parameter name.
	ParameterLimit = "limit"
)

var (
	// ErrInvalidLimit indicates that UnsafeLimit received a value that cannot be
	// used safely by the pagination implementation.
	ErrInvalidLimit = errors.New("entpager: invalid limit")
	// ErrOffsetTooLarge indicates that a requested page would exceed
	// MaximumOffset or overflow an int.
	ErrOffsetTooLarge = errors.New("entpager: offset too large")
)

// ParameterNames specifies custom URL query-parameter names. Empty fields use
// ParameterPage and ParameterLimit, making the zero value useful.
type ParameterNames struct {
	Page  string
	Limit string
}

func (n ParameterNames) defaults() ParameterNames {
	if n.Page == "" {
		n.Page = ParameterPage
	}
	if n.Limit == "" {
		n.Limit = ParameterLimit
	}
	return n
}

type props struct {
	page  int
	limit int
}

func (p props) offset() (int, error) {
	pageOffset := p.page - 1
	if pageOffset == 0 {
		return 0, nil
	}
	if p.limit > MaximumOffset/pageOffset {
		return 0, fmt.Errorf(
			"%w: page %d with limit %d exceeds maximum offset %d",
			ErrOffsetTooLarge,
			p.page,
			p.limit,
			MaximumOffset,
		)
	}
	return p.limit * pageOffset, nil
}

// Option configures a call to Paginate.
type Option interface {
	apply(*props) error
}

type option func(*props) error

func (o option) apply(p *props) error {
	return o(p)
}

// Options combines multiple options into one option. Nil options are ignored.
func Options(opts ...Option) Option {
	return option(func(p *props) error {
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			if err := opt.apply(p); err != nil {
				return err
			}
		}

		return nil
	})
}

// Values sets the page and limit from url.Values using ParameterPage and
// ParameterLimit.
//
// Missing or malformed values use safe defaults. Numeric values are normalized
// by Page and Limit.
func Values(vals url.Values) Option {
	return ValuesWithNames(vals, ParameterNames{})
}

// ValuesWithNames sets the page and limit from url.Values using custom
// parameter names. Empty names use ParameterPage and ParameterLimit.
func ValuesWithNames(vals url.Values, names ParameterNames) Option {
	names = names.defaults()

	raw := vals.Get(names.Limit)
	limit, err := strconv.Atoi(raw)
	if err != nil {
		limit = DefaultLimit
	}

	rawPage := vals.Get(names.Page)
	page, _ := strconv.Atoi(rawPage)
	return Options(Limit(limit), Page(page))
}

// Request sets the page and limit from an HTTP request using ParameterPage and
// ParameterLimit. A nil request or URL uses safe defaults.
func Request(req *http.Request) Option {
	return RequestWithNames(req, ParameterNames{})
}

// RequestWithNames sets the page and limit from an HTTP request using custom
// parameter names. Empty names use ParameterPage and ParameterLimit. A nil
// request or URL uses safe defaults.
func RequestWithNames(req *http.Request, names ParameterNames) Option {
	var values url.Values
	if req != nil && req.URL != nil {
		values = req.URL.Query()
	}

	return ValuesWithNames(values, names)
}

// Page sets the page of entities to return. Values below one use the first
// page. Paginate returns ErrOffsetTooLarge if the resulting offset exceeds
// MaximumOffset.
func Page(page int) Option {
	return option(func(p *props) error {
		p.page = max(1, page)
		return nil
	})
}

// Limit sets the maximum number of entities to return. Values below one use
// DefaultLimit, and values above MaximumLimit are clamped to MaximumLimit.
func Limit(limit int) Option {
	return option(func(p *props) error {
		if limit <= 0 {
			limit = DefaultLimit
		}
		p.limit = min(limit, MaximumLimit)
		return nil
	})
}

// UnsafeLimit sets a limit without applying MaximumLimit. It is intended only
// for trusted callers that have independently bounded resource consumption.
// HTTP query parameters never select this option implicitly.
//
// UnsafeLimit returns ErrInvalidLimit from Paginate when limit is non-positive
// or cannot be incremented for the next-page lookahead query.
func UnsafeLimit(limit int) Option {
	return option(func(p *props) error {
		if limit <= 0 || limit == math.MaxInt {
			return fmt.Errorf("%w: unsafe limit must be between 1 and %d", ErrInvalidLimit, math.MaxInt-1)
		}
		p.limit = limit
		return nil
	})
}

// Response contains a page of entities and pagination metadata.
type Response[I any] struct {
	List     []I `json:"list,omitempty"`
	Page     int `json:"page"`
	Limit    int `json:"limit"`
	NextPage int `json:"next_page"`
}

// String summarizes the response and the number and type of its entities.
func (r Response[I]) String() string {
	var i I
	return fmt.Sprintf("[]%T(List: %d, Page: %d, Limit: %d, NextPage: %d)", i, len(r.List), r.Page, r.Limit, r.NextPage)
}

// Paginate applies opts to q and returns one page of entities. It fetches one
// extra entity for bounded queries to determine whether a next page exists.
func Paginate[T any, E Ent[E, T]](ctx context.Context, q Ent[E, T], opts ...Option) (Response[T], error) {
	params := props{
		page:  1,
		limit: DefaultLimit,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt.apply(&params); err != nil {
			return Response[T]{}, err
		}
	}

	offset, err := params.offset()
	if err != nil {
		return Response[T]{}, err
	}

	q = q.Limit(params.limit + 1)

	results, err := q.Offset(offset).All(ctx)
	if err != nil {
		return Response[T]{}, err
	}

	nextPage := 0

	if len(results) > params.limit {
		results = results[:params.limit]
		nextPage = max(1, params.page) + 1
	}

	return Response[T]{
		List:     results,
		NextPage: nextPage,
		Page:     params.page,
		Limit:    params.limit,
	}, nil
}
