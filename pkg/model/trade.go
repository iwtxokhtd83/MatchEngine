package model

import "time"

// Trade represents an executed trade between two orders.
type Trade struct {
	BuyOrderID  string
	SellOrderID string
	Price       float64
	Quantity    float64
	Timestamp   time.Time
}
