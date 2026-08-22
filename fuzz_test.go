package entpager_test

import (
	"context"
	"errors"
	"math"
	"net/url"
	"strconv"
	"testing"

	"github.com/hcarriz/entpager"
)

func FuzzPaginationFromValues(f *testing.F) {
	f.Add("", "", uint8(32))
	f.Add("1", "25", uint8(50))
	f.Add("2", "5", uint8(10))
	f.Add("-5", "0", uint8(16))
	f.Add("invalid", "invalid", uint8(8))
	f.Add("1000", "1000", uint8(255))
	f.Add(strconv.Itoa(math.MaxInt), strconv.Itoa(math.MaxInt), uint8(1))
	f.Add(strconv.Itoa(-math.MaxInt-1), strconv.Itoa(-math.MaxInt-1), uint8(1))

	f.Fuzz(func(t *testing.T, rawPage, rawLimit string, amount uint8) {
		values := url.Values{
			entpager.ParameterPage:  {rawPage},
			entpager.ParameterLimit: {rawLimit},
		}
		pagination := entpager.PaginationFromValues(values)
		if pagination.Page < 1 {
			t.Fatalf("parsed Page = %d, want at least 1", pagination.Page)
		}
		if pagination.Limit < 1 || pagination.Limit > entpager.MaximumLimit {
			t.Fatalf("parsed Limit = %d, want between 1 and %d", pagination.Limit, entpager.MaximumLimit)
		}

		got, err := entpager.Paginate(
			context.Background(),
			newFake(int(amount)),
			pagination,
		)
		if err != nil {
			if !errors.Is(err, entpager.ErrOffsetTooLarge) {
				t.Fatalf("Paginate() error = %v", err)
			}
			return
		}

		if got.Page < 1 {
			t.Fatalf("Page = %d, want at least 1", got.Page)
		}
		if got.Limit < 1 || got.Limit > entpager.MaximumLimit {
			t.Fatalf("Limit = %d, want between 1 and %d", got.Limit, entpager.MaximumLimit)
		}
		if len(got.List) > int(amount) {
			t.Fatalf("len(List) = %d, want at most %d", len(got.List), amount)
		}
		if len(got.List) > got.Limit {
			t.Fatalf("len(List) = %d, want at most Limit %d", len(got.List), got.Limit)
		}

		for i := 1; i < len(got.List); i++ {
			if got.List[i] != got.List[i-1]+1 {
				t.Fatalf("List is not contiguous: %v", got.List)
			}
		}

		if got.NextPage != 0 {
			if got.NextPage != got.Page+1 {
				t.Fatalf("NextPage = %d, want %d", got.NextPage, got.Page+1)
			}
			if len(got.List) != got.Limit {
				t.Fatalf("len(List) = %d, want full page of %d", len(got.List), got.Limit)
			}
		}
	})
}
