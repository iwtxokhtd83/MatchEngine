package orderbook

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

// orderSide maintains one side of the order book as sorted price levels.
type orderSide struct {
	prices []decimal.Decimal          // sorted price keys
	levels map[string]*priceLevel     // price.String() -> level
	less   func(a, b decimal.Decimal) bool // sort comparator
	count  int                         // total order count
}

func newOrderSide(less func(a, b decimal.Decimal) bool) *orderSide {
	return &orderSide{
		prices: make([]decimal.Decimal, 0),
		levels: make(map[string]*priceLevel),
		less:   less,
	}
}

// insert adds an order to the correct price level using binary search.
// O(log p) to find the level, O(1) amortized to append to the level's queue.
func (s *orderSide) insert(order *model.Order) {
	key := order.Price.String()
	pl, exists := s.levels[key]
	if !exists {
		pl = newPriceLevel()
		s.levels[key] = pl
		// Binary search for insertion point
		idx := sort.Search(len(s.prices), func(i int) bool {
			return s.less(order.Price, s.prices[i]) || order.Price.Equal(s.prices[i])
		})
		// Insert price at idx
		s.prices = append(s.prices, decimal.Zero)
		copy(s.prices[idx+1:], s.prices[idx:])
		s.prices[idx] = order.Price
	}
	pl.append(order)
	s.count++
}

// remove removes an order by ID from its price level. O(1) level lookup + O(k) scan within level.
func (s *orderSide) remove(order *model.Order) {
	key := order.Price.String()
	pl, exists := s.levels[key]
	if !exists {
		return
	}
	if pl.remove(order.ID) {
		s.count--
		if pl.len() == 0 {
			s.removePrice(key, order.Price)
		}
	}
}

// removePrice removes an empty price level.
func (s *orderSide) removePrice(key string, price decimal.Decimal) {
	delete(s.levels, key)
	for i, p := range s.prices {
		if p.Equal(price) {
			s.prices = append(s.prices[:i], s.prices[i+1:]...)
			return
		}
	}
}

// best returns the best (first) order, or nil if empty. O(1).
func (s *orderSide) best() *model.Order {
	if len(s.prices) == 0 {
		return nil
	}
	key := s.prices[0].String()
	return s.levels[key].front()
}

// removeFilled removes all filled orders from all levels, cleaning up empty levels.
func (s *orderSide) removeFilled() []*model.Order {
	var filled []*model.Order
	pricesToRemove := make([]int, 0)

	for i, price := range s.prices {
		key := price.String()
		pl := s.levels[key]
		// Collect filled orders before removing
		for _, o := range pl.allOrders() {
			if o.IsFilled() {
				filled = append(filled, o)
				s.count--
			}
		}
		remaining := pl.removeFilled()
		if remaining == 0 {
			delete(s.levels, key)
			pricesToRemove = append(pricesToRemove, i)
		}
	}

	// Remove empty price levels from slice (reverse order to preserve indices)
	for i := len(pricesToRemove) - 1; i >= 0; i-- {
		idx := pricesToRemove[i]
		s.prices = append(s.prices[:idx], s.prices[idx+1:]...)
	}

	return filled
}

// flatten returns all orders in price-time priority order.
func (s *orderSide) flatten() []*model.Order {
	result := make([]*model.Order, 0, s.count)
	for _, price := range s.prices {
		key := price.String()
		pl := s.levels[key]
		result = append(result, pl.allOrders()...)
	}
	return result
}

// OrderBook maintains buy and sell orders using price-level maps with sorted price keys.
//
// Complexity:
//   - Insert: O(log p) where p = number of distinct price levels
//   - Remove by ID: O(1) map lookup + O(k) within price level
//   - Best price: O(1)
//   - Snapshot: O(n)
type OrderBook struct {
	bids   *orderSide
	asks   *orderSide
	orders map[string]*model.Order // order ID -> order for O(1) lookup
}

// New creates a new empty order book.
func New() *OrderBook {
	return &OrderBook{
		// Bids: best = highest price first
		bids: newOrderSide(func(a, b decimal.Decimal) bool {
			return a.GreaterThan(b)
		}),
		// Asks: best = lowest price first
		asks: newOrderSide(func(a, b decimal.Decimal) bool {
			return a.LessThan(b)
		}),
		orders: make(map[string]*model.Order),
	}
}

// HasOrder returns true if an order with the given ID exists in the book.
func (ob *OrderBook) HasOrder(orderID string) bool {
	_, ok := ob.orders[orderID]
	return ok
}

// AddOrder inserts an order into the appropriate side of the book.
func (ob *OrderBook) AddOrder(order *model.Order) {
	ob.orders[order.ID] = order
	if order.Side == model.Buy {
		ob.bids.insert(order)
	} else {
		ob.asks.insert(order)
	}
}

// RemoveOrder removes an order by ID from the book.
func (ob *OrderBook) RemoveOrder(orderID string) bool {
	order, ok := ob.orders[orderID]
	if !ok {
		return false
	}
	delete(ob.orders, orderID)
	if order.Side == model.Buy {
		ob.bids.remove(order)
	} else {
		ob.asks.remove(order)
	}
	return true
}

// BestBid returns the highest-priced buy order, or nil if empty.
func (ob *OrderBook) BestBid() *model.Order {
	return ob.bids.best()
}

// BestAsk returns the lowest-priced sell order, or nil if empty.
func (ob *OrderBook) BestAsk() *model.Order {
	return ob.asks.best()
}

// Spread returns the difference between best ask and best bid.
// Returns decimal.NewFromInt(-1) if either side is empty.
func (ob *OrderBook) Spread() decimal.Decimal {
	bid := ob.BestBid()
	ask := ob.BestAsk()
	if bid == nil || ask == nil {
		return decimal.NewFromInt(-1)
	}
	return ask.Price.Sub(bid.Price)
}

// Depth returns the number of orders on each side.
func (ob *OrderBook) Depth() (bids, asks int) {
	return ob.bids.count, ob.asks.count
}

// Bids returns all bid orders in price-time priority order (highest price first).
func (ob *OrderBook) Bids() []*model.Order {
	return ob.bids.flatten()
}

// Asks returns all ask orders in price-time priority order (lowest price first).
func (ob *OrderBook) Asks() []*model.Order {
	return ob.asks.flatten()
}

// RemoveFilled removes all fully filled orders from both sides.
func (ob *OrderBook) RemoveFilled() {
	for _, o := range ob.bids.removeFilled() {
		delete(ob.orders, o.ID)
	}
	for _, o := range ob.asks.removeFilled() {
		delete(ob.orders, o.ID)
	}
}

// Snapshot returns a deep copy of the order book for safe read access.
func (ob *OrderBook) Snapshot() *OrderBook {
	snap := New()
	for _, o := range ob.bids.flatten() {
		cp := *o
		snap.orders[cp.ID] = &cp
		snap.bids.insert(&cp)
	}
	for _, o := range ob.asks.flatten() {
		cp := *o
		snap.orders[cp.ID] = &cp
		snap.asks.insert(&cp)
	}
	return snap
}
