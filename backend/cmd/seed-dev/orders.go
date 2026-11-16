package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/addresses"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/cart"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/checkout"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/orders"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/reviews"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/shipping"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
)

// Demo orders: one buyer, a handful of checkouts in every order status, a few
// reviews, spread over the last weeks. They make the seller dashboard (6c),
// the order pages and later the admin pages show real numbers.
//
// Like the catalog, every order goes through the real services (cart →
// checkout → simulated payment → seller transitions → reviews), so stock,
// ledger, sold_count and rating counters stay consistent. The only direct SQL
// is moving created_at into the past, which no API can do.

const (
	BuyerEmail    = "pembeli.demo@contoh.id"
	BuyerPassword = "pembeli-123"
)

type demoLine struct {
	sku string // from the catalog seed: <store code>-<product>-<variant>
	qty int32
}

type demoOrder struct {
	daysAgo int
	lines   []demoLine
	fail    bool              // the simulated payment fails → every order cancelled
	status  map[string]string // store code → final status (default: paid)
	reviews map[string]int    // sku → rating, for delivered items
}

var demoOrders = []demoOrder{
	{daysAgo: 45, lines: []demoLine{{"GN-01-02", 2}, {"BK-01-02", 1}},
		status: map[string]string{"GN": orders.StatusDelivered, "BK": orders.StatusDelivered}},
	{daysAgo: 25, lines: []demoLine{{"GN-01-01", 1}, {"BK-02-02", 3}},
		status:  map[string]string{"GN": orders.StatusDelivered, "BK": orders.StatusDelivered},
		reviews: map[string]int{"GN-01-01": 5, "BK-02-02": 4}},
	{daysAgo: 18, lines: []demoLine{{"GN-05-01", 2}, {"RA-06-01", 2}},
		status:  map[string]string{"GN": orders.StatusDelivered, "RA": orders.StatusDelivered},
		reviews: map[string]int{"GN-05-01": 4, "RA-06-01": 5}},
	{daysAgo: 10, lines: []demoLine{{"GN-06-01", 3}},
		status: map[string]string{"GN": orders.StatusShipped}},
	{daysAgo: 6, lines: []demoLine{{"GN-02-01", 1}, {"SP-02-01", 1}},
		status: map[string]string{"GN": orders.StatusProcessing}},
	{daysAgo: 3, lines: []demoLine{{"GN-03-01", 1}},
		status: map[string]string{"GN": orders.StatusRefunded}},
	{daysAgo: 2, lines: []demoLine{{"GN-04-01", 1}, {"RA-03-01", 1}}},
	{daysAgo: 1, lines: []demoLine{{"GN-06-02", 2}}, fail: true},
}

var reviewComments = map[int]string{
	5: "Barang sesuai deskripsi, pengiriman cepat. Mantap!",
	4: "Kualitas bagus, packing rapi. Sedikit lama di pengiriman.",
}

// seedOrders runs once: it does nothing if the demo buyer already exists,
// and needs the demo catalog (penjual1..4) to be there.
func seedOrders(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, authSvc *auth.Service) error {
	var buyerExists, catalogExists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)", BuyerEmail).Scan(&buyerExists); err != nil {
		return err
	}
	if buyerExists {
		fmt.Println("demo buyer already exists; demo orders were seeded before, nothing to do")
		return nil
	}
	if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM product_variants WHERE sku = 'GN-01-01')").Scan(&catalogExists); err != nil {
		return err
	}
	if !catalogExists {
		fmt.Println("demo catalog not found (this database was not seeded by seed-dev); skipping demo orders")
		return nil
	}

	cartSvc := cart.NewService(pool)
	checkoutSvc := checkout.NewService(pool, checkout.Config{
		TTL:   cfg.CheckoutTTL,
		Rates: shipping.Rates{Regular: cfg.Shipping.Regular, Express: cfg.Shipping.Express, SameDay: cfg.Shipping.SameDay},
	})
	orderSvc := orders.NewService(pool)
	reviewSvc := reviews.NewService(pool)

	buyer, err := authSvc.CreateUserWithRole(ctx, auth.RegisterInput{Name: "Rina Pembeli", Email: BuyerEmail, Password: BuyerPassword}, authctx.RoleCustomer)
	if err != nil {
		return fmt.Errorf("demo buyer: %w", err)
	}
	addr, err := addresses.NewService(pool).Create(ctx, buyer.ID, addresses.Fields{
		Label: "Rumah", RecipientName: "Rina Pembeli", Phone: "081234567890",
		AddressLine: "Jl. Merdeka No. 10", City: "Bandung", Province: "Jawa Barat", PostalCode: "40111",
	}, true)
	if err != nil {
		return fmt.Errorf("demo address: %w", err)
	}

	for i, d := range demoOrders {
		if err := placeDemoOrder(ctx, pool, d, buyer.ID, addr.ID, cartSvc, checkoutSvc, orderSvc, reviewSvc); err != nil {
			return fmt.Errorf("demo order %d: %w", i+1, err)
		}
	}
	fmt.Printf("seeded %d demo checkouts; buyer login: %s / %s (local demo only)\n", len(demoOrders), BuyerEmail, BuyerPassword)
	return nil
}

