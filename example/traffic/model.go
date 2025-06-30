package traffic

//go:tag mkunion:"TrafficState,no-type-registry"
type (
	RedLight    struct{}
	YellowLight struct{}
	GreenLight  struct{}
)

//go:tag mkunion:"TrafficCommand,no-type-registry"
type (
	NextCMD struct{} // Move to next state in sequence
)

// Simple traffic light with no dependencies
type Dependencies struct{}
