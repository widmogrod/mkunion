package projection

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
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
	loaded := dag.
		Load(&GenerateHandler{
			Load: func(push func(message Item)) error {
				for item := range GenerateItemsEvery(withTime(10, 0), 20, 10*time.Millisecond) {
					push(item)
				}
				return nil
			},
		}).
		Map(Log("loaded"))

	mapped := loaded.
		Window(
			WithFixedWindow(50*time.Millisecond),
			WithTriggers(&AtWatermark{}),
		).
		Map(Log("window mapped"))

	end := mapped.
		Map(&SimpleProcessHandler{
			P: func(x Item, returning func(Item)) error {
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

	var result []string
	end.
		Map(&SimpleProcessHandler{
			P: func(x Item, returning func(Item)) error {
				result = append(result, schema.AsDefault[string](x.Data, "-"))
				return nil
			},
		})

	interpret := NewInMemoryTwoInterpreter()
	err := interpret.Run(context.Background(), dag.Build())
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"9,8,7,6,5,",
		"14,13,12,11,10,",
		"19,18,17,16,15,",
		"4,3,2,1,0,",
	}

	// order of elements is not guaranteed
	assert.ElementsMatch(t, expected, result)
}
