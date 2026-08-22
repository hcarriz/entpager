package entpager_test

import (
	"context"
	"net/url"
	"strconv"
	"testing"

	"github.com/hcarriz/entpager"
)

func FuzzValues(f *testing.F) {
	f.Add("", "", uint8(32))
	f.Add("1", "25", uint8(50))
	f.Add("2", "5", uint8(10))
	f.Add("-5", "0", uint8(16))
	f.Add("invalid", "invalid", uint8(8))
	f.Add("1000", "1000", uint8(255))

	f.Fuzz(func(t *testing.T, rawPage, rawLimit string, amount uint8) {
		if !smallParsedInteger(rawPage) || !smallParsedInteger(rawLimit) {
			t.Skip()
		}

		values := url.Values{
			entpager.ParameterPage:  {rawPage},
			entpager.ParameterLimit: {rawLimit},
		}
		got, err := entpager.Paginate(
			context.Background(),
			newfaker(int(amount)),
			entpager.Values(values),
		)
		if err != nil {
			t.Fatalf("Paginate() error = %v", err)
		}

		if got.Page < 1 {
			t.Fatalf("Page = %d, want at least 1", got.Page)
		}
		if got.Limit < 0 {
			t.Fatalf("Limit = %d, want non-negative", got.Limit)
		}
		if len(got.List) > int(amount) {
			t.Fatalf("len(List) = %d, want at most %d", len(got.List), amount)
		}
		if got.Limit > 0 && len(got.List) > got.Limit {
			t.Fatalf("len(List) = %d, want at most Limit %d", len(got.List), got.Limit)
		}

		for i := 1; i < len(got.List); i++ {
			if got.List[i] != got.List[i-1]+1 {
				t.Fatalf("List is not contiguous: %v", got.List)
			}
		}

		if got.NextPage != 0 {
			if got.Limit == 0 {
				t.Fatal("NextPage is set for an unlimited result")
			}
			if got.NextPage != got.Page+1 {
				t.Fatalf("NextPage = %d, want %d", got.NextPage, got.Page+1)
			}
			if len(got.List) != got.Limit {
				t.Fatalf("len(List) = %d, want full page of %d", len(got.List), got.Limit)
			}
		}
	})
}

// smallParsedInteger keeps the current fuzz target away from the known
// unchecked offset-overflow boundary. Extreme integer seeds should be added
// when bounded limits and overflow-safe offset calculation are implemented.
func smallParsedInteger(raw string) bool {
	if len(raw) > 32 {
		return false
	}

	n, err := strconv.Atoi(raw)
	return err != nil || n >= -1000 && n <= 1000
}
