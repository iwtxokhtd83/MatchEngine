# Four Bugs That Would Break a Matching Engine in Production

[MatchEngine](https://github.com/iwtxokhtd83/MatchEngine) is an open-source order matching engine written in Go. After the initial release, we opened issues for every defect we could find through code review. This article covers the four bugs we fixed in v0.2.0, why each one matters, and the exact code changes that resolved them.

These are not exotic edge cases. They are the kind of bugs that survive code review, pass unit tests, and then blow up under real traffic.

## Bug #1: Duplicate Order IDs Corrupt the Book

### The Problem

The engine accepted any order ID provided by the caller. There was no uniqueness check. Submitting two orders with the same ID caused both to exist in the book:

```go
e.SubmitOrder("BTC", model.NewLimitOrder("dup", model.Sell, d("100"), d("5")))
e.SubmitOrder("BTC", model.NewLimitOrder("dup", model.Sell, d("200"), d("10")))
// Both orders now live in the book with ID "dup"
```

When `CancelOrder("BTC", "dup")` was called, the old `RemoveOrder` did a linear scan and stopped at the first match. The second order became an orphan — still in the book, still matchable, but invisible to cancellation.

Worse, if a downstream system used order IDs as keys (for position tracking, risk management, or reconciliation), the duplicate would silently corrupt its state.

### The Fix

We added an order index to the engine — a `map[string]string` mapping order ID to symbol:

```go
type Engine struct {
    mu         sync.Mutex
    books      map[string]*orderbook.OrderBook
    orderIndex map[string]string  // order ID -> symbol
    // ...
}
```

On every `SubmitOrder`, the engine checks the index before proceeding:

```go
if _, exists := e.orderIndex[order.ID]; exists {
    return nil, fmt.Errorf("duplicate order ID: %s", order.ID)
}
```

When an order enters the book, it is registered in the index. When it is filled or cancelled, it is removed:

```go
// On add to book
e.orderIndex[order.ID] = symbol

// On cancel
if book.RemoveOrder(orderID) {
    delete(e.orderIndex, orderID)
    return true
}

// On fill (during cleanup)
if o.IsFilled() {
    delete(e.orderIndex, o.ID)
}
```

This means order IDs can be reused after an order is fully filled or cancelled — a deliberate design choice. The ID lifecycle is: available → active (in book) → available (after fill or cancel).

We also upgraded the order book itself to maintain an internal `orders map[string]*model.Order` index, which gives `RemoveOrder` O(1) lookup instead of the previous O(n) linear scan:

```go
type OrderBook struct {
    Bids   []*model.Order
    Asks   []*model.Order
    orders map[string]*model.Order  // O(1) lookup by ID
}

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
```

Because we know which side the order is on from the map lookup, we only scan one slice instead of both. The map also eliminates the ambiguity of the old code — there is exactly one entry per ID, always.

### The Tests

```go
func TestDuplicateOrderID(t *testing.T) {
    e := New()
    e.SubmitOrder("BTC", model.NewLimitOrder("dup", model.Sell, d("100"), d("5")))

    _, err := e.SubmitOrder("BTC", model.NewLimitOrder("dup", model.Sell, d("200"), d("10")))
    if err == nil {
        t.Error("expected error for duplicate order ID")
    }
}

func TestDuplicateOrderIDAfterFill(t *testing.T) {
    e := New()
    e.SubmitOrder("BTC", model.NewLimitOrder("reuse", model.Sell, d("100"), d("5")))
    e.SubmitOrder("BTC", model.NewLimitOrder("buyer", model.Buy, d("100"), d("5")))

    // Filled ID should be reusable
    _, err := e.SubmitOrder("BTC", model.NewLimitOrder("reuse", model.Sell, d("100"), d("3")))
    if err != nil {
        t.Errorf("expected reuse to succeed, got: %v", err)
    }
}
```

---

## Bug #2: GetOrderBook Leaks Internal State

### The Problem

`GetOrderBook` returned a direct pointer to the engine's internal order book:

```go
func (e *Engine) GetOrderBook(symbol string) *orderbook.OrderBook {
    e.mu.Lock()
    defer e.mu.Unlock()
    return e.books[symbol]  // returns the real thing
}
```

The mutex is released as soon as the method returns. The caller now holds a raw pointer to mutable shared state with no lock protection. Any read from another goroutine races with writes from `SubmitOrder`:

```go
// Goroutine 1: reading the book
book := e.GetOrderBook("BTC")
for _, ask := range book.Asks {  // iterating over a slice that may be
    fmt.Println(ask.Price)        // modified concurrently
}

// Goroutine 2: submitting an order (modifies book.Asks)
e.SubmitOrder("BTC", someOrder)
```

This is a textbook data race. Under Go's race detector (`go test -race`), this would be flagged immediately. In production without the race detector, it manifests as corrupted reads, panics from slice index out of range, or silently wrong data.

### The Fix

`GetOrderBook` now returns a deep-copy snapshot:

```go
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
```

The `Snapshot` method on `OrderBook` creates a completely independent copy:

```go
func (ob *OrderBook) Snapshot() *OrderBook {
    snap := &OrderBook{
        Bids:   make([]*model.Order, len(ob.Bids)),
        Asks:   make([]*model.Order, len(ob.Asks)),
        orders: make(map[string]*model.Order),
    }
    for i, o := range ob.Bids {
        cp := *o          // copy the Order struct by value
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
```

Key detail: we copy each `*model.Order` by dereferencing the pointer (`cp := *o`), which creates a new `Order` value on the stack, then take its address. This ensures the snapshot's orders are completely independent from the engine's orders. Mutating `snap.Asks[0].Remaining` does not touch the engine's internal state.

### The Tests

```go
func TestGetOrderBookReturnsSnapshot(t *testing.T) {
    e := New()
    e.SubmitOrder("SNAP", model.NewLimitOrder("s1", model.Sell, d("100"), d("10")))

    snap := e.GetOrderBook("SNAP")
    snap.Asks[0].Remaining = d("999")  // mutate the snapshot

    snap2 := e.GetOrderBook("SNAP")
    if snap2.Asks[0].Remaining.Equal(d("999")) {
        t.Error("snapshot mutation affected internal book")
    }
}

func TestGetOrderBookConcurrentAccess(t *testing.T) {
    e := New()
    e.SubmitOrder("CONC", model.NewLimitOrder("s1", model.Sell, d("100"), d("10")))

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            book := e.GetOrderBook("CONC")
            _ = book.Depth()
            _ = book.Spread()
        }()
    }
    wg.Wait()
}
```

The concurrent test spawns 100 goroutines all reading the order book simultaneously. With the old code, this would trigger the race detector. With snapshots, each goroutine gets its own independent copy.

---

## Bug #3: Unbounded Trade Log Leaks Memory

### The Problem

Every executed trade was appended to `e.trades` and never removed:

```go
e.trades = append(e.trades, trades...)
```

`GetTrades()` copied the entire slice on every call:

```go
func (e *Engine) GetTrades() []model.Trade {
    e.mu.Lock()
    defer e.mu.Unlock()
    result := make([]model.Trade, len(e.trades))
    copy(result, e.trades)
    return result
}
```

For a long-running engine processing thousands of trades per second, this is a memory leak. After a million trades, the slice holds a million `Trade` structs in memory permanently. `GetTrades()` allocates another million-element slice on every call. Eventually, the process runs out of memory.

### The Fix

We introduced two mechanisms: a bounded in-memory log with eviction, and a callback for real-time trade processing.

The engine now accepts functional options:

```go
e := engine.New(
    engine.WithMaxTradeLog(5000),
    engine.WithTradeHandler(func(symbol string, trade model.Trade) {
        // persist to database, publish to Kafka, etc.
    }),
)
```

The trade recording logic:

```go
for _, t := range trades {
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
```

Three modes of operation:

| Configuration | Behavior |
|--------------|----------|
| Default (`maxTrades=10000`) | Keeps last ~10,000 trades, evicts oldest 10% when full |
| `WithMaxTradeLog(0)` | No in-memory log at all. Use `WithTradeHandler` for processing. |
| `WithTradeHandler(fn)` | Callback invoked for every trade, regardless of log setting |

The eviction strategy removes 10% at a time rather than one-by-one. This amortizes the cost of slice shifting — removing one element from the front of a 10,000-element slice requires shifting 9,999 elements. Removing 1,000 at once means we only pay that cost every 1,000 trades instead of every trade.

The callback runs inside the mutex, which is intentional. It guarantees that the handler sees trades in order and that no concurrent `SubmitOrder` can interleave. If the handler needs to do slow I/O, it should buffer internally and process asynchronously.

### The Tests

```go
func TestTradeLogBounded(t *testing.T) {
    e := New(WithMaxTradeLog(10))

    for i := 0; i < 20; i++ {
        sid := fmt.Sprintf("s%d", i)
        bid := fmt.Sprintf("b%d", i)
        e.SubmitOrder("BND", model.NewLimitOrder(sid, model.Sell, d("100"), d("1")))
        e.SubmitOrder("BND", model.NewLimitOrder(bid, model.Buy, d("100"), d("1")))
    }

    trades := e.GetTrades()
    if len(trades) > 10 {
        t.Errorf("expected at most 10 trades, got %d", len(trades))
    }
}

func TestTradeLogDisabled(t *testing.T) {
    e := New(WithMaxTradeLog(0))
    e.SubmitOrder("DIS", model.NewLimitOrder("s1", model.Sell, d("100"), d("1")))
    e.SubmitOrder("DIS", model.NewLimitOrder("b1", model.Buy, d("100"), d("1")))

    if len(e.GetTrades()) != 0 {
        t.Error("expected empty trade log when disabled")
    }
}

func TestTradeHandler(t *testing.T) {
    var received []model.Trade
    e := New(WithTradeHandler(func(symbol string, trade model.Trade) {
        received = append(received, trade)
    }))

    e.SubmitOrder("CB", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
    e.SubmitOrder("CB", model.NewLimitOrder("b1", model.Buy, d("100"), d("3")))

    if len(received) != 1 || !received[0].Quantity.Equal(d("3")) {
        t.Error("trade handler did not receive expected trade")
    }
}
```

---

## Bug #4: No Symbol Validation

### The Problem

The engine accepted any string as a symbol, including empty strings, whitespace, and mixed-case variants:

```go
e.SubmitOrder("", order)          // creates a book keyed by ""
e.SubmitOrder("   ", order)       // creates a book keyed by "   "
e.SubmitOrder("btc/usd", order)   // different book from "BTC/USD"
e.SubmitOrder("BTC/USD", order)   // different book from "btc/usd"
```

A user typing `"btc/usd"` and `"BTC/USD"` would unknowingly place orders in two separate books. They would never match against each other. No error, no warning — just silent failure.

### The Fix

Three layers of defense:

**1. Normalization**: Every symbol is uppercased and trimmed before use.

```go
func normalizeSymbol(s string) string {
    return strings.ToUpper(strings.TrimSpace(s))
}
```

This is applied in `SubmitOrder`, `CancelOrder`, and `GetOrderBook`. The caller can pass `"btc/usd"`, `" BTC/USD "`, or `"BTC/USD"` — they all resolve to the same book.

**2. Empty rejection**: After normalization, empty strings are rejected.

```go
func (e *Engine) validateSymbol(symbol string) error {
    if symbol == "" {
        return fmt.Errorf("symbol cannot be empty")
    }
    if e.symbols != nil && !e.symbols[symbol] {
        return fmt.Errorf("symbol %q is not registered", symbol)
    }
    return nil
}
```

**3. Optional registration**: For strict environments, symbols can be pre-registered.

```go
e := engine.New()
e.RegisterSymbol("BTC/USD")
e.RegisterSymbol("ETH/USD")

// This works
e.SubmitOrder("BTC/USD", order)

// This returns an error: symbol "DOGE/USD" is not registered
e.SubmitOrder("DOGE/USD", order)
```

Registration is opt-in. If no symbols are registered, the engine accepts any non-empty symbol (the default behavior for development and testing). Once the first symbol is registered, strict mode activates.

### The Tests

```go
func TestEmptySymbolRejected(t *testing.T) {
    e := New()
    _, err := e.SubmitOrder("", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
    if err == nil {
        t.Error("expected error for empty symbol")
    }
    _, err = e.SubmitOrder("   ", model.NewLimitOrder("s2", model.Sell, d("100"), d("5")))
    if err == nil {
        t.Error("expected error for whitespace-only symbol")
    }
}

func TestSymbolNormalization(t *testing.T) {
    e := New()
    e.SubmitOrder("btc/usd", model.NewLimitOrder("s1", model.Sell, d("100"), d("5")))
    trades, _ := e.SubmitOrder("BTC/USD", model.NewLimitOrder("b1", model.Buy, d("100"), d("3")))
    if len(trades) != 1 {
        t.Fatal("expected match across normalized symbols")
    }
}

func TestRegisteredSymbolsOnly(t *testing.T) {
    e := New()
    e.RegisterSymbol("BTC/USD")

    _, err := e.SubmitOrder("DOGE/USD", model.NewLimitOrder("s1", model.Sell, d("1"), d("100")))
    if err == nil {
        t.Error("expected error for unregistered symbol")
    }
}
```

---

## The Pattern

Looking at these four bugs together, a pattern emerges. Each one is a boundary problem — a place where the engine's internal state meets the outside world:

| Bug | Boundary |
|-----|----------|
| Duplicate IDs | Caller-provided identifiers entering the engine |
| Leaked pointer | Internal state leaving the engine |
| Unbounded log | Time (state accumulating without limit) |
| No validation | Caller-provided symbols entering the engine |

The fixes follow a consistent strategy: validate at the boundary, copy across the boundary, and bound what accumulates inside.

These are not matching-engine-specific lessons. They apply to any stateful system that accepts external input and exposes internal state. The matching algorithm itself — the price-time priority loop — was never the problem. The problems were all in the plumbing around it.

## Summary

| Issue | Problem | Fix | Tests Added |
|-------|---------|-----|-------------|
| #2 | Duplicate order IDs | Order index map + rejection | 3 |
| #3 | GetOrderBook leaks pointer | Deep-copy snapshot | 2 |
| #5 | Unbounded trade log | Bounded log + callback | 3 |
| #10 | No symbol validation | Normalize + validate + optional registration | 3 |

Total: 11 new tests, 0 changes to the matching algorithm.

**GitHub**: [https://github.com/iwtxokhtd83/MatchEngine](https://github.com/iwtxokhtd83/MatchEngine)
**Release**: [v0.2.0](https://github.com/iwtxokhtd83/MatchEngine/releases/tag/v0.2.0)
