package projection

//go:generate go run ../../cmd/mkunion/main.go

//go:tag mkunion:"TriggerDescription"
type (
	//AtPeriod struct {
	//	Duration time.Duration
	//}
	//AtWindowItemSize struct {
	//	Number int
	//}
	AtWatermark struct {
		Timestamp int64
	}
	//AnyOf struct {
	//	Triggers []TriggerDescription
	//}
	//AllOf struct {
	//	Triggers []TriggerDescription
	//}
)

func FlushWindow[A any](x *Window, data []*Record[A]) []*Record[A] {
	return data
}
