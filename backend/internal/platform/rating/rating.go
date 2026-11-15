// Package rating turns the denormalised review counters (count, sum of
// stars) into the value shown to people. Products, stores and reviews all use
// it, so an average is rounded the same way everywhere.
package rating

import "math"

// Average returns total/count rounded to one decimal (4.66 -> 4.7), or 0
// when there are no reviews. It is a display value, never used for money.
func Average(count, total int32) float64 {
	if count <= 0 {
		return 0
	}
	return math.Round(float64(total)/float64(count)*10) / 10
}
