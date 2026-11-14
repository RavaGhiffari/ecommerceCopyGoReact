package orders

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var allStatuses = []string{StatusPendingPayment, StatusPaid, StatusProcessing, StatusShipped, StatusDelivered, StatusCancelled, StatusRefunded}

// Every (from, to, actor) combination is checked, so a change to the table
// that is not mirrored here fails the test.
func TestCanTransition_FullTable(t *testing.T) {
	allowed := map[string]bool{
		"pending_payment>paid/system":      true,
		"pending_payment>cancelled/system": true,
		"pending_payment>cancelled/buyer":  true,
		"paid>processing/seller":           true,
		"paid>refunded/buyer":              true,
		"processing>shipped/seller":        true,
		"shipped>delivered/seller":         true,
	}

	n := 0
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			for _, by := range []Actor{BySystem, ByBuyer, BySeller} {
				key := from + ">" + to + "/" + string(by)
				assert.Equal(t, allowed[key], CanTransition(from, to, by), key)
				n++
			}
		}
	}
	assert.Equal(t, 7*7*3, n)
}

func TestCanTransition_FinalStatusesAreFinal(t *testing.T) {
	for _, final := range []string{StatusDelivered, StatusCancelled, StatusRefunded} {
		for _, to := range allStatuses {
			for _, by := range []Actor{BySystem, ByBuyer, BySeller} {
				assert.False(t, CanTransition(final, to, by), "%s -> %s by %s", final, to, by)
			}
		}
	}
}
