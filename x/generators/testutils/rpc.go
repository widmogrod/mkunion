package testutils

import "github.com/widmogrod/mkunion/f"

// Inquiry and Reply show the RPC pairing: every Inquiry answers with a Reply variant,
// so besides the typed handler mkunion generates DispatchInquiry, NewInquiryClient and
// InquiryUnexpectedAnswer. The answers= option names the union to check against;
// a variant that answers with anything else fails generation.
//
//go:tag mkunion:"Inquiry,answers=Reply"
type (
	Ping  struct{ f.Returns[*Pong] }
	Fetch struct {
		f.Returns[*Page]
		URL string
	}
)

//go:tag mkunion:"Reply"
type (
	Pong    struct{}
	Page    struct{ Body string }
	Refused struct{ Reason string }
)

// Refused is the Reply that is an error, so a client can errors.As to it
// through InquiryUnexpectedAnswer.Unwrap.
func (r *Refused) Error() string { return r.Reason }

// Probe has no answers= option: Reading is inferred because every Probe
// answers with *Reading variant.
//
//go:tag mkunion:"Probe"
type (
	ReadTemp struct{ f.Returns[*Celsius] }
)

//go:tag mkunion:"Reading"
type (
	Celsius struct{ Value float64 }
)
