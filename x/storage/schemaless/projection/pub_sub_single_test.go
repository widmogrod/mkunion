package projection

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestPusSubSingle(t *testing.T) {
	pss := NewPubSubSingle()
	err := pss.Publish(context.Background(), Message{
		Key: "foo",
		Item: &Item{
			Key:       "foo",
			Data:      schema.FromGo("foo"),
			EventTime: 0,
		},
	})
	assert.NoError(t, err)

	pss.Finish()

	err = pss.Subscribe(context.Background(), 0, func(msg Message) error {
		assert.Equal(t, "foo", msg.Key)
		return nil
	})
	assert.NoError(t, err)

}
