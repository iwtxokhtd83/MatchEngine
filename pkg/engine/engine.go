package engine

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iwtxokhtd83/MatchEngine/pkg/model"
	"github.com/iwtxokhtd83/MatchEngine/pkg/orderbook"
)

const defaultMaxTradeLog = 10000

// TradeHandler is a callback invoked when a trade is executed.
type TradeHandler func(symbol string, trade model.Trade)

// Engine is the core matching engine that processes orders and produces trades.
type Engine struct {
	mu          sync.Mutex
	nextID      uint64                          // atomic counter for ID generation
	idPrefix    string                          // optional prefix for generated IDs
	books       map[string]*orderbook.OrderBook // symbol -> order book
	orderIndex  map[string]string               // order ID -> symbol (for duplicate detection)
	trades      []model.Trade
	maxTrades   int
	onTrade     TradeHandler
	symbols     map[string]bool // registered symbols (nil = accept any)
}

// Option configures the engine.
type Option func(*Engine)

// WithMaxTradeLog sets the maximum number of trades kept in memory.
// Oldest trades are evicted when the limit is reached. Set to 0 to disable the in-memory log.
func WithMaxTradeLog(n int) Option {
	return func(e *Engine) {
		e.maxTrades = n
	}
}

// WithTradeHandler sets a callback that is invoked for every executed trade.
func WithTradeHandler(h TradeHandler) Option {
	return func(e *Engine) {
		e.onTrade = h
	}
}

// WithIDPrefix sets a prefix for auto-generated order IDs (e.g., "ORD-").
func WithIDPrefix(prefix string) Option {
	return func(e *Engine) {
		e.idPrefix = prefix
	}
}

// New creates a new matching engine.
func New(opts ...Option) *Engine {
	e := &Engine{
		books:      make(map[string]*orderbook.OrderBook),
		orderIndex: make(map[string]string),
		trades:     make([]model.Trade, 0),
		maxTrades:  defaultMaxTradeLog,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// RegisterSymbol registers a valid trading symbol. If any symbols are registered,
// only registered symbols are accepted by SubmitOrder.
func (e *Engine) RegisterSymbol(symbol string) error {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.symbols == nil {
		e.symbols = make(map[string]bool)
	}
	e.symbols[symbol] = true
	return nil
}

// normalizeSymbol trims whitespace and uppercases the symbol.
func normalizeSymbol(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// validateSymbol checks that the symbol is valid.
func (e *Engine) validateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if e.symbols != nil && !e.symbols[symbol] {
		return fmt.Errorf("symbol %q is not registered", symbol)
	}
	return nil
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

// generateID returns a unique, monotonically increasing order ID.
func (e *Engine) generateID() string {
	id := atomic.AddUint64(&e.nextID, 1)
	return e.idPrefix + strconv.FormatUint(id, 10)
}

// SubmitLimitOrder creates a limit order with an auto-generated ID and submits it.
// Returns the generated order ID, any trades, and an error if invalid.
func (e *Engine) SubmitLimitOrder(symbol string, side model.Side, price, quantity decimal.Decimal) (string, []model.Trade, error) {
	id := e.generateID()
	order := model.NewLimitOrder(id, side, price, quantity)
	trades, err := e.SubmitOrder(symbol, order)
	return id, trades, err
}

// SubmitMarketOrder creates a market order with an auto-generated ID and submits it.
// Returns the generated order ID, any trades, and an error if invalid.
func (e *Engine) SubmitMarketOrder(symbol string, side model.Side, quantity decimal.Decimal) (string, []model.Trade, error) {
	id := e.generateID()
	order := model.NewMarketOrder(id, side, quantity)
	trades, err := e.SubmitOrder(symbol, order)
	return id, trades, err
}

// SubmitRequest creates an order from an OrderRequest with an auto-generated ID and submits it.
// Returns the generated order ID, any trades, and an error if invalid.
func (e *Engine) SubmitRequest(symbol string, req model.OrderRequest) (string, []model.Trade, error) {
	id := e.generateID()
	var order *model.Order
	if req.Type == model.Market {
		order = model.NewMarketOrder(id, req.Side, req.Quantity)
	} else {
		order = model.NewLimitOrder(id, req.Side, req.Price, req.Quantity)
	}
	trades, err := e.SubmitOrder(symbol, order)
	return id, trades, err
}

// SubmitOrder processes an incoming order, attempting to match it against the book.
// Returns any trades generated and an error if the order is invalid.
// For auto-generated IDs, prefer SubmitLimitOrder, SubmitMarketOrder, or SubmitRequest.
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

	symbol = normalizeSymbol(symbol)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate symbol
	if err := e.validateSymbol(symbol); err != nil {
		return nil, err
	}

	// Check for duplicate order ID
	if _, exists := e.orderIndex[order.ID]; exists {
		return nil, fmt.Errorf("duplicate order ID: %s", order.ID)
	}

	book := e.getOrCreateBook(symbol)
	trades := e.match(book, order)

	// If the order is not fully filled and it's a limit order, add remainder to book
	if !order.IsFilled() && order.Type == model.Limit {
		book.AddOrder(order)
		e.orderIndex[order.ID] = symbol
	}

	// Record trades
	for _, t := range trades {
		if e.onTrade != nil {
			e.onTrade(symbol, t)
		}
		if e.maxTrades > 0 {
			if len(e.trades) >= e.maxTrades {
				// Evict oldest 10%
				evict := e.maxTrades / 10
				if evict < 1 {
					evict = 1
				}
				e.trades = e.trades[evict:]
			}
			e.trades = append(e.trades, t)
		}
	}

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

	// Clean up filled orders from book and index
	e.cleanFilled(book)
	return trades
}

// cleanFilled removes filled orders from the book and the order index.
func (e *Engine) cleanFilled(book *orderbook.OrderBook) {
	for _, o := range book.Bids() {
		if o.IsFilled() {
			delete(e.orderIndex, o.ID)
		}
	}
	for _, o := range book.Asks() {
		if o.IsFilled() {
			delete(e.orderIndex, o.ID)
		}
	}
	book.RemoveFilled()
}

// matchBuy matches a buy order against the ask side.
func (e *Engine) matchBuy(book *orderbook.OrderBook, order *model.Order) []model.Trade {
	var trades []model.Trade

	for !order.IsFilled() {
		bestAsk := book.BestAsk()
		if bestAsk == nil {
			break
		}
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
	symbol = normalizeSymbol(symbol)

	e.mu.Lock()
	defer e.mu.Unlock()

	book, ok := e.books[symbol]
	if !ok {
		return false
	}
	if book.RemoveOrder(orderID) {
		delete(e.orderIndex, orderID)
		return true
	}
	return false
}

// GetTrades returns a copy of the in-memory trade log.
func (e *Engine) GetTrades() []model.Trade {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]model.Trade, len(e.trades))
	copy(result, e.trades)
	return result
}

// GetOrderBook returns a snapshot (deep copy) of the order book for a symbol.
// Safe for concurrent read access. Returns nil if the symbol has no book.
func (e *Engine) GetOrderBook(symbol string) *orderbook.OrderBook {
	symbol = normalizeSymbol(symbol)

	e.mu.Lock()
	defer e.mu.Unlock()

	book, ok := e.books[symbol]
	if !ok {
		return nil
	}
	return book.Snapshot()
}
