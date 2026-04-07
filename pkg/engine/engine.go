package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
	"github.com/iwtxokhtd83/MatchEngine/pkg/orderbook"
)

// Engine is the core matching engine that processes orders and produces trades.
type Engine struct {
	mu     sync.Mutex
	books  map[string]*orderbook.OrderBook // symbol -> order book
	trades []model.Trade
}

// New creates a new matching engine.
func New() *Engine {
	return &Engine{
		books:  make(map[string]*orderbook.OrderBook),
		trades: make([]model.Trade, 0),
	}
}

// getOrCreateBook returns the order book for a symbol, creating one if needed.
func (e *Engine) getOrCreateBook(symbol string) *orderbook.OrderBook {
	book, ok := e.books[symbol]
	if !ok {
		book = orderbook.New()
		e.books[symbol] = book
	}
	return book
}

// SubmitOrder processes an incoming order, attempting to match it against the book.
// Returns any trades generated and an error if the order is invalid.
func (e *Engine) SubmitOrder(symbol string, order *model.Order) ([]model.Trade, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}
	if order.Remaining.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("order quantity must be positive")
	}
	if order.Type == model.Limit && order.Price.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("limit order price must be positive")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	book := e.getOrCreateBook(symbol)
	trades := e.match(book, order)

	// If the order is not fully filled and it's a limit order, add remainder to book
	if !order.IsFilled() && order.Type == model.Limit {
		book.AddOrder(order)
	}

	e.trades = append(e.trades, trades...)
	return trades, nil
}

// match attempts to fill the incoming order against the opposite side of the book.
func (e *Engine) match(book *orderbook.OrderBook, order *model.Order) []model.Trade {
	var trades []model.Trade

	if order.Side == model.Buy {
		trades = e.matchBuy(book, order)
	} else {
		trades = e.matchSell(book, order)
	}

	book.RemoveFilled()
	return trades
}

// matchBuy matches a buy order against the ask side.
func (e *Engine) matchBuy(book *orderbook.OrderBook, order *model.Order) []model.Trade {
	var trades []model.Trade

	for !order.IsFilled() {
		bestAsk := book.BestAsk()
		if bestAsk == nil {
			break
		}

		// For limit orders, stop if ask price is higher than bid price
		if order.Type == model.Limit && bestAsk.Price.GreaterThan(order.Price) {
			break
		}

		trade := executeTrade(order, bestAsk, bestAsk.Price)
		trades = append(trades, trade)
	}

	return trades
}

// matchSell matches a sell order against the bid side.
func (e *Engine) matchSell(book *orderbook.OrderBook, order *model.Order) []model.Trade {
	var trades []model.Trade

	for !order.IsFilled() {
		bestBid := book.BestBid()
		if bestBid == nil {
			break
		}

		// For limit orders, stop if bid price is lower than ask price
		if order.Type == model.Limit && bestBid.Price.LessThan(order.Price) {
			break
		}

		trade := executeTrade(bestBid, order, bestBid.Price)
		trades = append(trades, trade)
	}

	return trades
}

// executeTrade fills the minimum quantity between two orders and returns a trade.
func executeTrade(buyOrder, sellOrder *model.Order, price decimal.Decimal) model.Trade {
	quantity := decimal.Min(buyOrder.Remaining, sellOrder.Remaining)
	buyOrder.Remaining = buyOrder.Remaining.Sub(quantity)
	sellOrder.Remaining = sellOrder.Remaining.Sub(quantity)

	return model.Trade{
		BuyOrderID:  buyOrder.ID,
		SellOrderID: sellOrder.ID,
		Price:       price,
		Quantity:    quantity,
		Timestamp:   time.Now(),
	}
}

// CancelOrder removes an order from the book.
func (e *Engine) CancelOrder(symbol, orderID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	book, ok := e.books[symbol]
	if !ok {
		return false
	}
	return book.RemoveOrder(orderID)
}

// GetTrades returns all executed trades.
func (e *Engine) GetTrades() []model.Trade {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]model.Trade, len(e.trades))
	copy(result, e.trades)
	return result
}

// GetOrderBook returns the order book for a symbol.
func (e *Engine) GetOrderBook(symbol string) *orderbook.OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.books[symbol]
}
