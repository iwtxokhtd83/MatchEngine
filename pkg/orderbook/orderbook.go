package orderbook

import (
	"sort"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
)

// OrderBook maintains buy and sell orders sorted by price-time priority.
type OrderBook struct {
	Bids []*model.Order // buy orders: highest price first
	Asks []*model.Order // sell orders: lowest price first
}

// New creates a new empty order book.
func New() *OrderBook {
	return &OrderBook{
		Bids: make([]*model.Order, 0),
		Asks: make([]*model.Order, 0),
	}
}

// AddOrder inserts an order into the appropriate side of the book.
func (ob *OrderBook) AddOrder(order *model.Order) {
	if order.Side == model.Buy {
		ob.Bids = append(ob.Bids, order)
		sort.SliceStable(ob.Bids, func(i, j int) bool {
			if ob.Bids[i].Price == ob.Bids[j].Price {
				return ob.Bids[i].Timestamp.Before(ob.Bids[j].Timestamp)
			}
			return ob.Bids[i].Price > ob.Bids[j].Price
		})
	} else {
		ob.Asks = append(ob.Asks, order)
		sort.SliceStable(ob.Asks, func(i, j int) bool {
			if ob.Asks[i].Price == ob.Asks[j].Price {
				return ob.Asks[i].Timestamp.Before(ob.Asks[j].Timestamp)
			}
			return ob.Asks[i].Price < ob.Asks[j].Price
		})
	}
}

// RemoveOrder removes an order by ID from the book.
func (ob *OrderBook) RemoveOrder(orderID string) bool {
	for i, o := range ob.Bids {
		if o.ID == orderID {
			ob.Bids = append(ob.Bids[:i], ob.Bids[i+1:]...)
			return true
		}
	}
	for i, o := range ob.Asks {
		if o.ID == orderID {
			ob.Asks = append(ob.Asks[:i], ob.Asks[i+1:]...)
			return true
		}
	}
	return false
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
// Returns -1 if either side is empty.
func (ob *OrderBook) Spread() float64 {
	bid := ob.BestBid()
	ask := ob.BestAsk()
	if bid == nil || ask == nil {
		return -1
	}
	return ask.Price - bid.Price
}

// Depth returns the number of orders on each side.
func (ob *OrderBook) Depth() (bids, asks int) {
	return len(ob.Bids), len(ob.Asks)
}

// removeFilled removes all fully filled orders from both sides.
func (ob *OrderBook) RemoveFilled() {
	ob.Bids = filterFilled(ob.Bids)
	ob.Asks = filterFilled(ob.Asks)
}

func filterFilled(orders []*model.Order) []*model.Order {
	result := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		if !o.IsFilled() {
			result = append(result, o)
		}
	}
	return result
}
