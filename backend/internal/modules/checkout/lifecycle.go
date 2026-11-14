package checkout

// The life of a placed checkout (slice 5c):
//
//	pending_payment ──payment succeeded──▶ paid       (orders paid, stock consumed, sold_count +)
//	       ├────────payment failed───────▶ cancelled  (decision Q9)
//	       ├────────buyer cancels────────▶ cancelled
//	       └────────time runs out────────▶ expired    (orders cancelled)
//	   the last three release the reserved stock
//
// THE RULE: every change starts by locking the checkout row (FOR UPDATE).
// "Pay" and "expire" (or "pay" and "cancel") for the same checkout therefore
// run one after the other, never interleaved. The second one sees that the
// status is no longer pending_payment and stops, so exactly one of them wins.
//
// Lock order inside every path: checkout -> its orders (id order) ->
// inventory rows (variant_id order) -> products (id order) -> payment row.
// Placing a checkout locks cart -> inventory, and a refund locks order ->
// inventory -> products: no path takes two of these in the opposite order,
// so they cannot deadlock each other.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/inventory"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/orders"
)

const (
	StatusPaid      = "paid"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"

	PaymentSucceeded = "succeeded"
	PaymentFailed    = "failed"
	PaymentCancelled = "cancelled"
	PaymentExpired   = "expired"

	EventSucceeded = "payment.succeeded"
	EventFailed    = "payment.failed"
)

var (
	ErrNotPending      = errors.New("checkout is not waiting for payment")
	ErrExpired         = errors.New("checkout expired")
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrEventConflict: the event id was already used for another payment or
	// another outcome. A real gateway never does that; a client bug might.
	ErrEventConflict = errors.New("payment event id already used differently")
)

// Outcome of a simulated payment.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
)

func (o Outcome) eventType() string {
	if o == OutcomeSuccess {
		return EventSucceeded
	}
	return EventFailed
}

// closing describes one way a pending checkout ends without payment.
type closing struct {
	checkoutStatus string
	paymentStatus  string
	by             orders.Actor
	note           string // order history note and ledger reason
}

var (
	closeByBuyer   = closing{StatusCancelled, PaymentCancelled, orders.ByBuyer, "Cancelled by the buyer"}
	closeByFailure = closing{StatusCancelled, PaymentFailed, orders.BySystem, "Payment failed"}
	closeByExpiry  = closing{StatusExpired, PaymentExpired, orders.BySystem, "Payment time ran out"}
)

// closePending ends a pending checkout: every order -> cancelled (with a
// history row), every reserved unit -> available again (with a release
// ledger row), checkout and payment get their final status. The caller has
// locked the checkout row and checked that it is pending.
func closePending(ctx context.Context, tx pgx.Tx, checkoutID uuid.UUID, how closing, actorID *uuid.UUID) error {
	orderRepo := orders.NewRepository(tx)
	list, err := orderRepo.LockByCheckout(ctx, checkoutID)
	if err != nil {
		return err
	}
	for i := range list {
		if err := orderRepo.Transition(ctx, &list[i], orders.StatusCancelled, how.by, actorID, how.note); err != nil {
			return err
		}
	}

	lines, err := orderRepo.StockForCheckout(ctx, checkoutID)
	if err != nil {
		return err
	}
	if _, err := inventory.LockMany(ctx, tx, orders.VariantIDs(lines)); err != nil {
		return err
	}
	ref := inventory.Reference{Type: ReferenceCheckout, ID: checkoutID}
	for _, l := range lines {
		if err := inventory.Release(ctx, tx, l.VariantID, l.Quantity, ref, actorID, how.note); err != nil {
			return err
		}
	}

	return NewRepository(tx).Close(ctx, checkoutID, how.checkoutStatus, how.paymentStatus)
}

// markPaid applies a successful payment: orders -> paid, reserved stock
// consumed (it is sold now), sold_count raised, checkout and payment final.
// The caller has locked the checkout row and checked that it is pending and
// not expired.
func markPaid(ctx context.Context, tx pgx.Tx, checkoutID uuid.UUID) error {
	orderRepo := orders.NewRepository(tx)
	list, err := orderRepo.LockByCheckout(ctx, checkoutID)
	if err != nil {
		return err
	}
	for i := range list {
		if err := orderRepo.Transition(ctx, &list[i], orders.StatusPaid, orders.BySystem, nil, "Payment received"); err != nil {
			return err
		}
	}

	lines, err := orderRepo.StockForCheckout(ctx, checkoutID)
	if err != nil {
		return err
	}
	if _, err := inventory.LockMany(ctx, tx, orders.VariantIDs(lines)); err != nil {
		return err
	}
	ref := inventory.Reference{Type: ReferenceCheckout, ID: checkoutID}
	for _, l := range lines {
		if err := inventory.Consume(ctx, tx, l.VariantID, l.Quantity, ref); err != nil {
			return err
		}
	}
	if err := orderRepo.AddSoldCounts(ctx, lines, 1); err != nil {
		return err
	}

	return NewRepository(tx).MarkPaid(ctx, checkoutID)
}

