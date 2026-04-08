# Testing

## Running Tests

```bash
go test ./... -v
```

## Test Suite

All tests are in `pkg/engine/engine_test.go`.

### Core Matching Tests

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
| `TestDecimalPrecision` | 0.1 + 0.2 = 0.3 exactness with decimal |

### Bug Fix Tests

| Test | Issue | What It Verifies |
|------|-------|-----------------|
| `TestDuplicateOrderID` | #2 | Duplicate order IDs are rejected |
| `TestDuplicateOrderIDAfterFill` | #2 | Filled order IDs can be reused |
| `TestDuplicateOrderIDAfterCancel` | #2 | Cancelled order IDs can be reused |
| `TestGetOrderBookReturnsSnapshot` | #3 | Snapshot mutation doesn't affect engine |
| `TestGetOrderBookConcurrentAccess` | #3 | 100 concurrent goroutines read safely |
| `TestTradeLogBounded` | #5 | Trade log stays within configured limit |
| `TestTradeLogDisabled` | #5 | Log disabled with `WithMaxTradeLog(0)` |
| `TestTradeHandler` | #5 | Trade callback receives executed trades |
| `TestEmptySymbolRejected` | #10 | Empty and whitespace symbols rejected |
| `TestSymbolNormalization` | #10 | Case-insensitive symbol matching |
| `TestRegisteredSymbolsOnly` | #10 | Unregistered symbols rejected |

## Adding New Tests

```go
func TestYourScenario(t *testing.T) {
    e := New()

    e.SubmitOrder("SYM", model.NewLimitOrder("s1", model.Sell, d("100"), d("10")))

    trades, err := e.SubmitOrder("SYM", model.NewLimitOrder("b1", model.Buy, d("100"), d("5")))

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(trades) != 1 {
        t.Fatalf("expected 1 trade, got %d", len(trades))
    }
}
```

## Benchmarking (Future)

```go
func BenchmarkMatchingEngine(b *testing.B) {
    e := New()
    for i := 0; i < b.N; i++ {
        id := fmt.Sprintf("order-%d", i)
        if i%2 == 0 {
            e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Sell, d("100"), d("1")))
        } else {
            e.SubmitOrder("BTC", model.NewLimitOrder(id, model.Buy, d("100"), d("1")))
        }
    }
}
```

```bash
go test -bench=. -benchmem ./pkg/engine/
```
