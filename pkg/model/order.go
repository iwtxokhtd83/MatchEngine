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
)

func (t OrderType) String() string {
	if t == Limit {
		return "LIMIT"
	}
	return "MARKET"
}

// Order represents a trading order.
type Order struct {
	ID        string
	OwnerID   string          // identifies the trader (used for self-trade prevention)
	Side      Side
	Type      OrderType
	Price     decimal.Decimal // ignored for market orders
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
