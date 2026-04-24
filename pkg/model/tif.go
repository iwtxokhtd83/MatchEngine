package model

// TimeInForce defines how long an order remains active.
type TimeInForce int

const (
	// GTC (Good-Till-Cancelled) rests in the book until filled or cancelled. Default behavior.
	GTC TimeInForce = iota
	// IOC (Immediate-or-Cancel) fills as much as possible immediately, then cancels the rest.
	IOC
	// FOK (Fill-or-Kill) must be filled entirely immediately or is cancelled completely.
	FOK
)

func (t TimeInForce) String() string {
	switch t {
	case GTC:
		return "GTC"
	case IOC:
		return "IOC"
	case FOK:
		return "FOK"
	default:
		return "UNKNOWN"
	}
}
