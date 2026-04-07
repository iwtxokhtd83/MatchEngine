# Testing

## Running Tests

```bash
go test ./... -v
```

## Test Suite

All tests are in `pkg/engine/engine_test.go`. The suite covers the core matching scenarios:

### TestLimitOrderMatching

Verifies basic limit order crossing. A sell order at 100 is placed, then a buy order at 100 arrives. The engine should produce one trade at price 100 with the buy order's quantity.

### TestPartialFill

A sell order for 3 units is placed. A buy order for 10 units arrives at the same price. The engine should:
- Produce one trade for 3 units
- Leave the buy order in the book with 7 remaining

### TestMarketOrder

Two sell limit orders at different prices (100 and 101) are placed. A market buy for 8 units arrives. The engine should:
- Fill 5 units at 100 (first price level)
- Fill 3 units at 101 (second price level)
- Produce 2 trades total

### TestNoMatchWhenPricesDontCross

A sell at 200 and a buy at 199 are placed. No trade should occur. Both orders should rest in the book.

### TestCancelOrder

A sell order is placed and then cancelled. Verifies:
- First cancel returns `true`
- Second cancel returns `false` (already removed)
- Order book is empty after cancellation

### TestPriceTimePriority

Two sell orders at the same price are placed sequentially. A buy order arrives that can only fill one. The engine should match against the first sell order (FIFO).

### TestMultipleSymbols

Orders are placed on two different symbols (AAPL and GOOG). A trade on AAPL should not affect the GOOG order book.

### TestInvalidOrders

Verifies that the engine rejects:
- `nil` orders
- Limit orders with negative price
- Orders with zero quantity

## Adding New Tests

Follow the existing pattern:

```go
func TestYourScenario(t *testing.T) {
    e := New()

    // Set up the book
    e.SubmitOrder("SYM", model.NewLimitOrder("s1", model.Sell, 100.0, 10))

    // Submit the order under test
    trades, err := e.SubmitOrder("SYM", model.NewLimitOrder("b1", model.Buy, 100.0, 5))

    // Assert results
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(trades) != 1 {
        t.Fatalf("expected 1 trade, got %d", len(trades))
    }
}
```

## Benchmarking (Future)

Benchmark tests can be added using Go's built-in benchmarking:

```go
func BenchmarkMatchingEngine(b *testing.B) {
    e := New()
    for i := 0; i < b.N; i++ {
        id := fmt.Sprintf("order-%d", i)
        if i%2 == 0 {
            e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Sell, 100.0, 1))
        } else {
            e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Buy, 100.0, 1))
        }
    }
}
```

Run with:

```bash
go test -bench=. -benchmem ./pkg/engine/
```
