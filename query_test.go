package entpager_test

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/hcarriz/entpager"
)

// fake behaves like an Ent query and records the applied limit and offset.
type fake struct {
	list   []int
	limit  int
	offset int
}

func (f *fake) All(context.Context) ([]int, error) {
	if f.offset >= len(f.list) {
		return []int{}, nil
	}

	end := len(f.list)
	if f.limit > 0 && f.offset+f.limit < len(f.list) {
		end = f.offset + f.limit
	}

	return f.list[f.offset:end], nil
}

func (f *fake) Limit(n int) *fake {
	f.limit = n
	return f
}

func (f *fake) Offset(n int) *fake {
	f.offset = n
	return f
}

func newFake(n int) *fake {
	return &fake{list: sequence(0, n)}
}

func sequence(start, n int) []int {
	result := make([]int, n)
	for i := range result {
		result[i] = start + i
	}
	return result
}

var _ entpager.Ent[*fake, int] = (*fake)(nil)

type errorQuery struct {
	err error
}

func (q *errorQuery) All(context.Context) ([]int, error) {
	return nil, q.err
}

func (q *errorQuery) Limit(int) *errorQuery {
	return q
}

func (q *errorQuery) Offset(int) *errorQuery {
	return q
}

var _ entpager.Ent[*errorQuery, int] = (*errorQuery)(nil)

