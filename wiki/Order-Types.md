# Order Types

MatchEngine supports two order types: Limit and Market.

## Limit Order

A limit order specifies the maximum price a buyer is willing to pay, or the minimum price a seller is willing to accept.

```go
order := model.NewLimitOrder("order-1", model.Buy, decimal.NewFromInt(100), decimal.NewFromInt(10))
```

Behavior:

- If a matching order exists on the opposite side at a compatible price, the order is filled immediately (fully or partially).
- If no match is available, the order rests in the order book until a future order matches it or it is cancelled.
- A buy limit order matches if `best ask price <= order price`.
- A sell limit order matches if `best bid price >= order price`.

### Example: Limit Order Resting in Book

```
Book state: Ask side empty

Submit: BUY 10 @ 100 (limit)
→ No asks available, order rests in book

Book state:
  Bids: 10 @ 100
  Asks: (empty)
```

### Example: Limit Order Matching Immediately

```
Book state:
  Asks: SELL 5 @ 99

Submit: BUY 10 @ 100 (limit)
→ Trade: 5 @ 99 (resting order's price)
→ Remaining 5 rests in book as a bid

Book state:
  Bids: 5 @ 100
  Asks: (empty)
```

## Market Order

A market order has no price constraint. It executes immediately at the best available prices, sweeping through multiple price levels if necessary.

```go
order := model.NewMarketOrder("order-2", model.Buy, decimal.NewFromInt(10))
```

Behavior:

- Matches against the opposite side starting from the best price.
- Continues matching at progressively worse prices until the order is fully filled or the opposite side is exhausted.
- Never rests in the order book. Any unfilled quantity is discarded.

### Example: Market Order Sweeping Multiple Levels

```
Book state:
  Asks: SELL 5 @ 100, SELL 5 @ 101

Submit: BUY 8 (market)
→ Trade 1: 5 @ 100
→ Trade 2: 3 @ 101
→ Order fully filled

Book state:
  Asks: SELL 2 @ 101
```

## Price Determination

The trade price is always the resting order's price (the order already in the book). This follows standard exchange convention:

- The **maker** (passive, resting order) sets the price.
- The **taker** (aggressive, incoming order) accepts the price.

## Order Fields

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique order identifier |
| `Side` | `Side` | `Buy` or `Sell` |
| `Type` | `OrderType` | `Limit` or `Market` |
| `Price` | `decimal.Decimal` | Limit price (ignored for market orders) |
| `Quantity` | `decimal.Decimal` | Original order size |
| `Remaining` | `decimal.Decimal` | Unfilled quantity (decreases on partial fills) |
| `Timestamp` | `time.Time` | Order creation time (used for FIFO priority) |
