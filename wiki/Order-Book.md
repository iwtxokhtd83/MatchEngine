# Order Book

The order book is the central data structure that holds all resting orders for a given trading symbol.

## Architecture

Each side of the book (bids and asks) is organized as a price-level map with sorted price keys:

```
orderSide
├── prices: [50200, 50100, 50000]          ← sorted decimal slice (binary search)
├── levels: map[string]*priceLevel         ← price.String() -> FIFO order queue
│   ├── "50200" → [order1, order2]
│   ├── "50100" → [order3]
│   └── "50000" → [order4, order5, order6]
└── count: 6                               ← total order count
```

This replaces the previous sorted-slice approach (O(n log n) per insert) with a price-level structure that gives O(log p) insertion where p is the number of distinct price levels.

## Sorting Rules

### Bids (Buy Side)

Price levels sorted by price descending. Within each level, orders are in FIFO order.

```
Bids:
  10 @ 105  (arrived 10:00:01)
  5  @ 105  (arrived 10:00:03)   ← same price, later arrival
  8  @ 100  (arrived 10:00:00)
  3  @ 99   (arrived 10:00:02)
```

### Asks (Sell Side)

Price levels sorted by price ascending. Within each level, orders are in FIFO order.

```
Asks:
  5  @ 106  (arrived 10:00:00)
  2  @ 106  (arrived 10:00:04)   ← same price, later arrival
  10 @ 110  (arrived 10:00:01)
  7  @ 115  (arrived 10:00:02)
```

## Complexity

| Operation | Old (sorted slice) | New (price-level map) |
|-----------|-------------------|----------------------|
| AddOrder | O(n log n) full re-sort | O(log p) binary search + O(1) append |
| RemoveOrder | O(n) linear scan | O(1) map lookup + O(k) within level |
| BestBid / BestAsk | O(1) | O(1) |
| RemoveFilled | O(n) | O(n) |
| Snapshot | O(n) | O(n) |

Where: n = total orders, p = distinct price levels, k = orders at a given price.

In practice, p is much smaller than n. A book with 10,000 orders might have only 50-100 distinct price levels.

## Operations

### AddOrder

Uses `sort.Search()` (binary search) to find the correct position in the sorted price slice. If the price level already exists, the order is simply appended to that level's FIFO queue in O(1). If it's a new price, a new level is created and the price is inserted at the correct position.

### RemoveOrder

O(1) map lookup to find the order, then removes it from its price level. If the level becomes empty, the price is removed from the sorted slice.

### BestBid / BestAsk

Returns the front order of the first price level. O(1).

### Bids() / Asks()

Returns all orders flattened in price-time priority order. Used by `Snapshot()` and for display.

### Spread

Returns `BestAsk.Price.Sub(BestBid.Price)`. Returns `decimal.NewFromInt(-1)` if either side is empty.

### Depth

Returns the total order count on each side.

### RemoveFilled

Scans all levels, removes filled orders, and cleans up empty price levels.

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
