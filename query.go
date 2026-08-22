// Package entpager provides dependency-free offset pagination for Ent-style
// queries.
//
// Paginate accepts any query with All, Offset, and Limit methods matching the
// Ent interface. Pagination values can be constructed directly or derived from
// URL values and HTTP requests. A bounded query fetches one extra entity to
// determine whether Response.NextPage should be set, avoiding a separate count
// query.
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

// Pagination identifies the requested page and maximum number of entities to
// return. Its zero value requests the first page with DefaultLimit entities.
//
// Pagination is intended to be embedded in application-specific parameter
// types that also contain filters or sorting controls.
type Pagination struct {
	Page  int
	Limit int
}

func (p Pagination) normalized() Pagination {
	p.Page = max(1, p.Page)
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	p.Limit = min(p.Limit, MaximumLimit)
	return p
}

func (p Pagination) offset() (int, error) {
	pageOffset := p.Page - 1
	if pageOffset == 0 {
		return 0, nil
	}
	if p.Limit > MaximumOffset/pageOffset {
		return 0, fmt.Errorf(
			"%w: page %d with limit %d exceeds maximum offset %d",
			ErrOffsetTooLarge,
			p.Page,
			p.Limit,
			MaximumOffset,
		)
	}
	return p.Limit * pageOffset, nil
}

// Option configures a call to Paginate.
type Option interface {
	apply(*Pagination) error
}

type option func(*Pagination) error

func (o option) apply(p *Pagination) error {
	return o(p)
}

// PaginationFromValues returns pagination parsed from vals using ParameterPage
// and ParameterLimit.
//
// Missing or malformed values use safe defaults. Page values below one use the
// first page, and limits are clamped to the inclusive range from one to
// MaximumLimit.
func PaginationFromValues(vals url.Values) Pagination {
	return PaginationFromValuesWithNames(vals, ParameterNames{})
}

// PaginationFromValuesWithNames returns pagination parsed from vals using
// custom parameter names. Empty names use ParameterPage and ParameterLimit.
func PaginationFromValuesWithNames(vals url.Values, names ParameterNames) Pagination {
	names = names.defaults()

	raw := vals.Get(names.Limit)
	limit, err := strconv.Atoi(raw)
	if err != nil {
		limit = DefaultLimit
	}

	rawPage := vals.Get(names.Page)
	page, _ := strconv.Atoi(rawPage)
	return Pagination{Page: page, Limit: limit}.normalized()
}

// PaginationFromRequest returns pagination parsed from an HTTP request using
// ParameterPage and ParameterLimit. A nil request or URL uses safe defaults.
func PaginationFromRequest(req *http.Request) Pagination {
	return PaginationFromRequestWithNames(req, ParameterNames{})
}

// PaginationFromRequestWithNames returns pagination parsed from an HTTP request
// using custom parameter names. Empty names use ParameterPage and
// ParameterLimit. A nil request or URL uses safe defaults.
func PaginationFromRequestWithNames(req *http.Request, names ParameterNames) Pagination {
	var values url.Values
	if req != nil && req.URL != nil {
		values = req.URL.Query()
	}

	return PaginationFromValuesWithNames(values, names)
}

// UnsafeLimit sets a limit without applying MaximumLimit. It is intended only
// for trusted callers that have independently bounded resource consumption.
// HTTP query parameters never select this option implicitly.
//
// UnsafeLimit returns ErrInvalidLimit from Paginate when limit is non-positive
// or cannot be incremented for the next-page lookahead query.
func UnsafeLimit(limit int) Option {
	return option(func(p *Pagination) error {
		if limit <= 0 || limit == math.MaxInt {
			return fmt.Errorf("%w: unsafe limit must be between 1 and %d", ErrInvalidLimit, math.MaxInt-1)
		}
		p.Limit = limit
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

// Paginate applies pagination and opts to q and returns one page of entities.
// It fetches one extra entity for bounded queries to determine whether a next
// page exists. The pagination value is normalized before any options are
// applied, so its zero value is safe and useful.
func Paginate[T any, E Ent[E, T]](ctx context.Context, q Ent[E, T], pagination Pagination, opts ...Option) (Response[T], error) {
	params := pagination.normalized()

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

	q = q.Limit(params.Limit + 1)

	results, err := q.Offset(offset).All(ctx)
	if err != nil {
		return Response[T]{}, err
	}

	nextPage := 0

	if len(results) > params.Limit {
		results = results[:params.Limit]
		nextPage = params.Page + 1
	}

	return Response[T]{
		List:     results,
		NextPage: nextPage,
		Page:     params.Page,
		Limit:    params.Limit,
	}, nil
}
