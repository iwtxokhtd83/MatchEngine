# Matching Algorithm

## Price-Time Priority (FIFO)

MatchEngine uses the most common matching algorithm in financial exchanges: price-time priority.

The rules are:

1. **Price priority**: The most aggressive price is matched first. For buy orders, the highest bid goes first. For sell orders, the lowest ask goes first.
2. **Time priority**: Among orders at the same price level, the order that arrived earliest is matched first (FIFO — First In, First Out).

## The Matching Loop

When a new order arrives, the engine runs a loop against the opposite side of the book:

```
for each iteration:
  1. Is the incoming order fully filled? → stop
  2. Is the opposite side empty? → stop
  3. (Limit only) Do prices no longer cross? → stop
  4. Execute trade at the resting order's price
  5. Reduce remaining quantity on both orders
  6. If resting order is filled, remove it from book
```

### Buy Order Matching (against asks)

```go
for !order.IsFilled() {
    bestAsk := book.BestAsk()
    if bestAsk == nil {
        break // no liquidity
    }
    if order.Type == Limit && bestAsk.Price.GreaterThan(order.Price) {
        break // price doesn't cross
    }
    trade := executeTrade(order, bestAsk, bestAsk.Price)
    trades = append(trades, trade)
}
```

### Sell Order Matching (against bids)

```go
for !order.IsFilled() {
    bestBid := book.BestBid()
    if bestBid == nil {
        break
    }
    if order.Type == Limit && bestBid.Price.LessThan(order.Price) {
        break
    }
    trade := executeTrade(bestBid, order, bestBid.Price)
    trades = append(trades, trade)
}
```

## Trade Execution

Each match produces a trade. The fill quantity is the minimum of the two orders' remaining amounts:

```go
quantity := decimal.Min(buyOrder.Remaining, sellOrder.Remaining)
buyOrder.Remaining = buyOrder.Remaining.Sub(quantity)
sellOrder.Remaining = sellOrder.Remaining.Sub(quantity)
```

This handles three scenarios naturally:

| Scenario | Incoming | Resting | Result |
|----------|----------|---------|--------|
| Full fill (both) | 10 remaining | 10 remaining | Both filled, trade qty = 10 |
| Partial fill (incoming larger) | 10 remaining | 3 remaining | Resting filled, incoming has 7 left |
| Partial fill (resting larger) | 3 remaining | 10 remaining | Incoming filled, resting has 7 left |

## Post-Match Behavior

After matching completes:

- **Limit order with remaining quantity**: Added to the order book on its side.
- **Market order with remaining quantity**: Discarded (market orders never rest).
- **Fully filled orders**: Removed from the book via `RemoveFilled()`.

## Walkthrough Example

Starting state:

```
Asks:
  s1: SELL 5 @ 100
  s2: SELL 3 @ 101
  s3: SELL 10 @ 105

Bids: (empty)
```

Incoming order: `BUY 7 @ 102 (limit)`

**Iteration 1**: Best ask is s1 (5 @ 100). Price crosses (100 <= 102).
→ Trade: 5 @ 100. s1 filled. Incoming remaining: 2.

**Iteration 2**: Best ask is s2 (3 @ 101). Price crosses (101 <= 102).
→ Trade: 2 @ 101. s2 partially filled (remaining: 1). Incoming remaining: 0.

**Iteration 3**: Incoming order is fully filled. Stop.

Result:

```
Trades:
  BUY <-> s1 | 5 @ 100
  BUY <-> s2 | 2 @ 101

Asks:
  s2: SELL 1 @ 101   (partially filled)
  s3: SELL 10 @ 105

Bids: (empty)
```
