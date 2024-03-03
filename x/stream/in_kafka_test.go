package stream

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"testing"
)

func TestKafkaStream(t *testing.T) {
	address := os.Getenv("KAFKA_SERVERS")
	if address == "" {
		t.Skip(`Skipping test because:
- KAFKA_SERVERS that points to localstack is not set.
- Assuming localstack is not running.

To run this test, please set KAFKA_SERVERS like:
	export KAFKA_SERVERS=localhost
`)
		return
	}

	cm := kafka.ConfigMap{
		"bootstrap.servers":  address,
		"group.id":           "myGroup23",
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	}
	pm := kafka.ConfigMap{
		"bootstrap.servers": address,
	}

	stream := NewKafkaStream[int](cm, pm, WithSystemTime)
	if assert.NotNil(t, stream) {
		HappyPathSpec(t, stream, func() int {
			return rand.Int()
		})
	}
}
