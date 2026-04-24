# Order Types

MatchEngine supports four order types and three time-in-force policies.

## Order Types

### Limit Order

Specifies a maximum buy price or minimum sell price. Rests in the book if not immediately filled (with GTC time-in-force).

```go
req := model.NewLimitOrderRequest(model.Buy, decimal.NewFromInt(100), decimal.NewFromInt(10))
```

### Market Order

No price constraint. Executes immediately at the best available prices. Never rests in the book.

```go
req := model.NewMarketOrderRequest(model.Buy, decimal.NewFromInt(10))
```

### Stop-Market Order

A dormant order that becomes a market order when the last trade price reaches the stop price.

- **Buy stop**: triggers when last price >= stop price (used for breakout entries or stop-loss on shorts)
- **Sell stop**: triggers when last price <= stop price (used for stop-loss on longs)

```go
req := model.NewStopMarketRequest(model.Buy, decimal.NewFromInt(105), decimal.NewFromInt(5))
```

### Stop-Limit Order

Like stop-market, but converts to a limit order instead of a market order when triggered.

```go
req := model.NewStopLimitRequest(model.Buy, decimal.NewFromInt(105), decimal.NewFromInt(106), decimal.NewFromInt(5))
// Triggers at 105, places limit buy at 106
```

## Time-in-Force (TIF)

TIF controls how long an order remains active. Applies to limit orders.

| TIF | Behavior |
|-----|----------|
| **GTC** (Good-Till-Cancelled) | Rests in book until filled or cancelled. Default. |
| **IOC** (Immediate-or-Cancel) | Fills as much as possible immediately, cancels the rest. Never rests. |
| **FOK** (Fill-or-Kill) | Must be filled entirely immediately, or is cancelled completely. All-or-nothing. |

### IOC Example

```go
req := model.NewIOCOrderRequest(model.Buy, decimal.NewFromInt(100), decimal.NewFromInt(10))
id, trades, _ := e.SubmitRequest("BTC/USD", req)
// Fills what's available, discards the rest
```

### FOK Example

```go
req := model.NewFOKOrderRequest(model.Buy, decimal.NewFromInt(100), decimal.NewFromInt(10))
id, trades, _ := e.SubmitRequest("BTC/USD", req)
// Either fills all 10 or returns 0 trades
```

## Price Determination

The trade price is always the resting order's price (maker sets the price, taker accepts it).

## Order Fields

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique order identifier |
| `OwnerID` | `string` | Trader identifier (for STP) |
| `Side` | `Side` | `Buy` or `Sell` |
| `Type` | `OrderType` | `Limit`, `Market`, `StopMarket`, `StopLimit` |
| `TIF` | `TimeInForce` | `GTC`, `IOC`, `FOK` |
| `Price` | `decimal.Decimal` | Limit price |
| `StopPrice` | `decimal.Decimal` | Trigger price for stop orders |
| `Quantity` | `decimal.Decimal` | Original order size |
| `Remaining` | `decimal.Decimal` | Unfilled quantity |
| `Timestamp` | `time.Time` | Order creation time |
