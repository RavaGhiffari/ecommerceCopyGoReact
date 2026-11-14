// Package inventory owns stock quantities and their ledger.
//
// The one rule this package exists to guarantee: stock never changes without
// a matching row in inventory_movements, written in the same transaction.
// That is why the functions here take a queries.DBTX and are meant to be
// called with a pgx.Tx: the caller's transaction covers the quantity AND the
// ledger row, so one can never be saved without the other.
//
// Movements (PLAN.md §5.2):
//
//	adjustment   seller sets the stock           available ±n
//	reservation  checkout placed                 available -q, reserved +q
//	release      cancel, payment failed, expiry  available +q, reserved -q
//	consumption  payment succeeded               reserved -q
//	restock      paid order refunded             available +q
//
// Callers that move several variants lock them first with LockMany, so every
// transaction takes inventory locks in the same (variant_id) order.
package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

const (
	TypeAdjustment  = "adjustment"
	TypeReservation = "reservation"
	TypeRelease     = "release"
	TypeConsumption = "consumption"
	TypeRestock     = "restock"
	// MaxQuantity keeps typos like 1000000000 out; also well inside INTEGER.
	MaxQuantity = 1_000_000
)

var (
	ErrNotFound          = errors.New("inventory row not found")
	ErrInsufficientStock = errors.New("not enough available stock")
	// ErrNotReserved: a release or consumption found fewer reserved units than
	// expected. That is always a bug (the ledger and the orders disagree), so
	// the caller's transaction must roll back.
	ErrNotReserved = errors.New("reserved stock is lower than expected")
)

// Create adds the stock row for a new variant and sets its initial quantity
// through SetAvailable, so even the first stock gets a ledger entry.
func Create(ctx context.Context, db queries.DBTX, variantID uuid.UUID, initial int32, actor *uuid.UUID) error {
	q := queries.New(db)
	if err := q.CreateInventory(ctx, variantID); err != nil {
		return fmt.Errorf("create inventory: %w", err)
	}
	if initial == 0 {
		return nil
	}
	_, err := SetAvailable(ctx, db, variantID, initial, actor, "initial stock")
	return err
}

// SetAvailable sets a variant's available quantity to an absolute value and
// records the difference. Returns the new quantity.
//
// Step by step, inside the caller's transaction:
//  1. SELECT ... FOR UPDATE locks the stock row. A concurrent change to the
//     same variant now waits for our commit instead of racing us.
//  2. delta = new - current (nothing to record when it is 0).
//  3. UPDATE the quantity. The CHECK (available_quantity >= 0) constraint is
//     the final guard against negative stock.
//  4. INSERT the ledger row with the delta, the reason and who did it.
func SetAvailable(ctx context.Context, db queries.DBTX, variantID uuid.UUID, quantity int32, actor *uuid.UUID, reason string) (int32, error) {
	q := queries.New(db)

	current, err := q.LockInventory(ctx, variantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock inventory: %w", err)
	}

	delta := quantity - current.AvailableQuantity
	if delta == 0 {
		return quantity, nil
	}

	if err := q.SetAvailableQuantity(ctx, queries.SetAvailableQuantityParams{VariantID: variantID, AvailableQuantity: quantity}); err != nil {
		return 0, fmt.Errorf("set available quantity: %w", err)
	}
	if err := q.InsertInventoryMovement(ctx, queries.InsertInventoryMovementParams{
		VariantID:      variantID,
		MovementType:   TypeAdjustment,
		AvailableDelta: delta,
		Reason:         reason,
		ActorID:        actor,
	}); err != nil {
		return 0, fmt.Errorf("record inventory movement: %w", err)
	}
	return quantity, nil
}

// Stock is one locked inventory row.
type Stock struct {
	Available int32
	Reserved  int32
}

// LockMany locks the stock rows of several variants in one statement and in
// variant_id order (the deadlock guard, see inventory.sql). Variants without
// a stock row are simply absent from the map. Must run inside a transaction:
// the locks last until it commits or rolls back.
func LockMany(ctx context.Context, tx queries.DBTX, variantIDs []uuid.UUID) (map[uuid.UUID]Stock, error) {
	rows, err := queries.New(tx).LockInventoryRows(ctx, variantIDs)
	if err != nil {
		return nil, fmt.Errorf("lock inventory rows: %w", err)
	}
	out := make(map[uuid.UUID]Stock, len(rows))
	for _, r := range rows {
		out[r.VariantID] = Stock{Available: r.AvailableQuantity, Reserved: r.ReservedQuantity}
	}
	return out, nil
}

