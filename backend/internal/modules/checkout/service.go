package checkout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/inventory"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/orders"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/shipping"
)

type Config struct {
	TTL   time.Duration // how long an unpaid checkout holds its stock (Q5)
	Rates shipping.Rates
}

type Service struct {
	pool *pgxpool.Pool
	cfg  Config
}

func NewService(pool *pgxpool.Pool, cfg Config) *Service {
	return &Service{pool: pool, cfg: cfg}
}

// Quote prices the current cart for an address and shipping choice, without
// writing anything. Lines with an issue are returned flagged (HasIssues), not
// as an error, so the page can show what to fix.
func (s *Service) Quote(ctx context.Context, userID uuid.UUID, in Input) (Quote, error) {
	addr, err := s.address(ctx, s.pool, userID, in.AddressID)
	if err != nil {
		return Quote{}, err
	}
	rows, err := NewRepository(s.pool).Lines(ctx, userID)
	if err != nil {
		return Quote{}, err
	}
	if len(rows) == 0 {
		return Quote{}, ErrCartEmpty
	}
	return buildQuote(rows, userID, addr, in.Shipping, s.cfg.Rates, false)
}

// Place creates the checkout. created=false means the Idempotency-Key was
// seen before with the same request, and the first checkout is returned.
//
// One transaction, step by step (PLAN.md §5.4):
//
//  1. Lock the cart row. All checkouts of this user, and their cart edits,
//     now run one after another. This is also what makes the idempotency
//     check below safe: a second submit with the same key waits here and
//     then finds the first checkout.
//  2. Idempotency: same key + same request -> return the first checkout;
//     same key + different request -> ErrKeyReused (409).
//  3. The address, filtered by user_id (someone else's -> 404).
//  4. Lock every inventory row of the cart in variant_id order (no deadlock).
//  5. Read the lines AFTER the locks, so the stock we check cannot change
//     before we commit, and price them from the database.
//  6. Any issue -> IssuesError (409). Returning an error rolls back, so
//     nothing is written and the cart is untouched.
//  7. Insert checkout, orders (+ items, history), payment.
//  8. Reserve the stock (+ ledger rows).
//  9. Remove the purchased lines from the cart.
//
// Any error in any step rolls back all of it.
func (s *Service) Place(ctx context.Context, userID, key uuid.UUID, in Input) (id uuid.UUID, created bool, err error) {
	hash := in.Hash()

	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)

		// 1.
		cartID, hasCart, err := repo.LockCart(ctx, userID)
		if err != nil {
			return err
		}

		// 2.
		prevID, seen, sameRequest, err := repo.PreviousCheckout(ctx, userID, key, hash)
		if err != nil {
			return err
		}
		if seen {
			if !sameRequest {
				return ErrKeyReused
			}
			id = prevID
			return nil // nothing written; the transaction just ends
		}
		if !hasCart {
			return ErrCartEmpty
		}

		// 3.
		addr, err := s.address(ctx, tx, userID, in.AddressID)
		if err != nil {
			return err
		}

		// 4.
		variantIDs, err := repo.CartVariantIDs(ctx, userID)
		if err != nil {
			return err
		}
		if len(variantIDs) == 0 {
			return ErrCartEmpty
		}
		stock, err := inventory.LockMany(ctx, tx, variantIDs)
		if err != nil {
			return err
		}

		// 5. The locked values are authoritative; a variant without a stock
		// row counts as out of stock.
		rows, err := repo.Lines(ctx, userID)
		if err != nil {
			return err
		}
		for i := range rows {
			rows[i].AvailableQuantity = stock[rows[i].VariantID].Available
		}
		quote, err := buildQuote(rows, userID, addr, in.Shipping, s.cfg.Rates, true)
		if err != nil {
			return err
		}

		// 6.
		if quote.HasIssues {
			return &IssuesError{Lines: quote.IssueLines()}
		}

		// 7.
		checkoutID, expiresAt, err := repo.Create(ctx, newCheckout{
			UserID: userID, IdempotencyKey: key, RequestHash: hash,
			ItemsTotal: quote.ItemsTotal, ShippingTotal: quote.ShippingTotal, GrandTotal: quote.GrandTotal,
			TTL: s.cfg.TTL,
		})
		if err != nil {
			return err
		}
		orderRepo := orders.NewRepository(tx)
		now := time.Now()
		for _, g := range quote.Groups {
			if _, _, err := orderRepo.Create(ctx, newOrder(checkoutID, userID, addr, g), now); err != nil {
				return err
			}
		}
		if err := repo.CreatePayment(ctx, checkoutID, quote.GrandTotal, expiresAt); err != nil {
			return err
		}

		// 8. The rows are locked and were checked above, so a shortage here
		// would be a bug; it still rolls everything back.
		ref := inventory.Reference{Type: ReferenceCheckout, ID: checkoutID}
		for _, g := range quote.Groups {
			for _, l := range g.Lines {
				if err := inventory.Reserve(ctx, tx, l.VariantID, l.Quantity, ref, &userID); err != nil {
					return err
				}
			}
		}

		// 9.
		if err := repo.RemoveCartLines(ctx, cartID, variantIDs); err != nil {
			return err
		}

		id, created = checkoutID, true
		return nil
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, created, nil
}

// Get returns a checkout of the user with its payment and orders.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Checkout, []orders.Order, error) {
	c, err := NewRepository(s.pool).Get(ctx, userID, id)
	if err != nil {
		return Checkout{}, nil, err
	}
	list, err := orders.NewRepository(s.pool).ForCheckout(ctx, c.ID)
	if err != nil {
		return Checkout{}, nil, err
	}
	return c, list, nil
}

func (s *Service) address(ctx context.Context, db queries.DBTX, userID, id uuid.UUID) (addresses.Address, error) {
	a, err := addresses.NewRepository(db).Get(ctx, userID, id)
	if errors.Is(err, addresses.ErrNotFound) {
		return addresses.Address{}, ErrAddressNotFound
	}
	return a, err
}

func newOrder(checkoutID, buyerID uuid.UUID, addr addresses.Address, g Group) orders.NewOrder {
	items := make([]orders.NewItem, len(g.Lines))
	for i, l := range g.Lines {
		items[i] = orders.NewItem{
			VariantID: l.VariantID, ProductID: l.ProductID, ProductName: l.ProductName,
			VariantName: l.VariantName, SKU: l.SKU, UnitPrice: l.UnitPrice, Quantity: l.Quantity,
			LineTotal: l.LineTotal,
		}
	}
	return orders.NewOrder{
		CheckoutID: checkoutID, StoreID: g.Store.ID, StoreName: g.Store.Name, BuyerID: buyerID,
		ShippingMethod: string(g.Method), ShippingFee: g.ShippingFee,
		ItemsSubtotal: g.Subtotal, Total: g.Total,
		ShipTo: orders.ShippingAddress{
			RecipientName: addr.RecipientName, Phone: addr.Phone, AddressLine: addr.AddressLine,
			City: addr.City, Province: addr.Province, PostalCode: addr.PostalCode,
		},
		Items: items,
	}
}
