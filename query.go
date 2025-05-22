package entpager

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Interface for ent.
type Ent[Self, Entity any] interface {
	All(context.Context) ([]Entity, error)
	Offset(int) Self
	Limit(int) Self
}

var (
	DefaultLimit   = 25
	ParameterPage  = "page"
	ParameterLimit = "limit"
)

type props struct {
	page  int
	limit int
}

func (p props) offset() int {
	return max(0, p.limit*(max(1, p.page)-1))
}

type Option interface {
	apply(*props) error
}

type option func(*props) error

func (o option) apply(p *props) error {
	return o(p)
}

// Combine multiple Option to create a single Option.
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

// Set the limit and page using url.Values
//
// If limit is malformed, it uses the default limit.
// If page is malformed, it returns the first page.
func Values(vals url.Values) Option {
	raw := vals.Get(ParameterLimit)
	limit, err := strconv.Atoi(raw)
	if err != nil {
		limit = DefaultLimit
	}

	rawPage := vals.Get(ParameterPage)
	page, _ := strconv.Atoi(rawPage)
	return Options(Limit(limit), Page(page))
}

// Set the limit and page using a *http.Request.
func Request(req *http.Request) Option {

	result := make(url.Values)

	if req != nil {
		if req.URL != nil {
			result = req.URL.Query()
		}
	}

	return Values(result)

}

// The page of entities to return.
func Page(page int) Option {
	return option(func(p *props) error {
		p.page = max(1, page)
		return nil
	})
}

// The limit on the number of entities to return.
func Limit(limit int) Option {
	return option(func(p *props) error {
		p.limit = max(0, limit)
		return nil
	})
}

// The results of the paginate command.
type Response[I any] struct {
	List     []I `json:"list,omitempty"`
	Page     int `json:"page"`
	Limit    int `json:"limit"`
	NextPage int `json:"next_page"`
}

func (r Response[I]) String() string {
	var i I
	return fmt.Sprintf("[]%T(List: %d, Page: %d, Limit: %d, NextPage: %d)", i, len(r.List), r.Page, r.Limit, r.NextPage)

}

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

	if params.limit > 0 {
		q = q.Limit(params.limit + 1)
	}

	results, err := q.Offset(params.offset()).All(ctx)
	if err != nil {
		return Response[T]{}, err
	}

	nextPage := 0

	if params.limit > 0 && len(results) > params.limit {
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
