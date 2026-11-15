package orders

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
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

// ---- writes (inside the checkout transaction) --------------------------------

// NewOrder is everything needed to create one store order. All amounts are
// computed by the checkout service from server data; the DB CHECK constraints
// verify total = items_subtotal + shipping_fee and each line_total.
type NewOrder struct {
	CheckoutID     uuid.UUID
	StoreID        uuid.UUID
	StoreName      string
	BuyerID        uuid.UUID
	ShippingMethod string
	ShippingFee    decimal.Decimal
	ItemsSubtotal  decimal.Decimal
	Total          decimal.Decimal
	ShipTo         ShippingAddress
	Items          []NewItem
}

type NewItem struct {
	VariantID   uuid.UUID
	ProductID   uuid.UUID
	ProductName string
	VariantName string
	SKU         string
	UnitPrice   decimal.Decimal
	Quantity    int32
	LineTotal   decimal.Decimal
}

// maxNumberAttempts: with ~1 billion numbers per day a single collision is
// already very unlikely; five in a row means something else is wrong.
const maxNumberAttempts = 5

// Create inserts the order, its item snapshots and its first history row
// (NULL -> pending_payment). It returns the order id and number.
//
// The order number is random, so it can collide. InsertOrder uses
// ON CONFLICT (order_number) DO NOTHING: a collision returns no row instead
// of an error, which matters because any error inside a PostgreSQL
// transaction aborts the whole transaction (every later statement fails).
func (r *Repository) Create(ctx context.Context, o NewOrder, now time.Time) (uuid.UUID, string, error) {
	var (
		id     uuid.UUID
		number string
		err    error
	)
	for range maxNumberAttempts {
		number = NewOrderNumber(now)
		id, err = r.q.InsertOrder(ctx, queries.InsertOrderParams{
			CheckoutID: o.CheckoutID, StoreID: o.StoreID, BuyerID: o.BuyerID, OrderNumber: number,
			ShippingMethod: o.ShippingMethod, ShippingFee: o.ShippingFee,
			ItemsSubtotal: o.ItemsSubtotal, Total: o.Total, StoreName: o.StoreName,
			ShipRecipientName: o.ShipTo.RecipientName, ShipPhone: o.ShipTo.Phone,
			ShipAddressLine: o.ShipTo.AddressLine, ShipCity: o.ShipTo.City,
			ShipProvince: o.ShipTo.Province, ShipPostalCode: o.ShipTo.PostalCode,
		})
		if !errors.Is(err, pgx.ErrNoRows) {
			break
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", fmt.Errorf("insert order: no free order number after %d attempts", maxNumberAttempts)
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("insert order: %w", err)
	}

	for _, it := range o.Items {
		if err := r.q.InsertOrderItem(ctx, queries.InsertOrderItemParams{
			OrderID: id, VariantID: it.VariantID, ProductID: it.ProductID,
			ProductName: it.ProductName, VariantName: it.VariantName, Sku: it.SKU,
			UnitPrice: it.UnitPrice, Quantity: it.Quantity, LineTotal: it.LineTotal,
		}); err != nil {
			return uuid.Nil, "", fmt.Errorf("insert order item: %w", err)
		}
	}

	if err := r.q.InsertOrderStatusHistory(ctx, queries.InsertOrderStatusHistoryParams{
		OrderID: id, ToStatus: StatusPendingPayment, ActorID: &o.BuyerID, Note: "Order created",
	}); err != nil {
		return uuid.Nil, "", fmt.Errorf("insert order history: %w", err)
	}
	return id, number, nil
}

// ---- status changes (inside the caller's transaction) ---------------------------

// LockByCheckout locks every order of a checkout, in id order.
func (r *Repository) LockByCheckout(ctx context.Context, checkoutID uuid.UUID) ([]Locked, error) {
	rows, err := r.q.LockCheckoutOrders(ctx, checkoutID)
	if err != nil {
		return nil, fmt.Errorf("lock checkout orders: %w", err)
	}
	out := make([]Locked, len(rows))
	for i, row := range rows {
		out[i] = Locked{ID: row.ID, CheckoutID: row.CheckoutID, Status: row.Status}
	}
	return out, nil
}

// LockForBuyer locks one order of the buyer (another buyer's -> ErrNotFound).
func (r *Repository) LockForBuyer(ctx context.Context, buyerID, orderID uuid.UUID) (Locked, error) {
	row, err := r.q.LockBuyerOrder(ctx, queries.LockBuyerOrderParams{ID: orderID, BuyerID: buyerID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Locked{}, ErrNotFound
	}
	if err != nil {
		return Locked{}, fmt.Errorf("lock buyer order: %w", err)
	}
	return Locked{ID: row.ID, CheckoutID: row.CheckoutID, Status: row.Status}, nil
}

// Transition changes the status of a LOCKED order if the table allows it,
// and appends the history row. actorID is nil for the system.
func (r *Repository) Transition(ctx context.Context, o *Locked, to string, by Actor, actorID *uuid.UUID, note string) error {
	if !CanTransition(o.Status, to, by) {
		return &TransitionError{From: o.Status, To: to}
	}
	n, err := r.q.SetOrderStatus(ctx, queries.SetOrderStatusParams{ID: o.ID, FromStatus: o.Status, ToStatus: to})
	if err != nil {
		return fmt.Errorf("set order status: %w", err)
	}
	if n == 0 {
		// The row is locked, so its status cannot have changed under us. A
		// caller that forgot to lock ends up here instead of racing.
		return fmt.Errorf("order %s is no longer %s", o.ID, o.Status)
	}
	from := o.Status
	if err := r.q.InsertOrderStatusHistory(ctx, queries.InsertOrderStatusHistoryParams{
		OrderID: o.ID, FromStatus: &from, ToStatus: to, ActorID: actorID, Note: note,
	}); err != nil {
		return fmt.Errorf("insert order history: %w", err)
	}
	o.Status = to
	return nil
}

// StockForCheckout is what a checkout reserved, per variant, in variant order.
func (r *Repository) StockForCheckout(ctx context.Context, checkoutID uuid.UUID) ([]StockLine, error) {
	rows, err := r.q.ListCheckoutStock(ctx, checkoutID)
	if err != nil {
		return nil, fmt.Errorf("list checkout stock: %w", err)
	}
	out := make([]StockLine, len(rows))
	for i, row := range rows {
		out[i] = StockLine{VariantID: row.VariantID, ProductID: row.ProductID, Quantity: row.Quantity}
	}
	return out, nil
}

// StockForOrder is what one order bought, per variant, in variant order.
func (r *Repository) StockForOrder(ctx context.Context, orderID uuid.UUID) ([]StockLine, error) {
	rows, err := r.q.ListOrderStock(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order stock: %w", err)
	}
	out := make([]StockLine, len(rows))
	for i, row := range rows {
		out[i] = StockLine{VariantID: row.VariantID, ProductID: row.ProductID, Quantity: row.Quantity}
	}
	return out, nil
}

// AddSoldCounts adds sign x quantity to products.sold_count (+1 on payment,
// -1 on refund). Products are updated one by one in product_id order: like
// the inventory rows, a fixed order means two transactions that touch the
// same products cannot deadlock.
func (r *Repository) AddSoldCounts(ctx context.Context, lines []StockLine, sign int32) error {
	perProduct := map[uuid.UUID]int32{}
	for _, l := range lines {
		perProduct[l.ProductID] += l.Quantity
	}
	ids := make([]uuid.UUID, 0, len(perProduct))
	for id := range perProduct {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	for _, id := range ids {
		if err := r.q.AddProductSoldCount(ctx, queries.AddProductSoldCountParams{ID: id, Delta: sign * perProduct[id]}); err != nil {
			return fmt.Errorf("update sold count: %w", err)
		}
	}
	return nil
}

// ---- reads ----------------------------------------------------------------------

// ForCheckout returns the orders of one checkout with their items. The caller
// has already checked that the checkout belongs to the user.
func (r *Repository) ForCheckout(ctx context.Context, checkoutID uuid.UUID) ([]Order, error) {
	rows, err := r.q.ListOrdersByCheckout(ctx, checkoutID)
	if err != nil {
		return nil, fmt.Errorf("list orders by checkout: %w", err)
	}
	out := make([]Order, len(rows))
	for i, row := range rows {
		// The three order queries select the same columns, so their generated
		// row types are identical and convert into each other.
		out[i] = toOrder(queries.GetBuyerOrderRow(row))
	}
	return out, r.attachItems(ctx, out)
}

func (r *Repository) CountForBuyer(ctx context.Context, buyerID uuid.UUID) (int64, error) {
	n, err := r.q.CountBuyerOrders(ctx, buyerID)
	if err != nil {
		return 0, fmt.Errorf("count buyer orders: %w", err)
	}
	return n, nil
}

func (r *Repository) ListForBuyer(ctx context.Context, buyerID uuid.UUID, limit, offset int32) ([]Order, error) {
	rows, err := r.q.ListBuyerOrders(ctx, queries.ListBuyerOrdersParams{BuyerID: buyerID, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list buyer orders: %w", err)
	}
	out := make([]Order, len(rows))
	for i, row := range rows {
		out[i] = toOrder(queries.GetBuyerOrderRow(row))
	}
	return out, r.attachItems(ctx, out)
}

// GetForBuyer returns one order with items and history, only if buyerID owns it.
func (r *Repository) GetForBuyer(ctx context.Context, buyerID, orderID uuid.UUID) (Order, error) {
	row, err := r.q.GetBuyerOrder(ctx, queries.GetBuyerOrderParams{ID: orderID, BuyerID: buyerID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get buyer order: %w", err)
	}
	return r.detail(ctx, toOrder(row))
}

// ---- seller (5d): every method takes the caller's store id -------------------

func (r *Repository) CountForStore(ctx context.Context, storeID uuid.UUID, status *string) (int64, error) {
	n, err := r.q.CountStoreOrders(ctx, queries.CountStoreOrdersParams{StoreID: storeID, Status: status})
	if err != nil {
		return 0, fmt.Errorf("count store orders: %w", err)
	}
	return n, nil
}

func (r *Repository) ListForStore(ctx context.Context, storeID uuid.UUID, status *string, limit, offset int32) ([]Order, error) {
	rows, err := r.q.ListStoreOrders(ctx, queries.ListStoreOrdersParams{
		StoreID: storeID, Status: status, PageLimit: limit, PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list store orders: %w", err)
	}
	out := make([]Order, len(rows))
	for i, row := range rows {
		out[i] = toOrder(queries.GetBuyerOrderRow(row))
	}
	return out, r.attachItems(ctx, out)
}

// GetForStore returns one order of the store with items and history
// (another store's order -> ErrNotFound).
func (r *Repository) GetForStore(ctx context.Context, storeID, orderID uuid.UUID) (Order, error) {
	row, err := r.q.GetStoreOrder(ctx, queries.GetStoreOrderParams{ID: orderID, StoreID: storeID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get store order: %w", err)
	}
	return r.detail(ctx, toOrder(queries.GetBuyerOrderRow(row)))
}

// LockForStore locks one order of the store (another store's -> ErrNotFound).
func (r *Repository) LockForStore(ctx context.Context, storeID, orderID uuid.UUID) (Locked, error) {
	row, err := r.q.LockStoreOrder(ctx, queries.LockStoreOrderParams{ID: orderID, StoreID: storeID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Locked{}, ErrNotFound
	}
	if err != nil {
		return Locked{}, fmt.Errorf("lock store order: %w", err)
	}
	return Locked{ID: row.ID, CheckoutID: row.CheckoutID, Status: row.Status}, nil
}

// detail adds the items and the status history to one order.
func (r *Repository) detail(ctx context.Context, o Order) (Order, error) {
	list := []Order{o}
	if err := r.attachItems(ctx, list); err != nil {
		return Order{}, err
	}
	o = list[0]
	hist, err := r.q.ListOrderHistory(ctx, o.ID)
	if err != nil {
		return Order{}, fmt.Errorf("list order history: %w", err)
	}
	o.History = make([]HistoryEntry, len(hist))
	for i, h := range hist {
		o.History[i] = HistoryEntry{FromStatus: h.FromStatus, ToStatus: h.ToStatus, Note: h.Note, CreatedAt: h.CreatedAt}
	}
	return o, nil
}

// attachItems loads the items of all given orders with ONE query (no N+1).
func (r *Repository) attachItems(ctx context.Context, list []Order) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(list))
	index := make(map[uuid.UUID]int, len(list))
	for i, o := range list {
		ids[i] = o.ID
		index[o.ID] = i
		list[i].Items = []Item{}
	}
	rows, err := r.q.ListOrderItems(ctx, ids)
	if err != nil {
		return fmt.Errorf("list order items: %w", err)
	}
	for _, row := range rows {
		it := Item{
			ID: row.ID, VariantID: row.VariantID, ProductID: row.ProductID, ProductSlug: row.ProductSlug,
			ProductName: row.ProductName, VariantName: row.VariantName, SKU: row.Sku,
			UnitPrice: row.UnitPrice, Quantity: row.Quantity, LineTotal: row.LineTotal,
		}
		if row.ImageUrl != "" {
			it.Image = &Image{URL: row.ImageUrl, Alt: row.ImageAlt}
		}
		if row.ReviewID != nil && row.ReviewRating != nil && row.ReviewComment != nil {
			it.Review = &ItemReview{ID: *row.ReviewID, Rating: *row.ReviewRating, Comment: *row.ReviewComment}
		}
		i := index[row.OrderID]
		list[i].Items = append(list[i].Items, it)
	}
	return nil
}

func toOrder(r queries.GetBuyerOrderRow) Order {
	return Order{
		ID: r.ID, CheckoutID: r.CheckoutID, OrderNumber: r.OrderNumber, Status: r.Status,
		StoreID: r.StoreID, StoreSlug: r.StoreSlug, StoreName: r.StoreName,
		ShippingMethod: r.ShippingMethod, ShippingFee: r.ShippingFee,
		ItemsSubtotal: r.ItemsSubtotal, Total: r.Total,
		ShipTo: ShippingAddress{
			RecipientName: r.ShipRecipientName, Phone: r.ShipPhone, AddressLine: r.ShipAddressLine,
			City: r.ShipCity, Province: r.ShipProvince, PostalCode: r.ShipPostalCode,
		},
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}
