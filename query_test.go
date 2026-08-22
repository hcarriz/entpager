package entpager_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/hcarriz/entpager"
)

// This behaves like ent.
type fake struct {
	list   []int
	limit  int
	offset int
}

func (f *fake) All(ctx context.Context) ([]int, error) {
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

func newfaker(n int) *fake {
	f := &fake{}

	for i := 0; i < n; i++ {
		f.list = append(f.list, i)
	}

	return f
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

func TestSearch(t *testing.T) {
	type args struct {
		amount int
		opts   []entpager.Option
	}
	tests := []struct {
		name    string
		args    args
		want    entpager.Response[int]
		wantErr bool
	}{
		{
			name: "all",
			args: args{
				amount: 5,
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "all, nil opts",
			args: args{
				amount: 5,
				opts:   []entpager.Option{nil},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "limited",
			args: args{
				amount: 10,
				opts: []entpager.Option{
					entpager.Limit(5),
				},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    5,
				NextPage: 2,
			},
			wantErr: false,
		},
		{
			name: "limited to zero",
			args: args{
				amount: 5,
				opts: []entpager.Option{
					entpager.Limit(0),
				},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    0,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "limited, page 2",
			args: args{
				amount: 10,
				opts: []entpager.Option{
					entpager.Limit(5),
					entpager.Page(2),
				},
			},
			want: entpager.Response[int]{
				List:     []int{5, 6, 7, 8, 9},
				Page:     2,
				Limit:    5,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "requested",
			args: args{
				amount: 10,
				opts: []entpager.Option{
					entpager.Request(&http.Request{
						URL: &url.URL{
							RawQuery: url.Values{"page": []string{"2"}, "limit": []string{"5"}}.Encode(),
						},
					}),
				},
			},
			want: entpager.Response[int]{
				List:     []int{5, 6, 7, 8, 9},
				Page:     2,
				Limit:    5,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "nil request",
			args: args{
				amount: 5,
				opts:   []entpager.Option{entpager.Request(nil)},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "request with nil URL",
			args: args{
				amount: 5,
				opts:   []entpager.Option{entpager.Request(&http.Request{})},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "combined nil option",
			args: args{
				amount: 5,
				opts:   []entpager.Option{entpager.Options(nil)},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "requested, but bad values",
			args: args{
				amount: 10,
				opts: []entpager.Option{
					entpager.Request(&http.Request{
						URL: &url.URL{
							RawQuery: url.Values{"page": []string{"a"}, "limit": []string{"c"}}.Encode(),
						},
					}),
				},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
				Page:     1,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
		{
			name: "requested, but bad page and good limit",
			args: args{
				amount: 10,
				opts: []entpager.Option{
					entpager.Request(&http.Request{
						URL: &url.URL{
							RawQuery: url.Values{"page": []string{"a"}, "limit": []string{"5"}}.Encode(),
						},
					}),
				},
			},
			want: entpager.Response[int]{
				List:     []int{0, 1, 2, 3, 4},
				Page:     1,
				Limit:    5,
				NextPage: 2,
			},
			wantErr: false,
		},
		{
			name: "requested, but bad limit and good page",
			args: args{
				amount: 50,
				opts: []entpager.Option{
					entpager.Request(&http.Request{
						URL: &url.URL{
							RawQuery: url.Values{"page": []string{"2"}, "limit": []string{"a"}}.Encode(),
						},
					}),
				},
			},
			want: entpager.Response[int]{
				List: []int{
					25, 26, 27, 28, 29, 30, 31, 32, 33,
					34, 35, 36, 37, 38, 39, 40, 41, 42, 43,
					44, 45, 46, 47, 48, 49,
				},
				Page:     2,
				Limit:    entpager.DefaultLimit,
				NextPage: 0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := entpager.Paginate(context.Background(), newfaker(tt.args.amount), tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Search() = %s, want %s", got, tt.want)
			}

			t.Log(got.String())
		})
	}
}

func TestPaginateReturnsQueryError(t *testing.T) {
	t.Parallel()

	want := errors.New("query failed")
	got, err := entpager.Paginate(context.Background(), &errorQuery{err: want})

	if !errors.Is(err, want) {
		t.Fatalf("Paginate() error = %v, want %v", err, want)
	}
	if !reflect.DeepEqual(got, entpager.Response[int]{}) {
		t.Fatalf("Paginate() response = %v, want zero response", got)
	}
}