// Reference says what caused a movement, e.g. {"checkout", <id>}.
type Reference struct {
	Type string
	ID   uuid.UUID
}

// Reserve moves quantity units of a variant from available to reserved for an
// unpaid checkout, and records it in the ledger (available -q, reserved +q).
//
// The UPDATE is conditional (WHERE available_quantity >= q), so even without
// a prior lock it can never oversell: 0 rows changed means ErrInsufficientStock
// and nothing was written. Callers in a checkout still lock first (LockMany),
// so they can report every short line at once instead of failing on the first.
func Reserve(ctx context.Context, tx queries.DBTX, variantID uuid.UUID, quantity int32, ref Reference, actor *uuid.UUID) error {
	return move(ctx, tx, change{
		kind: TypeReservation, variant: variantID, quantity: quantity, ref: ref, actor: actor, reason: "checkout",
		available: -quantity, reserved: quantity, failure: ErrInsufficientStock,
		update: func(q *queries.Queries) (int64, error) {
			return q.ReserveInventory(ctx, queries.ReserveInventoryParams{VariantID: variantID, Quantity: quantity})
		},
	})
}

// Release gives reserved units back to available: the checkout was cancelled,
// its payment failed or it expired (available +q, reserved -q).
func Release(ctx context.Context, tx queries.DBTX, variantID uuid.UUID, quantity int32, ref Reference, actor *uuid.UUID, reason string) error {
	return move(ctx, tx, change{
		kind: TypeRelease, variant: variantID, quantity: quantity, ref: ref, actor: actor, reason: reason,
		available: quantity, reserved: -quantity, failure: ErrNotReserved,
		update: func(q *queries.Queries) (int64, error) {
			return q.ReleaseInventory(ctx, queries.ReleaseInventoryParams{VariantID: variantID, Quantity: quantity})
		},
	})
}

// Consume removes reserved units for good: they are paid for (reserved -q).
// Available does not change; it already went down at the reservation.
func Consume(ctx context.Context, tx queries.DBTX, variantID uuid.UUID, quantity int32, ref Reference) error {
	return move(ctx, tx, change{
		kind: TypeConsumption, variant: variantID, quantity: quantity, ref: ref, reason: "payment received",
		reserved: -quantity, failure: ErrNotReserved,
		update: func(q *queries.Queries) (int64, error) {
			return q.ConsumeInventory(ctx, queries.ConsumeInventoryParams{VariantID: variantID, Quantity: quantity})
		},
	})
}

// Restock puts sold units back on sale after a refund (available +q).
func Restock(ctx context.Context, tx queries.DBTX, variantID uuid.UUID, quantity int32, ref Reference, actor *uuid.UUID, reason string) error {
	return move(ctx, tx, change{
		kind: TypeRestock, variant: variantID, quantity: quantity, ref: ref, actor: actor, reason: reason,
		available: quantity, failure: ErrNotFound,
		update: func(q *queries.Queries) (int64, error) {
			return q.RestockInventory(ctx, queries.RestockInventoryParams{VariantID: variantID, Quantity: quantity})
		},
	})
}

// change is one checkout-related stock movement.
type change struct {
	kind      string
	variant   uuid.UUID
	quantity  int32
	ref       Reference
	actor     *uuid.UUID
	reason    string
	available int32 // ledger deltas; must match what update does
	reserved  int32
	failure   error // returned when update changes no row
	update    func(*queries.Queries) (int64, error)
}

// move runs the conditional UPDATE and, only if it changed the row, writes
// the matching ledger row. Both happen in the caller's transaction, so the
// ledger invariant (SUM of deltas = quantities) holds after every commit.
func move(ctx context.Context, tx queries.DBTX, c change) error {
	if c.quantity <= 0 {
		return fmt.Errorf("%s: quantity must be positive, got %d", c.kind, c.quantity)
	}
	q := queries.New(tx)
	n, err := c.update(q)
	if err != nil {
		return fmt.Errorf("%s inventory: %w", c.kind, err)
	}
	if n == 0 {
		return fmt.Errorf("%s of variant %s: %w", c.kind, c.variant, c.failure)
	}
	refType := c.ref.Type
	if err := q.InsertInventoryMovement(ctx, queries.InsertInventoryMovementParams{
		VariantID:      c.variant,
		MovementType:   c.kind,
		AvailableDelta: c.available,
		ReservedDelta:  c.reserved,
		Reason:         c.reason,
		ReferenceType:  &refType,
		ReferenceID:    &c.ref.ID,
		ActorID:        c.actor,
	}); err != nil {
		return fmt.Errorf("record %s: %w", c.kind, err)
	}
	return nil
}
