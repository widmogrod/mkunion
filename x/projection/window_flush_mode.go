package projection

//go:generate go run ../../cmd/mkunion/main.go

//go:tag mkunion:"WindowFlushMode"
type (
	//Accumulate struct {
	//	AllowLateArrival time.Duration
	//}
	Discard struct{}
	//AccumulatingAndRetracting struct {
	//	AllowLateArrival time.Duration
	//}
)
