// Package inventory owns stock quantities and their ledger.
//
// The one rule this package exists to guarantee: stock never changes without
// a matching row in inventory_movements, written in the same transaction.
// That is why the functions here take a queries.DBTX and are meant to be
// called with a pgx.Tx: the caller's transaction covers the quantity AND the
// ledger row, so one can never be saved without the other.
//
// Phase 2 only has manual adjustments by the seller. Reservations (checkout),
// releases (cancel/expiry) and consumption (payment) arrive in later phases
// and will use the same Lock -> update -> record pattern.
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
	TypeAdjustment = "adjustment"
	// MaxQuantity keeps typos like 1000000000 out; also well inside INTEGER.
	MaxQuantity = 1_000_000
)

var ErrNotFound = errors.New("inventory row not found")

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
