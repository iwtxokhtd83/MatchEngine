package orderbook

import "github.com/iwtxokhtd83/MatchEngine/pkg/model"

// priceLevel holds all orders at a single price point in FIFO order.
type priceLevel struct {
	orders []*model.Order
}

func newPriceLevel() *priceLevel {
	return &priceLevel{orders: make([]*model.Order, 0, 4)}
}

func (pl *priceLevel) append(order *model.Order) {
	pl.orders = append(pl.orders, order)
}

func (pl *priceLevel) front() *model.Order {
	if len(pl.orders) == 0 {
		return nil
	}
	return pl.orders[0]
}

func (pl *priceLevel) remove(orderID string) bool {
	for i, o := range pl.orders {
		if o.ID == orderID {
			pl.orders = append(pl.orders[:i], pl.orders[i+1:]...)
			return true
		}
	}
	return false
}

func (pl *priceLevel) len() int {
	return len(pl.orders)
}

// removeFilled removes all filled orders and returns the remaining count.
func (pl *priceLevel) removeFilled() int {
	n := 0
	for _, o := range pl.orders {
		if !o.IsFilled() {
			pl.orders[n] = o
			n++
		}
	}
	pl.orders = pl.orders[:n]
	return n
}

// allOrders returns all orders at this price level in FIFO order.
func (pl *priceLevel) allOrders() []*model.Order {
	return pl.orders
}
