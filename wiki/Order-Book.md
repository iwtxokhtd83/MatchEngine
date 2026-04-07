# Order Book

The order book is the central data structure that holds all resting orders for a given trading symbol.

## Structure

```go
type OrderBook struct {
    Bids []*model.Order // buy orders: highest price first
    Asks []*model.Order // sell orders: lowest price first
}
```

Each side is a sorted slice of order pointers.

## Sorting Rules

### Bids (Buy Side)

Sorted by:
1. Price **descending** (highest price first — most aggressive buyer at the top)
2. Timestamp **ascending** (earliest order first at the same price — FIFO)

```
Bids:
  10 @ 105  (arrived 10:00:01)
  5  @ 105  (arrived 10:00:03)   ← same price, later arrival
  8  @ 100  (arrived 10:00:00)
  3  @ 99   (arrived 10:00:02)
```

### Asks (Sell Side)

Sorted by:
1. Price **ascending** (lowest price first — most aggressive seller at the top)
2. Timestamp **ascending** (earliest order first at the same price — FIFO)

```
Asks:
  5  @ 106  (arrived 10:00:00)
  2  @ 106  (arrived 10:00:04)   ← same price, later arrival
  10 @ 110  (arrived 10:00:01)
  7  @ 115  (arrived 10:00:02)
```

## Operations

### AddOrder

Inserts an order into the correct side and re-sorts the slice using `sort.SliceStable` to preserve time ordering among equal prices.

Time complexity: O(n log n) due to sorting. Acceptable for moderate volumes; can be optimized with a tree structure for high-frequency use cases.

### RemoveOrder

Finds an order by ID and removes it from the slice. Used for order cancellation.

Time complexity: O(n) linear scan.

### BestBid / BestAsk

Returns the first element of the respective slice — the most aggressive resting order.

Time complexity: O(1).

### Spread

Returns `BestAsk.Price.Sub(BestBid.Price)`. A tighter spread indicates higher liquidity. Returns `decimal.NewFromInt(-1)` if either side is empty.

### Depth

Returns the count of orders on each side: `(len(Bids), len(Asks))`.

### RemoveFilled

Filters out all orders where `Remaining.LessThanOrEqual(decimal.Zero)`. Called after each matching cycle to clean up fully filled orders.

## Visualization

```
         Order Book: BTC/USD
╔═══════════════╦═══════════════╗
║   Bids (Buy)  ║  Asks (Sell)  ║
╠═══════════════╬═══════════════╣
║  10 @ 50200   ║   5 @ 50300   ║  ← Best Bid / Best Ask
║   5 @ 50100   ║   8 @ 50400   ║
║   3 @ 50000   ║  12 @ 50500   ║
╚═══════════════╩═══════════════╝
        Spread: 100
```

## Design Notes

The current implementation uses sorted slices rather than a tree (e.g., red-black tree or skip list). This is a conscious trade-off:

| Approach | Insert | Remove | Best Price | Complexity |
|----------|--------|--------|------------|------------|
| Sorted slice | O(n log n) | O(n) | O(1) | Low |
| Red-black tree | O(log n) | O(log n) | O(log n) | Medium |
| Skip list | O(log n) avg | O(log n) avg | O(1) | Medium |

For an educational project prioritizing readability, sorted slices are the right choice. The upgrade path to a tree is localized to the `AddOrder` and `RemoveOrder` methods.
