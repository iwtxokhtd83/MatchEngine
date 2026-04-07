# Building a Trade Matching Engine from Scratch in Go

Every exchange — whether it trades stocks, crypto, or commodities — has a matching engine at its core. It is the component that takes buy and sell orders, finds compatible pairs, and executes trades. Despite being so fundamental, the inner workings of a matching engine are rarely discussed in the open. Most implementations live behind proprietary walls.

This article walks through [MatchEngine](https://github.com/iwtxokhtd83/MatchEngine), an open-source order matching engine written in Go. We will cover the core algorithm, the data structures behind the order book, and the design decisions that keep the codebase small and understandable.

## Why Go?

Go is a natural fit for this kind of system:

- Goroutines and mutexes make concurrency straightforward without the complexity of async runtimes.
- The standard library is rich enough that zero external dependencies are needed.
- Compilation to a single static binary makes deployment trivial.
- The language's simplicity keeps the codebase readable — important for an open-source educational project.

## The Domain Model

A matching engine operates on two fundamental entities: orders and trades.

### Orders

An order represents a participant's intent to buy or sell a certain quantity at a certain price (or at any price, in the case of a market order).

```go
type Order struct {
    ID        string
    Side      Side      // Buy or Sell
    Type      OrderType // Limit or Market
    Price     float64
    Quantity  float64
    Remaining float64
    Timestamp time.Time
}
```

The `Remaining` field is key. When an order is partially filled, `Remaining` decreases while `Quantity` preserves the original size. This separation makes it easy to track fill progress without losing the original intent.

Two order types are supported:

- **Limit orders** specify a maximum buy price or minimum sell price. They rest in the order book if no immediate match is available.
- **Market orders** execute immediately at the best available price. They never enter the book — whatever isn't filled is simply dropped.

### Trades

A trade is the result of two orders being matched:

```go
type Trade struct {
    BuyOrderID  string
    SellOrderID string
    Price       float64
    Quantity    float64
    Timestamp   time.Time
}
```

Simple and immutable. Once a trade is created, it is a historical fact.

## The Order Book

The order book is the central data structure. It maintains two sorted lists:

- **Bids** (buy orders): sorted by price descending, then by time ascending.
- **Asks** (sell orders): sorted by price ascending, then by time ascending.

This sorting scheme implements **price-time priority** (also known as FIFO matching):

1. The most aggressive price always gets matched first.
2. Among orders at the same price, the earliest arrival wins.

```
Order Book for BTC/USD
─────────────────────────────
  Bids (Buy)    │  Asks (Sell)
─────────────────────────────
  10 @ 50200    │   5 @ 50300
   5 @ 50100    │   8 @ 50400
   3 @ 50000    │  12 @ 50500
─────────────────────────────
     Best Bid: 50200
     Best Ask: 50300
     Spread:   100
```

The `BestBid()` and `BestAsk()` methods return the top of each side in O(1) since the slices are always sorted. The spread — the gap between the best bid and best ask — is a key indicator of market liquidity.

### Why Sorted Slices Instead of a Tree?

Many production matching engines use red-black trees or skip lists for O(log n) insertion. We chose sorted slices with `sort.SliceStable` for a reason: clarity. This is an educational project first. The algorithmic trade-off is acceptable for moderate order volumes, and the code is immediately understandable to anyone who reads it.

Upgrading to a tree-based structure later is a localized change — it only affects the `AddOrder` method.

## The Matching Algorithm

When a new order arrives, the engine attempts to match it against the opposite side of the book. The logic is symmetric for buys and sells, so let's trace through a buy order:

```
Incoming: BUY 8 BTC @ MARKET

Ask side of the book:
  s1: SELL 5 @ 50000
  s2: SELL 5 @ 50100

Step 1: Match against s1
  → Trade: 5 BTC @ 50000
  → s1 is fully filled (removed from book)
  → Incoming order remaining: 3

Step 2: Match against s2
  → Trade: 3 BTC @ 50100
  → s2 partially filled (remaining: 2, stays in book)
  → Incoming order remaining: 0 (fully filled)

Result: 2 trades executed
```

The core matching loop is clean:

```go
func (e *Engine) matchBuy(book *orderbook.OrderBook, order *model.Order) []model.Trade {
    var trades []model.Trade

    for !order.IsFilled() {
        bestAsk := book.BestAsk()
        if bestAsk == nil {
            break
        }
        if order.Type == model.Limit && bestAsk.Price > order.Price {
            break
        }
        trade := executeTrade(order, bestAsk, bestAsk.Price)
        trades = append(trades, trade)
    }

    return trades
}
```

Three exit conditions:

1. The incoming order is fully filled.
2. The opposite side of the book is empty.
3. For limit orders, the prices no longer cross (best ask is too expensive).

Market orders skip condition 3 — they consume whatever liquidity is available regardless of price.

### Trade Execution

The `executeTrade` function handles the actual fill:

```go
func executeTrade(buyOrder, sellOrder *model.Order, price float64) model.Trade {
    quantity := min(buyOrder.Remaining, sellOrder.Remaining)
    buyOrder.Remaining -= quantity
    sellOrder.Remaining -= quantity

    return model.Trade{
        BuyOrderID:  buyOrder.ID,
        SellOrderID: sellOrder.ID,
        Price:       price,
        Quantity:    quantity,
        Timestamp:   time.Now(),
    }
}
```

The fill quantity is always the minimum of the two remaining amounts. This naturally handles both full fills and partial fills without any special-case logic.

### Price Determination

The trade price is always the **resting order's price** — the order that was already in the book. This is standard exchange behavior: the passive side (maker) determines the price, and the aggressive side (taker) accepts it.

## Thread Safety

The engine uses a `sync.Mutex` to serialize access to the order books:

```go
type Engine struct {
    mu     sync.Mutex
    books  map[string]*orderbook.OrderBook
    trades []model.Trade
}
```

Every public method (`SubmitOrder`, `CancelOrder`, `GetTrades`, `GetOrderBook`) acquires the lock before accessing shared state. This is a deliberate choice for simplicity — a single lock per engine instance.

For higher throughput, you could:

- Use per-symbol locks (one mutex per order book) to allow parallel matching across different symbols.
- Use a lock-free ring buffer for trade output.
- Shard the engine by symbol across multiple goroutines.

But for correctness and clarity, a single mutex is the right starting point.

## Multi-Symbol Support

The engine maintains a map from symbol to order book:

```go
books map[string]*orderbook.OrderBook
```

Order books are created lazily on first use. Symbols are completely isolated — a trade in `BTC/USD` has zero impact on the `ETH/USD` book. This is verified in the test suite:

```go
func TestMultipleSymbols(t *testing.T) {
    e := New()
    e.SubmitOrder("AAPL", model.NewLimitOrder("a-s1", model.Sell, 150.0, 10))
    e.SubmitOrder("GOOG", model.NewLimitOrder("g-s1", model.Sell, 2800.0, 5))

    trades, _ := e.SubmitOrder("AAPL", model.NewLimitOrder("a-b1", model.Buy, 150.0, 10))
    // 1 AAPL trade, GOOG book unaffected
}
```

## Order Cancellation

Cancellation is straightforward — find the order by ID and remove it:

```go
func (e *Engine) CancelOrder(symbol, orderID string) bool {
    e.mu.Lock()
    defer e.mu.Unlock()

    book, ok := e.books[symbol]
    if !ok {
        return false
    }
    return book.RemoveOrder(orderID)
}
```

The boolean return tells the caller whether the order existed. Cancelling an already-filled or already-cancelled order returns `false`.

## Test Coverage

The test suite covers the critical matching scenarios:

| Test | What It Verifies |
|------|-----------------|
| `TestLimitOrderMatching` | Basic limit order crossing |
| `TestPartialFill` | Partial fills leave remainder in book |
| `TestMarketOrder` | Market orders sweep multiple price levels |
| `TestNoMatchWhenPricesDontCross` | Orders rest in book when spread exists |
| `TestCancelOrder` | Cancel removes order, double-cancel fails |
| `TestPriceTimePriority` | FIFO ordering at same price level |
| `TestMultipleSymbols` | Symbol isolation |
| `TestInvalidOrders` | Nil, negative price, zero quantity rejected |

Run them with:

```bash
go test ./... -v
```

## Project Structure

```
.
├── pkg/
│   ├── model/        # Order and Trade types
│   ├── orderbook/    # Sorted order book with price-time priority
│   └── engine/       # Core matching logic + tests
├── cmd/
│   └── example/      # Runnable demo
├── go.mod
├── LICENSE
└── README.md
```

The separation between `model`, `orderbook`, and `engine` is intentional:

- `model` has zero dependencies — pure data types.
- `orderbook` depends only on `model` — it knows how to store and sort orders, but not how to match them.
- `engine` orchestrates everything — it owns the matching algorithm and coordinates order books.

This layering makes each package independently testable and replaceable.

## What's Next?

This engine is a solid foundation, but a production system would need more:

- **Persistent storage**: Write trades and order state to a database or write-ahead log for crash recovery.
- **Decimal precision**: Replace `float64` with a fixed-point decimal type (like `shopspring/decimal`) to avoid floating-point rounding issues in financial calculations.
- **Order types**: Stop orders, stop-limit orders, fill-or-kill (FOK), immediate-or-cancel (IOC), and good-till-cancelled (GTC) with expiry.
- **Event streaming**: Publish order book updates and trades to a message queue (Kafka, NATS) for downstream consumers.
- **REST/gRPC API**: Expose the engine over the network so clients can submit orders programmatically.
- **Benchmarking**: Add Go benchmarks to measure throughput (orders/second) and latency (time-to-match).
- **Optimized data structures**: Replace sorted slices with a red-black tree or skip list for O(log n) insertion at scale.

Each of these is a natural extension point. The clean package boundaries make it possible to evolve one layer without rewriting the others.

## Running the Demo

```bash
git clone https://github.com/iwtxokhtd83/MatchEngine.git
cd MatchEngine
go run cmd/example/main.go
```

Output:

```
=== Order Matching Engine Demo ===

Placing sell orders...
  SELL 1.0 @ 50000
  SELL 2.0 @ 50100
  SELL 1.5 @ 50200

Placing buy limit order: BUY 0.5 @ 50000...
  TRADE: b1 <-> s1 | 0.5000 @ 50000.00

Placing market buy order: BUY 2.0 (market)...
  TRADE: b2 <-> s1 | 0.5000 @ 50000.00
  TRADE: b2 <-> s2 | 1.5000 @ 50100.00

--- Order Book ---
Bids:
Asks:
  s2: 0.5000 @ 50100.00
  s3: 1.5000 @ 50200.00

Total trades executed: 3
```

## Conclusion

A matching engine doesn't have to be mysterious. At its core, it is a sorted data structure and a loop that compares prices. The complexity in production systems comes from the surrounding infrastructure — networking, persistence, monitoring, and regulatory compliance — not from the matching logic itself.

MatchEngine strips away that complexity to expose the algorithm in its purest form. Fork it, extend it, break it, learn from it. That's what open source is for.

**GitHub**: [https://github.com/iwtxokhtd83/MatchEngine](https://github.com/iwtxokhtd83/MatchEngine)
**License**: MIT