type variantRef struct {
	variantID, storeID, ownerID uuid.UUID
	storeCode                   string
}

func lookupVariant(ctx context.Context, pool *pgxpool.Pool, sku string) (variantRef, error) {
	r := variantRef{storeCode: sku[:2]}
	err := pool.QueryRow(ctx, `
		SELECT v.id, s.id, s.owner_id
		FROM product_variants v
		JOIN products p ON p.id = v.product_id
		JOIN stores s   ON s.id = p.store_id
		WHERE v.sku = $1`, sku).Scan(&r.variantID, &r.storeID, &r.ownerID)
	if err != nil {
		return r, fmt.Errorf("variant %s: %w", sku, err)
	}
	return r, nil
}

func placeDemoOrder(
	ctx context.Context, pool *pgxpool.Pool, d demoOrder, buyerID, addressID uuid.UUID,
	cartSvc *cart.Service, checkoutSvc *checkout.Service, orderSvc *orders.Service, reviewSvc *reviews.Service,
) error {
	// 1. Fill the cart, remembering which store each line belongs to.
	stores := map[uuid.UUID]variantRef{} // store id → a ref of that store
	skuOfVariant := map[uuid.UUID]string{}
	for _, l := range d.lines {
		ref, err := lookupVariant(ctx, pool, l.sku)
		if err != nil {
			return err
		}
		qty := l.qty
		if _, err := cartSvc.AddItem(ctx, buyerID, ref.variantID, &qty); err != nil {
			return fmt.Errorf("add %s: %w", l.sku, err)
		}
		stores[ref.storeID] = ref
		skuOfVariant[ref.variantID] = l.sku
	}

	// 2. Check out (regular shipping everywhere) and pay.
	ship := map[uuid.UUID]shipping.Method{}
	for id := range stores {
		ship[id] = shipping.Regular
	}
	checkoutID, _, err := checkoutSvc.Place(ctx, buyerID, uuid.New(), checkout.Input{AddressID: addressID, Shipping: ship})
	if err != nil {
		return fmt.Errorf("place: %w", err)
	}
	c, placed, err := checkoutSvc.Get(ctx, buyerID, checkoutID)
	if err != nil {
		return err
	}
	outcome := checkout.OutcomeSuccess
	if d.fail {
		outcome = checkout.OutcomeFailure
	}
	if _, _, err := checkoutSvc.SimulatePayment(ctx, buyerID, c.Payment.ID, outcome, uuid.New()); err != nil {
		return fmt.Errorf("pay: %w", err)
	}

	// 3. Move each store order to its final status, the way people would.
	if !d.fail {
		for _, o := range placed {
			ref := stores[o.StoreID]
			target := d.status[ref.storeCode]
			if err := advance(ctx, orderSvc, buyerID, ref.ownerID, o.ID, target); err != nil {
				return fmt.Errorf("order %s → %s: %w", o.ID, target, err)
			}
			if target != orders.StatusDelivered {
				continue
			}
			for _, it := range o.Items {
				rating, ok := d.reviews[skuOfVariant[it.VariantID]]
				if !ok {
					continue
				}
				in := reviews.Input{Rating: rating, Comment: reviewComments[rating]}
				if _, err := reviewSvc.Create(ctx, buyerID, it.ID, in); err != nil {
					return fmt.Errorf("review: %w", err)
				}
			}
		}
	}

	// 4. Move the whole story into the past (no API can do this).
	if d.daysAgo > 0 {
		for _, q := range []string{
			"UPDATE checkouts SET created_at = created_at - make_interval(days => $2) WHERE id = $1",
			"UPDATE orders SET created_at = created_at - make_interval(days => $2) WHERE checkout_id = $1",
			`UPDATE order_status_histories h SET created_at = h.created_at - make_interval(days => $2)
			 FROM orders o WHERE o.id = h.order_id AND o.checkout_id = $1`,
			// updated_at moves too: "edited" means updated_at > created_at, and a
			// seeded review was never edited.
			`UPDATE reviews r SET created_at = r.created_at - make_interval(days => $2),
			                      updated_at = r.updated_at - make_interval(days => $2)
			 FROM order_items oi JOIN orders o ON o.id = oi.order_id
			 WHERE oi.id = r.order_item_id AND o.checkout_id = $1`,
		} {
			if _, err := pool.Exec(ctx, q, checkoutID, d.daysAgo); err != nil {
				return fmt.Errorf("backdate: %w", err)
			}
		}
	}
	return nil
}

// advance walks a paid order to target through the real transitions:
// seller steps for processing/shipped/delivered, the buyer's cancel for a refund.
func advance(ctx context.Context, svc *orders.Service, buyerID, sellerID, orderID uuid.UUID, target string) error {
	switch target {
	case "", orders.StatusPaid:
		return nil
	case orders.StatusRefunded:
		_, err := svc.CancelForBuyer(ctx, buyerID, orderID)
		return err
	}
	for _, step := range []string{orders.StatusProcessing, orders.StatusShipped, orders.StatusDelivered} {
		if _, err := svc.UpdateStatusForStore(ctx, sellerID, orderID, step, ""); err != nil {
			return err
		}
		if step == target {
			return nil
		}
	}
	return fmt.Errorf("unknown target status %q", target)
}