func TestPaginate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		amount     int
		pagination entpager.Pagination
		opts       []entpager.Option
		want       entpager.Response[int]
	}{
		{
			name:   "defaults",
			amount: 5,
			want: entpager.Response[int]{
				List:  sequence(0, 5),
				Page:  1,
				Limit: entpager.DefaultLimit,
			},
		},
		{
			name:   "nil option",
			amount: 5,
			opts:   []entpager.Option{nil},
			want: entpager.Response[int]{
				List:  sequence(0, 5),
				Page:  1,
				Limit: entpager.DefaultLimit,
			},
		},
		{
			name:   "bounded limit",
			amount: 10,
			pagination: entpager.Pagination{
				Limit: 5,
			},
			want: entpager.Response[int]{
				List:     sequence(0, 5),
				Page:     1,
				Limit:    5,
				NextPage: 2,
			},
		},
		{
			name:   "zero limit uses default",
			amount: 30,
			pagination: entpager.Pagination{
				Limit: 0,
			},
			want: entpager.Response[int]{
				List:     sequence(0, entpager.DefaultLimit),
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 2,
			},
		},
		{
			name:   "negative limit uses default",
			amount: 30,
			pagination: entpager.Pagination{
				Limit: -1,
			},
			want: entpager.Response[int]{
				List:     sequence(0, entpager.DefaultLimit),
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 2,
			},
		},
		{
			name:   "large limit is clamped",
			amount: entpager.MaximumLimit + 1,
			pagination: entpager.Pagination{
				Limit: 1000,
			},
			want: entpager.Response[int]{
				List:     sequence(0, entpager.MaximumLimit),
				Page:     1,
				Limit:    entpager.MaximumLimit,
				NextPage: 2,
			},
		},
		{
			name:   "second page",
			amount: 10,
			pagination: entpager.Pagination{
				Page:  2,
				Limit: 5,
			},
			want: entpager.Response[int]{
				List:  sequence(5, 5),
				Page:  2,
				Limit: 5,
			},
		},
		{
			name:   "negative page uses first page",
			amount: 10,
			pagination: entpager.Pagination{
				Page:  -100,
				Limit: 5,
			},
			want: entpager.Response[int]{
				List:     sequence(0, 5),
				Page:     1,
				Limit:    5,
				NextPage: 2,
			},
		},
		{
			name:   "HTTP values",
			amount: 10,
			pagination: entpager.PaginationFromRequest(&http.Request{URL: &url.URL{
				RawQuery: url.Values{"page": {"2"}, "limit": {"5"}}.Encode(),
			}}),
			want: entpager.Response[int]{
				List:  sequence(5, 5),
				Page:  2,
				Limit: 5,
			},
		},
		{
			name:   "malformed HTTP values use defaults",
			amount: 30,
			pagination: entpager.PaginationFromValues(url.Values{
				"page":  {"invalid"},
				"limit": {"invalid"},
			}),
			want: entpager.Response[int]{
				List:     sequence(0, entpager.DefaultLimit),
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 2,
			},
		},
		{
			name:   "unsafe HTTP limits are clamped",
			amount: entpager.MaximumLimit + 1,
			pagination: entpager.PaginationFromValues(url.Values{
				"limit": {"1000000"},
			}),
			want: entpager.Response[int]{
				List:     sequence(0, entpager.MaximumLimit),
				Page:     1,
				Limit:    entpager.MaximumLimit,
				NextPage: 2,
			},
		},
		{
			name:       "nil request uses defaults",
			amount:     5,
			pagination: entpager.PaginationFromRequest(nil),
			want: entpager.Response[int]{
				List:  sequence(0, 5),
				Page:  1,
				Limit: entpager.DefaultLimit,
			},
		},
		{
			name:       "request with nil URL uses defaults",
			amount:     5,
			pagination: entpager.PaginationFromRequest(&http.Request{}),
			want: entpager.Response[int]{
				List:  sequence(0, 5),
				Page:  1,
				Limit: entpager.DefaultLimit,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := entpager.Paginate(context.Background(), newFake(tt.amount), tt.pagination, tt.opts...)
			if err != nil {
				t.Fatalf("Paginate() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Paginate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestCustomParameterNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		names entpager.ParameterNames
		want  entpager.Response[int]
	}{
		{
			name:  "custom page and limit",
			names: entpager.ParameterNames{Page: "p", Limit: "size"},
			want:  entpager.Response[int]{List: sequence(5, 5), Page: 2, Limit: 5},
		},
		{
			name:  "empty page name uses default",
			names: entpager.ParameterNames{Limit: "size"},
			want:  entpager.Response[int]{List: sequence(5, 5), Page: 2, Limit: 5},
		},
		{
			name:  "empty limit name uses default",
			names: entpager.ParameterNames{Page: "p"},
			want:  entpager.Response[int]{List: sequence(5, 5), Page: 2, Limit: 5},
		},
	}

	values := url.Values{
		"page":  {"2"},
		"p":     {"2"},
		"limit": {"5"},
		"size":  {"5"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := &http.Request{URL: &url.URL{RawQuery: values.Encode()}}
			pagination := entpager.PaginationFromRequestWithNames(req, tt.names)
			got, err := entpager.Paginate(
				context.Background(),
				newFake(10),
				pagination,
			)
			if err != nil {
				t.Fatalf("Paginate() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Paginate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestUnsafeLimit(t *testing.T) {
	t.Parallel()

	query := newFake(entpager.MaximumLimit + 2)
	got, err := entpager.Paginate(
		context.Background(),
		query,
		entpager.Pagination{},
		entpager.UnsafeLimit(entpager.MaximumLimit+1),
	)
	if err != nil {
		t.Fatalf("Paginate() error = %v", err)
	}

	want := entpager.Response[int]{
		List:     sequence(0, entpager.MaximumLimit+1),
		Page:     1,
		Limit:    entpager.MaximumLimit + 1,
		NextPage: 2,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Paginate() = %s, want %s", got, want)
	}
	if query.limit != entpager.MaximumLimit+2 {
		t.Fatalf("query limit = %d, want lookahead limit %d", query.limit, entpager.MaximumLimit+2)
	}
}

func TestUnsafeLimitRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
	}{
		{name: "zero", limit: 0},
		{name: "negative", limit: -1},
		{name: "maximum int", limit: math.MaxInt},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := entpager.Paginate(
				context.Background(),
				newFake(1),
				entpager.Pagination{},
				entpager.UnsafeLimit(tt.limit),
			)
			if !errors.Is(err, entpager.ErrInvalidLimit) {
				t.Fatalf("Paginate() error = %v, want ErrInvalidLimit", err)
			}
			if !reflect.DeepEqual(got, entpager.Response[int]{}) {
				t.Fatalf("Paginate() response = %v, want zero response", got)
			}
		})
	}
}

func TestMaximumOffset(t *testing.T) {
	t.Parallel()

	t.Run("boundary is allowed", func(t *testing.T) {
		t.Parallel()

		query := newFake(0)
		page := entpager.MaximumOffset/entpager.MaximumLimit + 1
		got, err := entpager.Paginate(
			context.Background(),
			query,
			entpager.Pagination{Page: page, Limit: entpager.MaximumLimit},
		)
		if err != nil {
			t.Fatalf("Paginate() error = %v", err)
		}
		if got.Page != page {
			t.Fatalf("Page = %d, want %d", got.Page, page)
		}
		if query.offset != entpager.MaximumOffset {
			t.Fatalf("query offset = %d, want %d", query.offset, entpager.MaximumOffset)
		}
	})

	tests := []struct {
		name       string
		pagination entpager.Pagination
		opts       []entpager.Option
	}{
		{
			name: "above boundary",
			pagination: entpager.Pagination{
				Page:  entpager.MaximumOffset/entpager.MaximumLimit + 2,
				Limit: entpager.MaximumLimit,
			},
		},
		{
			name:       "maximum int page",
			pagination: entpager.Pagination{Page: math.MaxInt},
		},
		{
			name:       "unsafe limit still observes maximum offset",
			pagination: entpager.Pagination{Page: 2},
			opts: []entpager.Option{
				entpager.UnsafeLimit(entpager.MaximumOffset + 1),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			query := newFake(0)
			got, err := entpager.Paginate(context.Background(), query, tt.pagination, tt.opts...)
			if !errors.Is(err, entpager.ErrOffsetTooLarge) {
				t.Fatalf("Paginate() error = %v, want ErrOffsetTooLarge", err)
			}
			if !reflect.DeepEqual(got, entpager.Response[int]{}) {
				t.Fatalf("Paginate() response = %v, want zero response", got)
			}
			if query.limit != 0 || query.offset != 0 {
				t.Fatalf("query was modified before offset validation: %+v", query)
			}
		})
	}
}

func TestPaginateReturnsQueryError(t *testing.T) {
	t.Parallel()

	want := errors.New("query failed")
	got, err := entpager.Paginate(context.Background(), &errorQuery{err: want}, entpager.Pagination{})

	if !errors.Is(err, want) {
		t.Fatalf("Paginate() error = %v, want %v", err, want)
	}
	if !reflect.DeepEqual(got, entpager.Response[int]{}) {
		t.Fatalf("Paginate() response = %v, want zero response", got)
	}
}
