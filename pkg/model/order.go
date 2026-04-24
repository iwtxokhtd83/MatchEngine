package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Side represents the order side (buy or sell).
type Side int

const (
	Buy Side = iota
	Sell
)

func (s Side) String() string {
	if s == Buy {
		return "BUY"
	}
	return "SELL"
}

// OrderType represents the type of order.
type OrderType int

const (
	Limit OrderType = iota
	Market
	StopMarket // becomes Market when stop price is triggered
	StopLimit  // becomes Limit when stop price is triggered
)

func (t OrderType) String() string {
	switch t {
	case Limit:
		return "LIMIT"
	case Market:
		return "MARKET"
	case StopMarket:
		return "STOP_MARKET"
	case StopLimit:
		return "STOP_LIMIT"
	default:
		return "UNKNOWN"
	}
}

// Order represents a trading order.
type Order struct {
	ID        string
	OwnerID   string          // identifies the trader (used for self-trade prevention)
	Side      Side
	Type      OrderType
	TIF       TimeInForce     // time-in-force policy (default: GTC)
	Price     decimal.Decimal // limit price (ignored for market orders)
	StopPrice decimal.Decimal // trigger price for stop orders
	Quantity  decimal.Decimal
	Remaining decimal.Decimal
	Timestamp time.Time
}

// NewLimitOrder creates a new limit order.
func NewLimitOrder(id string, side Side, price, quantity decimal.Decimal) *Order {
	return &Order{
		ID:        id,
		Side:      side,
		Type:      Limit,
		Price:     price,
		Quantity:  quantity,
		Remaining: quantity,
		Timestamp: time.Now(),
	}
}

// NewMarketOrder creates a new market order.
func NewMarketOrder(id string, side Side, quantity decimal.Decimal) *Order {
	return &Order{
		ID:        id,
		Side:      side,
		Type:      Market,
		Quantity:  quantity,
		Remaining: quantity,
		Timestamp: time.Now(),
	}
}

// IsFilled returns true if the order is completely filled.
func (o *Order) IsFilled() bool {
	return o.Remaining.LessThanOrEqual(decimal.Zero)
}
