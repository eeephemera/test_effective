package store

import (
	"testing"
	"time"
)

func TestMonthsInclusive(t *testing.T) {
	cases := []struct {
		name string
		a, b time.Time
		want int
	}{
		{"same month", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC), 1},
		{"two months partial", time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC), time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC), 2},
		{"three months", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC), 3},
		{"cross year", time.Date(2024, 11, 3, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC), 4},
		{"end before start", time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := monthsInclusive(c.a, c.b)
			if got != c.want {
				t.Fatalf("monthsInclusive(%v,%v) = %d; want %d", c.a, c.b, got, c.want)
			}
		})
	}
}
