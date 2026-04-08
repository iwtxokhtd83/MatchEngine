# API Reference

## Engine

The `Engine` is the main entry point. It manages order books for multiple symbols and coordinates matching.

### Create Engine

```go
import "github.com/iwtxokhtd83/MatchEngine/pkg/engine"

// Default engine
e := engine.New()

// With options
e := engine.New(
    engine.WithMaxTradeLog(5000),
    engine.WithTradeHandler(func(symbol string, trade model.Trade) {
        log.Printf("Trade on %s: %s @ %s", symbol, trade.Quantity, trade.Price)
    }),
)
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithMaxTradeLog(n)` | Max trades kept in memory. Oldest evicted when full. Set to 0 to disable. | 10000 |
| `WithTradeHandler(fn)` | Callback invoked for every executed trade. | nil |

---

### RegisterSymbol

```go
func (e *Engine) RegisterSymbol(symbol string) error
```

Registers a valid trading symbol. Once any symbol is registered, only registered symbols are accepted by `SubmitOrder`. Symbols are normalized (uppercased, trimmed).

**Example:**

```go
e.RegisterSymbol("BTC/USD")
e.RegisterSymbol("ETH/USD")
// Now only BTC/USD and ETH/USD are accepted
```

---

### SubmitOrder

```go
func (e *Engine) SubmitOrder(symbol string, order *model.Order) ([]model.Trade, error)
```

Submits an order to the engine. Symbols are automatically normalized (uppercased, trimmed).

**Errors:**
- `order cannot be nil`
- `order quantity must be positive`
- `limit order price must be positive`
- `symbol cannot be empty`
- `symbol "X" is not registered` (when symbol registration is used)
- `duplicate order ID: X`

**Example:**

```go
order := model.NewLimitOrder("buy-1", model.Buy, decimal.NewFromInt(100), decimal.NewFromInt(10))
trades, err := e.SubmitOrder("AAPL", order)
```

---

### CancelOrder

```go
func (e *Engine) CancelOrder(symbol, orderID string) bool
```

Removes a resting order from the book. The order ID becomes available for reuse after cancellation.

**Returns:** `true` if the order was found and removed, `false` otherwise.

---

### GetTrades

```go
func (e *Engine) GetTrades() []model.Trade
```

Returns a copy of the in-memory trade log. The log size is bounded by `WithMaxTradeLog`.

---

### GetOrderBook

```go
func (e *Engine) GetOrderBook(symbol string) *orderbook.OrderBook
```

Returns a deep-copy snapshot of the order book. Safe for concurrent read access — mutations to the snapshot do not affect the engine's internal state.

**Example:**

```go
book := e.GetOrderBook("BTC/USD")
if book != nil {
    fmt.Printf("Best bid: %s\n", book.BestBid().Price.StringFixed(2))
    fmt.Printf("Best ask: %s\n", book.BestAsk().Price.StringFixed(2))
    fmt.Printf("Spread: %s\n", book.Spread().StringFixed(2))
}
```

---

## Model Constructors

### NewLimitOrder

```go
func NewLimitOrder(id string, side Side, price, quantity decimal.Decimal) *Order
```

### NewMarketOrder

```go
func NewMarketOrder(id string, side Side, quantity decimal.Decimal) *Order
```

> All price and quantity fields use [`shopspring/decimal`](https://github.com/shopspring/decimal) for exact decimal arithmetic.

---

## Enums

### Side

```go
const (
    Buy  Side = iota  // 0
    Sell              // 1
)
```

### OrderType

```go
const (
    Limit  OrderType = iota  // 0
    Market                   // 1
)
```
