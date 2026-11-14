package orders

import "fmt"

// Actor is who asks for a status change. The transition table says not only
// WHICH changes exist but also WHO may make them, so for example a seller
// can never mark an order refunded and a buyer can never mark it shipped.
type Actor string

const (
	BySystem Actor = "system" // payment result, expiry
	ByBuyer  Actor = "buyer"
	BySeller Actor = "seller"
)

type transition struct{ from, to string }

// transitions is THE order lifecycle (PLAN.md §5.5). Every code path that
// changes an order's status goes through Repository.Transition, which checks
// this table; nothing else writes orders.status.
//
//	pending_payment ──payment──▶ paid ──seller──▶ processing ──▶ shipped ──▶ delivered
//	       │                      │
//	       └─▶ cancelled          └─buyer─▶ refunded
//	   (buyer, payment failure, expiry)
//
// delivered, cancelled and refunded are final: nothing leaves them.
var transitions = map[transition][]Actor{
	{StatusPendingPayment, StatusPaid}:      {BySystem},
	{StatusPendingPayment, StatusCancelled}: {BySystem, ByBuyer},
	{StatusPaid, StatusProcessing}:          {BySeller},
	{StatusPaid, StatusRefunded}:            {ByBuyer},
	{StatusProcessing, StatusShipped}:       {BySeller},
	{StatusShipped, StatusDelivered}:        {BySeller},
}

// CanTransition reports whether actor may move an order from one status to another.
func CanTransition(from, to string, by Actor) bool {
	for _, a := range transitions[transition{from, to}] {
		if a == by {
			return true
		}
	}
	return false
}

// TransitionError: the change is not in the table (409 INVALID_STATUS_TRANSITION).
type TransitionError struct {
	From, To string
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("order cannot go from %s to %s", e.From, e.To)
}

// UnpaidError: the buyer tried to cancel one unpaid order. One payment covers
// the whole checkout (decisions Q4/Q8), so an unpaid order is cancelled
// together with its checkout (POST /checkout/:id/cancel).
type UnpaidError struct {
	CheckoutID string
}

func (e *UnpaidError) Error() string { return "an unpaid order is cancelled with its checkout" }
