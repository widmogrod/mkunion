package projection

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"testing"
	"time"
)

func TestDefaultInMemoryInterpreter2(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "",
		PadLevelText:    true,
	})

	dag := NewDAGBuilder()
	loaded := dag.Load(&GenerateHandler{
		Load: func(push func(message Item)) error {
			for item := range GenerateItemsEvery(withTime(10, 0), 20, 10*time.Millisecond) {
				push(item)
			}
			return nil
		},
	})

	mapped := loaded.
		Window(WithTriggers(&AtPeriod{100 * time.Millisecond})).
		Map(Log("window"))

	mapped.
		Map(&SimpleProcessHandler{
			P: func(x Item, returning func(Item)) error {
				d, _ := shared.JSONMarshal[schema.Schema](schema.FromGo(x))
				log.Errorln("merge=", string(d))
				//return MustMatchProcessItem(
				//	x,
				//	func(x *Item) error {
				//		panic("implement me")
				//	},
				//	func(x *ItemAggregate) error {
				//		panic("implement me")
				//	},
				//	func(x *ItemAggregateAndRetract) error {
				//		returning(Item{
				//			Key: x.Key,
				//			Data: schema.FromGo(schema.Reduce(x.Aggregate, fmt.Sprintf("-1(%s)", schema.AsDefault[string](x.Retract, "-1")), func(x schema.Schema, agg string) string {
				//				return fmt.Sprintf("%d,%s", schema.AsDefault[int](x, 0), agg)
				//			})),
				//			EventTime: x.EventTime,
				//			Window:    x.Window,
				//		})
				//		return nil
				//
				//	},
				//)

				previous := schema.GetSchemaDefault[string](x.Data, "Previous", "")
				current, _ := schema.GetSchema(x.Data, "Current")

				returning(Item{
					Key: x.Key,
					Data: schema.FromGo(schema.Reduce(current, previous, func(x schema.Schema, agg string) string {
						return fmt.Sprintf("%d,%s", schema.AsDefault[int](x, 0), agg)
					})),
					EventTime: x.EventTime,
					Window:    x.Window,
				})
				log.Info("merge end")
				return nil
			},
		}, WithAccumulatingAndRetracting()).
		Map(Log("log"))

	interpret := NewInMemoryTwoInterpreter()
	err := interpret.Run(context.Background(), dag.Build())
	if err != nil {
		t.Fatal(err)
	}
}
