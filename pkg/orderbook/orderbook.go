package orderbook

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

// OrderBook maintains buy and sell orders sorted by price-time priority.
type OrderBook struct {
	Bids   []*model.Order          // buy orders: highest price first
	Asks   []*model.Order          // sell orders: lowest price first
	orders map[string]*model.Order // order ID -> order for O(1) lookup
}

// New creates a new empty order book.
func New() *OrderBook {
	return &OrderBook{
		Bids:   make([]*model.Order, 0),
		Asks:   make([]*model.Order, 0),
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
		ob.Bids = append(ob.Bids, order)
		sort.SliceStable(ob.Bids, func(i, j int) bool {
			if ob.Bids[i].Price.Equal(ob.Bids[j].Price) {
				return ob.Bids[i].Timestamp.Before(ob.Bids[j].Timestamp)
			}
			return ob.Bids[i].Price.GreaterThan(ob.Bids[j].Price)
		})
	} else {
		ob.Asks = append(ob.Asks, order)
		sort.SliceStable(ob.Asks, func(i, j int) bool {
			if ob.Asks[i].Price.Equal(ob.Asks[j].Price) {
				return ob.Asks[i].Timestamp.Before(ob.Asks[j].Timestamp)
			}
			return ob.Asks[i].Price.LessThan(ob.Asks[j].Price)
		})
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
		ob.Bids = removeFromSlice(ob.Bids, orderID)
	} else {
		ob.Asks = removeFromSlice(ob.Asks, orderID)
	}
	return true
}

func removeFromSlice(orders []*model.Order, id string) []*model.Order {
	for i, o := range orders {
		if o.ID == id {
			return append(orders[:i], orders[i+1:]...)
		}
	}
	return orders
}

// BestBid returns the highest-priced buy order, or nil if empty.
func (ob *OrderBook) BestBid() *model.Order {
	if len(ob.Bids) == 0 {
		return nil
	}
	return ob.Bids[0]
}

// BestAsk returns the lowest-priced sell order, or nil if empty.
func (ob *OrderBook) BestAsk() *model.Order {
	if len(ob.Asks) == 0 {
		return nil
	}
	return ob.Asks[0]
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
	return len(ob.Bids), len(ob.Asks)
}

// RemoveFilled removes all fully filled orders from both sides.
func (ob *OrderBook) RemoveFilled() {
	ob.Bids = ob.filterFilled(ob.Bids)
	ob.Asks = ob.filterFilled(ob.Asks)
}

func (ob *OrderBook) filterFilled(orders []*model.Order) []*model.Order {
	result := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		if !o.IsFilled() {
			result = append(result, o)
		} else {
			delete(ob.orders, o.ID)
		}
	}
	return result
}

// Snapshot returns a deep copy of the order book for safe read access.
func (ob *OrderBook) Snapshot() *OrderBook {
	snap := &OrderBook{
		Bids:   make([]*model.Order, len(ob.Bids)),
		Asks:   make([]*model.Order, len(ob.Asks)),
		orders: make(map[string]*model.Order),
	}
	for i, o := range ob.Bids {
		cp := *o
		snap.Bids[i] = &cp
		snap.orders[cp.ID] = &cp
	}
	for i, o := range ob.Asks {
		cp := *o
		snap.Asks[i] = &cp
		snap.orders[cp.ID] = &cp
	}
	return snap
}
