package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade represents an executed trade between two orders.
type Trade struct {
	BuyOrderID  string
	SellOrderID string
	Price       decimal.Decimal
	Quantity    decimal.Decimal
	Timestamp   time.Time
}
