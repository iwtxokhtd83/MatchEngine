package model

// STPMode defines the self-trade prevention mode.
type STPMode int

const (
	// STPNone disables self-trade prevention (default).
	STPNone STPMode = iota
	// STPCancelResting cancels the resting order and continues matching the incoming order.
	STPCancelResting
	// STPCancelIncoming cancels the incoming order entirely when a self-trade is detected.
	STPCancelIncoming
	// STPCancelBoth cancels both the resting and incoming orders.
	STPCancelBoth
	// STPDecrement reduces both orders by the overlap quantity without producing a trade.
	STPDecrement
)

func (m STPMode) String() string {
	switch m {
	case STPNone:
		return "NONE"
	case STPCancelResting:
		return "CANCEL_RESTING"
	case STPCancelIncoming:
		return "CANCEL_INCOMING"
	case STPCancelBoth:
		return "CANCEL_BOTH"
	case STPDecrement:
		return "DECREMENT"
	default:
		return "UNKNOWN"
	}
}
