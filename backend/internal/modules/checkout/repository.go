package checkout

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

type Repository struct {
	q *queries.Queries
}

// NewRepository works with the pool or a transaction (both are DBTX).
func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

// LockCart locks the user's cart row for the rest of the transaction.
// found=false when the user never had a cart.
func (r *Repository) LockCart(ctx context.Context, userID uuid.UUID) (id uuid.UUID, found bool, err error) {
	id, err = r.q.LockCartByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("lock cart: %w", err)
	}
	return id, true, nil
}

// PreviousCheckout looks up an earlier checkout with the same key.
// Returns its id, whether it exists, and whether its request hash matches.
func (r *Repository) PreviousCheckout(ctx context.Context, userID, key uuid.UUID, hash []byte) (uuid.UUID, bool, bool, error) {
	row, err := r.q.GetCheckoutByIdempotencyKey(ctx, queries.GetCheckoutByIdempotencyKeyParams{UserID: userID, IdempotencyKey: key})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, false, nil
	}
	if err != nil {
		return uuid.Nil, false, false, fmt.Errorf("get checkout by idempotency key: %w", err)
	}
	return row.ID, true, bytes.Equal(row.RequestHash, hash), nil
}

func (r *Repository) CartVariantIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.q.ListCartVariantIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list cart variant ids: %w", err)
	}
	return ids, nil
}

func (r *Repository) Lines(ctx context.Context, userID uuid.UUID) ([]lineRow, error) {
	rows, err := r.q.ListCheckoutLines(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list checkout lines: %w", err)
	}
	out := make([]lineRow, len(rows))
	for i, row := range rows {
		out[i] = lineRow{
			ItemID: row.ID, Quantity: row.Quantity,
			VariantID: row.VariantID, VariantName: row.VariantName, SKU: row.Sku, Price: row.Price,
			ProductID: row.ProductID, ProductSlug: row.ProductSlug, ProductName: row.ProductName,
			ProductStatus: row.ProductStatus,
			Store:         StoreRef{ID: row.StoreID, Slug: row.StoreSlug, Name: row.StoreName, Location: row.StoreLocation},
			StoreStatus:   row.StoreStatus, StoreOwnerID: row.StoreOwnerID,
			AvailableQuantity: row.AvailableQuantity,
		}
		if row.ImageUrl != "" {
			out[i].Image = &Image{URL: row.ImageUrl, Alt: row.ImageAlt}
		}
	}
	return out, nil
}

type newCheckout struct {
	UserID         uuid.UUID
	IdempotencyKey uuid.UUID
	RequestHash    []byte
	ItemsTotal     decimal.Decimal
	ShippingTotal  decimal.Decimal
	GrandTotal     decimal.Decimal
	TTL            time.Duration
}

// Create inserts the checkout; expires_at = database NOW() + TTL.
func (r *Repository) Create(ctx context.Context, c newCheckout) (uuid.UUID, time.Time, error) {
	row, err := r.q.CreateCheckout(ctx, queries.CreateCheckoutParams{
		UserID: c.UserID, IdempotencyKey: c.IdempotencyKey, RequestHash: c.RequestHash,
		ItemsTotal: c.ItemsTotal, ShippingTotal: c.ShippingTotal, GrandTotal: c.GrandTotal,
		TtlSeconds: c.TTL.Seconds(),
	})
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("create checkout: %w", err)
	}
	return row.ID, row.ExpiresAt, nil
}

// CreatePayment adds the pending simulated payment of a checkout.
func (r *Repository) CreatePayment(ctx context.Context, checkoutID uuid.UUID, amount decimal.Decimal, expiresAt time.Time) error {
	if err := r.q.CreatePayment(ctx, queries.CreatePaymentParams{
		CheckoutID: checkoutID, Reference: newPaymentReference(), Amount: amount, ExpiresAt: expiresAt,
	}); err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *Repository) RemoveCartLines(ctx context.Context, cartID uuid.UUID, variantIDs []uuid.UUID) error {
	if _, err := r.q.DeleteCartItemsByVariants(ctx, queries.DeleteCartItemsByVariantsParams{CartID: cartID, VariantIds: variantIDs}); err != nil {
		return fmt.Errorf("remove purchased cart lines: %w", err)
	}
	return nil
}

// Get returns the checkout with its payment, only if userID owns it.
func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (Checkout, error) {
	row, err := r.q.GetCheckout(ctx, queries.GetCheckoutParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkout{}, ErrNotFound
	}
	if err != nil {
		return Checkout{}, fmt.Errorf("get checkout: %w", err)
	}
	return Checkout{
		ID: row.ID, Status: row.Status, ItemsTotal: row.ItemsTotal, ShippingTotal: row.ShippingTotal,
		GrandTotal: row.GrandTotal, ExpiresAt: row.ExpiresAt, PaidAt: row.PaidAt, CancelledAt: row.CancelledAt,
		CreatedAt: row.CreatedAt,
		Payment: Payment{
			ID: row.PaymentID, Reference: row.PaymentReference, Provider: row.PaymentProvider,
			Amount: row.PaymentAmount, Status: row.PaymentStatus, ExpiresAt: row.PaymentExpiresAt,
			PaidAt: row.PaymentPaidAt,
		},
	}, nil
}

// newPaymentReference returns e.g. "SIM-4F7K2M9QX0AB3CDE": 16 random
// Crockford characters (80 bits), so a collision is not a practical concern;
// the UNIQUE constraint would still reject one.
func newPaymentReference() string {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	var b [16]byte
	_, _ = rand.Read(b[:])
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = alphabet[v%32]
	}
	return "SIM-" + string(out)
}