// Cancel cancels a pending checkout of the user and releases its stock.
//
// If it has already run out of time (the sweeper has not reached it yet), it
// is closed as EXPIRED instead and ErrExpired is returned. That close must
// be committed, so it cannot be an error inside the transaction (an error
// rolls back); the transaction ends normally and the error is returned after.
func (s *Service) Cancel(ctx context.Context, userID, id uuid.UUID) error {
	expired := false
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		c, err := NewRepository(tx).LockForUser(ctx, userID, id)
		if err != nil {
			return err
		}
		if c.Status != StatusPendingPayment {
			return ErrNotPending
		}
		if c.Expired {
			expired = true
			return closePending(ctx, tx, c.ID, closeByExpiry, nil)
		}
		return closePending(ctx, tx, c.ID, closeByBuyer, &userID)
	})
	if err != nil {
		return err
	}
	if expired {
		return ErrExpired
	}
	return nil
}

// SimulatePayment plays the payment gateway: it delivers a success or
// failure notification for the user's payment. eventID identifies the
// notification; gateways retry deliveries, so the same eventID must have an
// effect only once.
//
// Returns the checkout id, and applied=false when this event was already
// processed (the caller just shows the current state).
//
// One transaction:
//  1. Lock the payment's checkout (filtered by owner -> 404).
//  2. Event seen before? Same payment and outcome -> no-op; otherwise 409.
//  3. Not pending -> ErrNotPending (already paid, cancelled or expired).
//  4. Past expires_at -> close it as expired, commit, return ErrExpired.
//  5. Store the event (UNIQUE event_id), then apply the outcome.
func (s *Service) SimulatePayment(ctx context.Context, userID, paymentID uuid.UUID, outcome Outcome, eventID uuid.UUID) (checkoutID uuid.UUID, applied bool, err error) {
	expired := false
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)

		// 1.
		c, err := repo.LockForPayment(ctx, userID, paymentID)
		if err != nil {
			return err
		}
		checkoutID = c.ID

		// 2. Safe to check-then-insert: the same payment's events are
		// serialised by the checkout lock taken above.
		prev, seen, err := repo.PaymentEvent(ctx, eventID)
		if err != nil {
			return err
		}
		if seen {
			if prev.PaymentID == paymentID && prev.EventType == outcome.eventType() {
				return nil // a repeated delivery: nothing to do
			}
			return ErrEventConflict
		}

		// 3.
		if c.Status != StatusPendingPayment {
			return ErrNotPending
		}
		// 4.
		if c.Expired {
			expired = true
			return closePending(ctx, tx, c.ID, closeByExpiry, nil)
		}

		// 5. ON CONFLICT DO NOTHING: an event id that another payment stored
		// concurrently comes back as "not inserted", not as an error.
		payload, _ := json.Marshal(map[string]string{"outcome": string(outcome)})
		inserted, err := repo.InsertPaymentEvent(ctx, paymentID, eventID, outcome.eventType(), payload)
		if err != nil {
			return err
		}
		if !inserted {
			return ErrEventConflict
		}
		applied = true
		if outcome == OutcomeSuccess {
			return markPaid(ctx, tx, c.ID)
		}
		return closePending(ctx, tx, c.ID, closeByFailure, nil)
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	if expired {
		return checkoutID, false, ErrExpired
	}
	return checkoutID, applied, nil
}

// ExpireDue closes up to batch checkouts whose time ran out, in one
// transaction, and returns how many it closed.
func (s *Service) ExpireDue(ctx context.Context, batch int) (int, error) {
	n := 0
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := NewRepository(tx).LockDue(ctx, batch)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := closePending(ctx, tx, id, closeByExpiry, nil); err != nil {
				return fmt.Errorf("expire checkout %s: %w", id, err)
			}
		}
		n = len(ids)
		return nil
	})
	return n, err
}

// SweepBatch is how many checkouts one sweeper transaction closes at most,
// so one run never holds many locks for long.
const SweepBatch = 50

// RunSweeper expires due checkouts once at start (catching up after a
// restart) and then every interval, until ctx is cancelled. It runs inside
// the API process: fine for one instance; with several, SKIP LOCKED already
// keeps two sweepers from closing the same checkout.
func (s *Service) RunSweeper(ctx context.Context, interval time.Duration) {
	s.sweep(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.sweep(ctx)
		}
	}
}

func (s *Service) sweep(ctx context.Context) {
	for {
		n, err := s.ExpireDue(ctx, SweepBatch)
		if err != nil {
			if ctx.Err() == nil { // a cancelled ctx is just the shutdown
				slog.ErrorContext(ctx, "expire checkouts", "error", err)
			}
			return
		}
		if n > 0 {
			slog.InfoContext(ctx, "expired checkouts", "count", n)
		}
		if n < SweepBatch {
			return
		}
	}
}
