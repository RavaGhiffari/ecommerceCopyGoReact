package orders

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/stores"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// MaxNoteLength matches order_status_histories.note VARCHAR(500).
const MaxNoteLength = 500

// defaultNotes is written to the history when the seller gives no note.
var defaultNotes = map[string]string{
	StatusProcessing: "The seller is preparing the order",
	StatusShipped:    "Order shipped",
	StatusDelivered:  "Order delivered",
}

// storeOf resolves the caller's store. Every seller operation starts here:
// the store always comes from the session, never from the request, and every
// query below filters by it. That is the whole ownership rule for sellers.
func (s *Service) storeOf(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, error) {
	st, err := stores.NewRepository(s.pool).GetByOwner(ctx, ownerID)
	if errors.Is(err, stores.ErrNotFound) {
		return uuid.Nil, ErrNoStore
	}
	if err != nil {
		return uuid.Nil, err
	}
	return st.ID, nil
}

// ListForStore returns one page of the seller's orders, newest first,
// optionally only one status ("" = all).
func (s *Service) ListForStore(ctx context.Context, ownerID uuid.UUID, status string, page, pageSize int) (Page, error) {
	errs := validation.Errors{}
	page, pageSize = paging(errs, page, pageSize)
	var filter *string
	if status != "" {
		if !slices.Contains(Statuses, status) {
			errs.Add("status", "must be one of "+strings.Join(Statuses, ", "))
		}
		filter = &status
	}
	if err := errs.Err(); err != nil {
		return Page{}, err
	}

	storeID, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return Page{}, err
	}
	repo := NewRepository(s.pool)
	total, err := repo.CountForStore(ctx, storeID, filter)
	if err != nil {
		return Page{}, err
	}
	items, err := repo.ListForStore(ctx, storeID, filter, int32(pageSize), int32((page-1)*pageSize))
	if err != nil {
		return Page{}, err
	}
	return newPage(items, page, pageSize, total), nil
}

// GetForStore returns ErrNotFound for another store's order.
func (s *Service) GetForStore(ctx context.Context, ownerID, orderID uuid.UUID) (Order, error) {
	storeID, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return Order{}, err
	}
	return NewRepository(s.pool).GetForStore(ctx, storeID, orderID)
}

// UpdateStatusForStore moves one order of the seller's store forward:
// paid -> processing -> shipped -> delivered (PLAN.md §5.5).
//
//  1. Validate the request: a known status (422 otherwise), note <= 500.
//  2. Lock the order row, filtered by store (another store's -> 404). The
//     buyer's refund locks the same row, so "seller starts processing" and
//     "buyer cancels" cannot both win.
//  3. Transition with BySeller: the table decides (pending_payment ->
//     processing, paid -> shipped, refunded, ... -> 409).
//  4. History row with the seller as actor and the note.
func (s *Service) UpdateStatusForStore(ctx context.Context, ownerID, orderID uuid.UUID, to, note string) (Order, error) {
	// 1.
	errs := validation.Errors{}
	if to == "" {
		errs.Add("status", "is required")
	} else if !slices.Contains(Statuses, to) {
		errs.Add("status", "must be one of "+strings.Join(Statuses, ", "))
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > MaxNoteLength {
		errs.Add("note", "must be at most 500 characters")
	}
	if err := errs.Err(); err != nil {
		return Order{}, err
	}
	if note == "" {
		note = defaultNotes[to]
	}

	storeID, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return Order{}, err
	}
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		// 2.
		o, err := repo.LockForStore(ctx, storeID, orderID)
		if err != nil {
			return err
		}
		// 3. + 4.
		return repo.Transition(ctx, &o, to, BySeller, &ownerID, note)
	})
	if err != nil {
		return Order{}, err
	}
	return NewRepository(s.pool).GetForStore(ctx, storeID, orderID)
}
