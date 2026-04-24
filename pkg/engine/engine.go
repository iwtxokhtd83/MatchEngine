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
	stpMode     model.STPMode                   // self-trade prevention mode
	books       map[string]*orderbook.OrderBook // symbol -> order book
	orderIndex  map[string]string               // order ID -> symbol (for duplicate detection)
	stopOrders  map[string][]*model.Order        // symbol -> pending stop orders
	lastPrices  map[string]decimal.Decimal       // symbol -> last trade price
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

// WithSTPMode sets the self-trade prevention mode.
// Default is STPNone (disabled). When enabled, orders from the same OwnerID
// will not match against each other.
func WithSTPMode(mode model.STPMode) Option {
	return func(e *Engine) {
		e.stpMode = mode
	}
}

// New creates a new matching engine.
func New(opts ...Option) *Engine {
	e := &Engine{
		books:      make(map[string]*orderbook.OrderBook),
		orderIndex: make(map[string]string),
		stopOrders: make(map[string][]*model.Order),
		lastPrices: make(map[string]decimal.Decimal),
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
	switch req.Type {
	case model.Market:
		order = model.NewMarketOrder(id, req.Side, req.Quantity)
	case model.StopMarket, model.StopLimit:
		order = &model.Order{
			ID:        id,
			Side:      req.Side,
			Type:      req.Type,
			Price:     req.Price,
			StopPrice: req.StopPrice,
			Quantity:  req.Quantity,
			Remaining: req.Quantity,
			Timestamp: time.Now(),
		}
	default: // Limit
		order = model.NewLimitOrder(id, req.Side, req.Price, req.Quantity)
	}
	order.OwnerID = req.OwnerID
	order.TIF = req.TIF
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
	if (order.Type == model.Limit || order.Type == model.StopLimit) && order.Price.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("limit order price must be positive")
	}
	if (order.Type == model.StopMarket || order.Type == model.StopLimit) && order.StopPrice.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("stop order requires a positive stop price")
	}

	symbol = normalizeSymbol(symbol)

	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.validateSymbol(symbol); err != nil {
		return nil, err
	}

	if _, exists := e.orderIndex[order.ID]; exists {
		return nil, fmt.Errorf("duplicate order ID: %s", order.ID)
	}

	// Stop orders: store and wait for trigger
	if order.Type == model.StopMarket || order.Type == model.StopLimit {
		e.stopOrders[symbol] = append(e.stopOrders[symbol], order)
		e.orderIndex[order.ID] = symbol
		return nil, nil
	}

	// FOK pre-check: verify enough liquidity exists
	if order.TIF == model.FOK {
		book := e.getOrCreateBook(symbol)
		if !e.canFillCompletely(book, order) {
			return nil, nil // silently reject — not enough liquidity
		}
	}

	book := e.getOrCreateBook(symbol)
	trades := e.match(book, order)

	// IOC: discard any remaining quantity (never rests in book)
	if order.TIF == model.IOC {
		order.Remaining = decimal.Zero
	}

	// If the order is not fully filled and it's a limit order with GTC, add to book
	if !order.IsFilled() && order.Type == model.Limit {
		book.AddOrder(order)
		e.orderIndex[order.ID] = symbol
	}

	// Record trades and update last price
	e.recordTrades(symbol, trades)

	// Check if any stop orders should be triggered
	if len(trades) > 0 {
		triggered := e.triggerStopOrders(symbol)
		trades = append(trades, triggered...)
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

// isSelfTrade returns true if both orders have the same non-empty OwnerID.
func isSelfTrade(a, b *model.Order) bool {
	return a.OwnerID != "" && a.OwnerID == b.OwnerID
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

		// Self-trade prevention
		if e.stpMode != model.STPNone && isSelfTrade(order, bestAsk) {
			if e.handleSTP(book, order, bestAsk) {
				continue // resting was removed, try next level
			}
			break // incoming was cancelled
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

		// Self-trade prevention
		if e.stpMode != model.STPNone && isSelfTrade(order, bestBid) {
			if e.handleSTP(book, order, bestBid) {
				continue
			}
			break
		}

		trade := executeTrade(bestBid, order, bestBid.Price)
		trades = append(trades, trade)
	}

	return trades
}

// handleSTP applies the self-trade prevention policy.
// Returns true if matching should continue (resting removed), false if incoming was cancelled.
func (e *Engine) handleSTP(book *orderbook.OrderBook, incoming, resting *model.Order) bool {
	switch e.stpMode {
	case model.STPCancelResting:
		// Cancel the resting order, continue matching incoming
		book.RemoveOrder(resting.ID)
		delete(e.orderIndex, resting.ID)
		return true

	case model.STPCancelIncoming:
		// Cancel the incoming order entirely
		incoming.Remaining = decimal.Zero
		return false

	case model.STPCancelBoth:
		// Cancel both orders
		book.RemoveOrder(resting.ID)
		delete(e.orderIndex, resting.ID)
		incoming.Remaining = decimal.Zero
		return false

	case model.STPDecrement:
		// Reduce both by the overlap quantity without producing a trade
		overlap := decimal.Min(incoming.Remaining, resting.Remaining)
		incoming.Remaining = incoming.Remaining.Sub(overlap)
		resting.Remaining = resting.Remaining.Sub(overlap)
		if resting.IsFilled() {
			book.RemoveOrder(resting.ID)
			delete(e.orderIndex, resting.ID)
		}
		if incoming.IsFilled() {
			return false
		}
		return true

	default:
		return true
	}
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

// canFillCompletely checks if the book has enough liquidity to fill the order entirely (for FOK).
func (e *Engine) canFillCompletely(book *orderbook.OrderBook, order *model.Order) bool {
	available := decimal.Zero
	if order.Side == model.Buy {
		for _, ask := range book.Asks() {
			if order.Type == model.Limit && ask.Price.GreaterThan(order.Price) {
				break
			}
			available = available.Add(ask.Remaining)
			if available.GreaterThanOrEqual(order.Remaining) {
				return true
			}
		}
	} else {
		for _, bid := range book.Bids() {
			if order.Type == model.Limit && bid.Price.LessThan(order.Price) {
				break
			}
			available = available.Add(bid.Remaining)
			if available.GreaterThanOrEqual(order.Remaining) {
				return true
			}
		}
	}
	return false
}

// recordTrades records trades to the log and updates the last price.
func (e *Engine) recordTrades(symbol string, trades []model.Trade) {
	for _, t := range trades {
		e.lastPrices[symbol] = t.Price
		if e.onTrade != nil {
			e.onTrade(symbol, t)
		}
		if e.maxTrades > 0 {
			if len(e.trades) >= e.maxTrades {
				evict := e.maxTrades / 10
				if evict < 1 {
					evict = 1
				}
				e.trades = e.trades[evict:]
			}
			e.trades = append(e.trades, t)
		}
	}
}

// triggerStopOrders checks pending stop orders and activates any that are triggered.
func (e *Engine) triggerStopOrders(symbol string) []model.Trade {
	lastPrice, ok := e.lastPrices[symbol]
	if !ok {
		return nil
	}

	stops := e.stopOrders[symbol]
	var remaining []*model.Order
	var allTrades []model.Trade

	for _, stop := range stops {
		if e.isStopTriggered(stop, lastPrice) {
			delete(e.orderIndex, stop.ID)
			// Convert stop to active order
			var active *model.Order
			if stop.Type == model.StopMarket {
				active = model.NewMarketOrder(stop.ID, stop.Side, stop.Remaining)
			} else { // StopLimit
				active = model.NewLimitOrder(stop.ID, stop.Side, stop.Price, stop.Remaining)
			}
			active.OwnerID = stop.OwnerID

			book := e.getOrCreateBook(symbol)
			trades := e.match(book, active)

			if !active.IsFilled() && active.Type == model.Limit {
				book.AddOrder(active)
				e.orderIndex[active.ID] = symbol
			}

			e.recordTrades(symbol, trades)
			allTrades = append(allTrades, trades...)
		} else {
			remaining = append(remaining, stop)
		}
	}

	e.stopOrders[symbol] = remaining
	return allTrades
}

// isStopTriggered checks if a stop order should be activated based on the last trade price.
func (e *Engine) isStopTriggered(stop *model.Order, lastPrice decimal.Decimal) bool {
	if stop.Side == model.Buy {
		// Buy stop triggers when price rises to or above stop price
		return lastPrice.GreaterThanOrEqual(stop.StopPrice)
	}
	// Sell stop triggers when price falls to or below stop price
	return lastPrice.LessThanOrEqual(stop.StopPrice)
}

// CancelOrder removes an order from the book.
func (e *Engine) CancelOrder(symbol, orderID string) bool {
	symbol = normalizeSymbol(symbol)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Try to cancel from order book
	book, ok := e.books[symbol]
	if ok && book.RemoveOrder(orderID) {
		delete(e.orderIndex, orderID)
		return true
	}

	// Try to cancel from stop orders
	stops := e.stopOrders[symbol]
	for i, s := range stops {
		if s.ID == orderID {
			e.stopOrders[symbol] = append(stops[:i], stops[i+1:]...)
			delete(e.orderIndex, orderID)
			return true
		}
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
