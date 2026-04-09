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
    engine.WithIDPrefix("ORD-"),
    engine.WithMaxTradeLog(5000),
    engine.WithTradeHandler(func(symbol string, trade model.Trade) {
        log.Printf("Trade on %s: %s @ %s", symbol, trade.Quantity, trade.Price)
    }),
)
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithIDPrefix(s)` | Prefix for auto-generated order IDs (e.g., `"ORD-"` produces `"ORD-1"`, `"ORD-2"`, ...) | `""` |
| `WithMaxTradeLog(n)` | Max trades kept in memory. Oldest evicted when full. Set to 0 to disable. | 10000 |
| `WithTradeHandler(fn)` | Callback invoked for every executed trade. | nil |

---

### SubmitLimitOrder (recommended)

```go
func (e *Engine) SubmitLimitOrder(symbol string, side model.Side, price, quantity decimal.Decimal) (string, []model.Trade, error)
```

Creates a limit order with an auto-generated unique ID and submits it.

**Returns:** `(orderID, trades, error)`

**Example:**

```go
id, trades, err := e.SubmitLimitOrder("BTC/USD", model.Sell, decimal.NewFromInt(50000), decimal.NewFromInt(1))
fmt.Printf("Order %s placed, %d trades\n", id, len(trades))
```

---

### SubmitMarketOrder (recommended)

```go
func (e *Engine) SubmitMarketOrder(symbol string, side model.Side, quantity decimal.Decimal) (string, []model.Trade, error)
```

Creates a market order with an auto-generated unique ID and submits it.

**Returns:** `(orderID, trades, error)`

**Example:**

```go
id, trades, err := e.SubmitMarketOrder("BTC/USD", model.Buy, decimal.RequireFromString("0.5"))
```

---

### SubmitRequest

```go
func (e *Engine) SubmitRequest(symbol string, req model.OrderRequest) (string, []model.Trade, error)
```

Creates an order from an `OrderRequest` struct with an auto-generated ID and submits it.

**Example:**

```go
req := model.NewLimitOrderRequest(model.Buy, decimal.NewFromInt(100), decimal.NewFromInt(10))
id, trades, err := e.SubmitRequest("AAPL", req)
```

---

### SubmitOrder (legacy)

```go
func (e *Engine) SubmitOrder(symbol string, order *model.Order) ([]model.Trade, error)
```

Submits an order with a caller-provided ID. Kept for backward compatibility. For new code, prefer `SubmitLimitOrder`, `SubmitMarketOrder`, or `SubmitRequest`.

---

### RegisterSymbol

```go
func (e *Engine) RegisterSymbol(symbol string) error
```

Registers a valid trading symbol. Once any symbol is registered, only registered symbols are accepted.

---

### CancelOrder

```go
func (e *Engine) CancelOrder(symbol, orderID string) bool
```

Removes a resting order from the book. Works with both auto-generated and caller-provided IDs.

---

### GetTrades

```go
func (e *Engine) GetTrades() []model.Trade
```

Returns a copy of the in-memory trade log.

---

### GetOrderBook

```go
func (e *Engine) GetOrderBook(symbol string) *orderbook.OrderBook
```

Returns a deep-copy snapshot of the order book. Safe for concurrent read access.

---

## OrderRequest

`OrderRequest` is used with `SubmitRequest` for a cleaner API without manual ID management.

```go
// Limit order request
req := model.NewLimitOrderRequest(model.Buy, price, quantity)

// Market order request
req := model.NewMarketOrderRequest(model.Sell, quantity)
```

---

## Model Constructors (for use with SubmitOrder)

### NewLimitOrder

```go
func NewLimitOrder(id string, side Side, price, quantity decimal.Decimal) *Order
```

### NewMarketOrder

```go
func NewMarketOrder(id string, side Side, quantity decimal.Decimal) *Order
```

---

## Enums

### Side

```go
const (
    Buy  Side = iota
    Sell
)
```

### OrderType

```go
const (
    Limit  OrderType = iota
    Market
)
```
