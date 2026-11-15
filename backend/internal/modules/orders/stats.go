package orders

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/rating"
)

// Seller dashboard (Phase 6c, PLAN.md Q12): GET /seller/stats.
//
// Everything is computed on read from orders, order_items and the product
// rating counters. There is no stats table to keep in sync: at this size a
// few aggregate queries on indexed columns are cheap, and the numbers can
// never drift from the orders themselves.

// RevenueStatuses are the statuses whose money the store keeps. Pending
// orders are not paid yet; cancelled and refunded orders gave the money back.
var RevenueStatuses = []string{StatusPaid, StatusProcessing, StatusShipped, StatusDelivered}

const (
	// StatsWindow is the "recent" period of the dashboard.
	StatsWindow = 30 * 24 * time.Hour
	// TopProductsLimit is the length of the best-seller table.
	TopProductsLimit = 5
)

// SellerStats is the dashboard of one store.
type SellerStats struct {
	Since          time.Time        // start of the window (now - StatsWindow)
	OrdersByStatus map[string]int64 // every status present, 0 when none
	RevenueAll     decimal.Decimal  // sum of items_subtotal, RevenueStatuses only
	RevenueWindow  decimal.Decimal
	UnitsAll       int64
	UnitsWindow    int64
	TopProducts    []TopProduct // of the window
	RatingAverage  float64
	RatingCount    int32
}

type TopProduct struct {
	ProductID uuid.UUID
	Name      string
	Status    string // the product's current status (archived products still count)
	Units     int64
	Revenue   decimal.Decimal
}

// StatsForStore builds the caller's dashboard. now is a parameter so the
// window is explicit; the handler passes time.Now().
//
// The four reads are not in one transaction: an order paid between two of
// them could make "revenue" and "units" disagree for one request. That is
// acceptable for a dashboard that is refreshed by reloading the page; a
// REPEATABLE READ transaction would fix it if it ever matters.
func (s *Service) StatsForStore(ctx context.Context, ownerID uuid.UUID, now time.Time) (SellerStats, error) {
	storeID, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return SellerStats{}, err
	}
	return NewRepository(s.pool).StoreStats(ctx, storeID, now.Add(-StatsWindow))
}

// StoreStats runs the aggregate queries of one store (all filter by storeID).
func (r *Repository) StoreStats(ctx context.Context, storeID uuid.UUID, since time.Time) (SellerStats, error) {
	st := SellerStats{Since: since, OrdersByStatus: make(map[string]int64, len(Statuses))}
	for _, s := range Statuses {
		st.OrdersByStatus[s] = 0
	}

	counts, err := r.q.CountStoreOrdersByStatus(ctx, storeID)
	if err != nil {
		return SellerStats{}, err
	}
	for _, c := range counts {
		st.OrdersByStatus[c.Status] = c.OrderCount
	}

	totals, err := r.q.StoreRevenueTotals(ctx, queries.StoreRevenueTotalsParams{
		StoreID: storeID, Statuses: RevenueStatuses, Since: since,
	})
	if err != nil {
		return SellerStats{}, err
	}
	st.RevenueAll, st.RevenueWindow = totals.RevenueAllTime, totals.RevenueWindow
	st.UnitsAll, st.UnitsWindow = totals.UnitsAllTime, totals.UnitsWindow

	top, err := r.q.ListStoreTopProducts(ctx, queries.ListStoreTopProductsParams{
		StoreID: storeID, Statuses: RevenueStatuses, Since: since, TopLimit: TopProductsLimit,
	})
	if err != nil {
		return SellerStats{}, err
	}
	st.TopProducts = make([]TopProduct, len(top))
	for i, p := range top {
		st.TopProducts[i] = TopProduct{ProductID: p.ProductID, Name: p.Name, Status: p.Status, Units: p.Units, Revenue: p.Revenue}
	}

	rt, err := r.q.GetStoreRatingTotals(ctx, storeID)
	if err != nil {
		return SellerStats{}, err
	}
	st.RatingCount = rt.RatingCount
	st.RatingAverage = rating.Average(rt.RatingCount, rt.RatingTotal)
	return st, nil
}
