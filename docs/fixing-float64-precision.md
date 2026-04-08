# Why We Replaced float64 With Decimal in Our Matching Engine

Every financial system eventually faces the same question: can we trust floating-point arithmetic with real money? For [MatchEngine](https://github.com/iwtxokhtd83/MatchEngine), an open-source order matching engine written in Go, the answer was no. This article explains the problem, walks through the migration from `float64` to `shopspring/decimal`, and shows the specific code patterns that changed.

## The Problem: float64 Cannot Represent Money

IEEE 754 double-precision floating-point (`float64` in Go) stores numbers in binary. Most decimal fractions — the kind humans use for prices — have no exact binary representation.

The classic example:

```go
fmt.Println(0.1 + 0.2 == 0.3) // false
fmt.Println(0.1 + 0.2)         // 0.30000000000000004
```

This is not a Go bug. It is how binary floating-point works in every language. The error is tiny — about `5.5e-17` — but in a matching engine, tiny errors compound.

### How This Breaks a Matching Engine

Consider a sell order for `0.1` BTC and another for `0.2` BTC, both at price `0.3`. A buy order arrives for `0.3` BTC at price `0.3`.

With `float64`:

```
Trade 1: fill 0.1, remaining = 0.3 - 0.1 = 0.19999999999999998
Trade 2: fill 0.2, remaining = 0.19999999999999998 - 0.2 = -2.7755575615628914e-17
```

The remaining quantity is `-2.78e-17` instead of `0`. The `IsFilled()` check was:

```go
func (o *Order) IsFilled() bool {
    return o.Remaining <= 0
}
```

This happens to work here because the result is slightly negative. But flip the order of operations and you might get `+2.78e-17` instead — a positive dust amount that makes `IsFilled()` return `false`. The order stays in the book with an unfillable phantom quantity.

Other failure modes:

- Two orders at the "same" price might not match because `100.0` stored through different arithmetic paths produces `99.99999999999999` on one side.
- Accumulated rounding across thousands of partial fills drifts the book's total quantity away from reality.
- Price comparison in sorting (`ob.Bids[i].Price == ob.Bids[j].Price`) fails for prices that should be equal, breaking FIFO ordering.

## The Fix: shopspring/decimal

We replaced every `float64` price and quantity field with [`shopspring/decimal`](https://github.com/shopspring/decimal), an arbitrary-precision decimal library for Go.

```go
import "github.com/shopspring/decimal"
```

`decimal.Decimal` stores numbers as a coefficient and a base-10 exponent internally. `0.1` is stored as `1 * 10^-1` — exact, no binary approximation.

```go
a := decimal.RequireFromString("0.1")
b := decimal.RequireFromString("0.2")
c := decimal.RequireFromString("0.3")

fmt.Println(a.Add(b).Equal(c)) // true
```

## What Changed: A File-by-File Walkthrough

### 1. Model Layer — Order and Trade Structs

Before:

```go
type Order struct {
    ID        string
    Side      Side
    Type      OrderType
    Price     float64
    Quantity  float64
    Remaining float64
    Timestamp time.Time
}
```

After:

```go
type Order struct {
    ID        string
    Side      Side
    Type      OrderType
    Price     decimal.Decimal
    Quantity  decimal.Decimal
    Remaining decimal.Decimal
    Timestamp time.Time
}
```

The same change applies to `Trade`:

```go
type Trade struct {
    BuyOrderID  string
    SellOrderID string
    Price       decimal.Decimal
    Quantity    decimal.Decimal
    Timestamp   time.Time
}
```

Constructor signatures changed accordingly:

```go
// Before
func NewLimitOrder(id string, side Side, price, quantity float64) *Order

// After
func NewLimitOrder(id string, side Side, price, quantity decimal.Decimal) *Order
```

### 2. IsFilled — The Most Critical One-Liner

Before:

```go
func (o *Order) IsFilled() bool {
    return o.Remaining <= 0
}
```

After:

```go
func (o *Order) IsFilled() bool {
    return o.Remaining.LessThanOrEqual(decimal.Zero)
}
```

With `float64`, `<=` comparison on a value like `2.78e-17` would incorrectly return `false`. With `decimal`, after subtracting `0.1` and `0.2` from `0.3`, the remaining is exactly `0` — no dust, no ambiguity.

### 3. Order Book — Price Comparisons in Sorting

The order book sorts bids and asks by price-time priority. Every price comparison changed from operators to method calls.

Before:

```go
sort.SliceStable(ob.Bids, func(i, j int) bool {
    if ob.Bids[i].Price == ob.Bids[j].Price {
        return ob.Bids[i].Timestamp.Before(ob.Bids[j].Timestamp)
    }
    return ob.Bids[i].Price > ob.Bids[j].Price
})
```

After:

```go
sort.SliceStable(ob.Bids, func(i, j int) bool {
    if ob.Bids[i].Price.Equal(ob.Bids[j].Price) {
        return ob.Bids[i].Timestamp.Before(ob.Bids[j].Timestamp)
    }
    return ob.Bids[i].Price.GreaterThan(ob.Bids[j].Price)
})
```

The `==` operator on `float64` is the most dangerous comparison in financial code. Two prices that should be equal — say, both constructed from the string `"100.05"` — might differ by an epsilon if they arrived through different arithmetic paths. `decimal.Equal()` compares the actual decimal value, not a binary approximation.

The `Spread()` method also changed:

```go
// Before
func (ob *OrderBook) Spread() float64 {
    return ask.Price - bid.Price
}

// After
func (ob *OrderBook) Spread() decimal.Decimal {
    return ask.Price.Sub(bid.Price)
}
```

### 4. Matching Engine — Trade Execution

The core matching logic changed from arithmetic operators to decimal methods.

Before:

```go
func executeTrade(buyOrder, sellOrder *model.Order, price float64) model.Trade {
    quantity := min(buyOrder.Remaining, sellOrder.Remaining)
    buyOrder.Remaining -= quantity
    sellOrder.Remaining -= quantity
    // ...
}

func min(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}
```

After:

```go
func executeTrade(buyOrder, sellOrder *model.Order, price decimal.Decimal) model.Trade {
    quantity := decimal.Min(buyOrder.Remaining, sellOrder.Remaining)
    buyOrder.Remaining = buyOrder.Remaining.Sub(quantity)
    sellOrder.Remaining = sellOrder.Remaining.Sub(quantity)
    // ...
}
```

The custom `min()` function is gone — `decimal.Min()` handles it. The `-=` operator is replaced by `.Sub()`, which returns a new `decimal.Decimal` value (the type is immutable).

Validation also changed:

```go
// Before
if order.Remaining <= 0 { ... }
if order.Price <= 0 { ... }

// After
if order.Remaining.LessThanOrEqual(decimal.Zero) { ... }
if order.Price.LessThanOrEqual(decimal.Zero) { ... }
```

### 5. Price Crossing Checks

The matching loop's price crossing logic changed from `>` and `<` to method calls:

```go
// Before: stop if ask price is higher than bid price
if order.Type == model.Limit && bestAsk.Price > order.Price {
    break
}

// After
if order.Type == model.Limit && bestAsk.Price.GreaterThan(order.Price) {
    break
}
```

Same pattern for sell-side matching:

```go
// Before
if order.Type == model.Limit && bestBid.Price < order.Price {
    break
}

// After
if order.Type == model.Limit && bestBid.Price.LessThan(order.Price) {
    break
}
```

## The Proof: TestDecimalPrecision

We added a test that specifically targets the `0.1 + 0.2 = 0.3` problem:

```go
func TestDecimalPrecision(t *testing.T) {
    e := New()

    // Two sells: 0.1 and 0.2, both at price 0.3
    e.SubmitOrder("PREC", model.NewLimitOrder("s1", model.Sell, d("0.3"), d("0.1")))
    e.SubmitOrder("PREC", model.NewLimitOrder("s2", model.Sell, d("0.3"), d("0.2")))

    // Buy exactly 0.3
    buy := model.NewLimitOrder("b1", model.Buy, d("0.3"), d("0.3"))
    trades, _ := e.SubmitOrder("PREC", buy)

    // Total filled quantity must be exactly 0.3
    totalQty := trades[0].Quantity.Add(trades[1].Quantity)
    if !totalQty.Equal(d("0.3")) {
        t.Errorf("expected total quantity 0.3, got %s", totalQty)
    }

    // No leftover dust — order is exactly filled
    if !buy.IsFilled() {
        t.Errorf("expected buy order to be fully filled, remaining: %s", buy.Remaining)
    }

    // Book should be completely empty
    book := e.GetOrderBook("PREC")
    bids, asks := book.Depth()
    if bids != 0 || asks != 0 {
        t.Errorf("expected empty book, got %d bids / %d asks", bids, asks)
    }
}
```

With `float64`, this test would fail — the buy order would have a non-zero remaining amount after the two fills. With `decimal`, `0.1 + 0.2` is exactly `0.3`, the order is exactly filled, and the book is clean.

## Migration Cheat Sheet

Here is a quick reference for the operator-to-method mapping:

| float64 | decimal.Decimal |
|---------|----------------|
| `a + b` | `a.Add(b)` |
| `a - b` | `a.Sub(b)` |
| `a * b` | `a.Mul(b)` |
| `a / b` | `a.Div(b)` |
| `a == b` | `a.Equal(b)` |
| `a > b` | `a.GreaterThan(b)` |
| `a < b` | `a.LessThan(b)` |
| `a >= b` | `a.GreaterThanOrEqual(b)` |
| `a <= b` | `a.LessThanOrEqual(b)` |
| `min(a, b)` | `decimal.Min(a, b)` |
| `fmt.Sprintf("%.2f", a)` | `a.StringFixed(2)` |

Constructing values:

```go
decimal.NewFromInt(100)              // from integer
decimal.NewFromString("99.95")       // from string (returns value, error)
decimal.RequireFromString("99.95")   // from string (panics on error)
decimal.NewFromFloat(99.95)          // from float64 (use sparingly — inherits float imprecision)
decimal.Zero                         // the zero value
```

Prefer `NewFromString` or `NewFromInt` over `NewFromFloat`. The whole point of this migration is to avoid float64 representation — using `NewFromFloat` reintroduces the problem at the boundary.

## Trade-offs

Nothing is free. Here is what we gained and what it cost:

| Aspect | float64 | decimal.Decimal |
|--------|---------|----------------|
| Precision | ~15-17 significant digits, binary rounding | Arbitrary precision, exact decimal |
| Performance | Hardware-native, ~1ns per op | Software-implemented, ~10-50x slower |
| Memory | 8 bytes | ~72 bytes per value |
| API ergonomics | Native operators (`+`, `-`, `==`) | Method calls (`.Add()`, `.Sub()`, `.Equal()`) |
| Correctness for money | Fundamentally broken | Correct by design |

For a matching engine, correctness is non-negotiable. The performance overhead of decimal arithmetic is negligible compared to the cost of network I/O, serialization, and persistence that a production system would have. If profiling later shows decimal as a bottleneck, the alternative is fixed-point integer arithmetic (storing prices in the smallest unit, like satoshis or cents) — but that adds complexity around unit conversion and overflow handling.

## Conclusion

The migration touched every file that handles prices or quantities — 5 source files and 1 test file. The diff is mechanical: replace types, replace operators with methods, replace constructors. No algorithmic changes were needed. The matching logic, the sorting, the order book structure — all stayed the same. Only the numeric representation changed.

That is the key insight: fixing floating-point precision in a financial system is not an architectural change. It is a type change. The earlier you make it, the smaller the diff.

**GitHub**: [https://github.com/iwtxokhtd83/MatchEngine](https://github.com/iwtxokhtd83/MatchEngine)
**Commit**: `fix: replace float64 with shopspring/decimal for exact financial arithmetic`
**Issue**: [#1 — float64 precision causes incorrect trade amounts](https://github.com/iwtxokhtd83/MatchEngine/issues/1)
