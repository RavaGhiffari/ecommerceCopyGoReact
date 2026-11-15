package rating

import "testing"

func TestAverage(t *testing.T) {
	cases := []struct {
		count, total int32
		want         float64
	}{
		{0, 0, 0},
		{1, 5, 5},
		{3, 14, 4.7}, // 4.666…
		{4, 17, 4.3}, // 4.25 rounds half away from zero
		{2, 3, 1.5},
	}
	for _, c := range cases {
		if got := Average(c.count, c.total); got != c.want {
			t.Errorf("Average(%d, %d) = %v, want %v", c.count, c.total, got, c.want)
		}
	}
}
