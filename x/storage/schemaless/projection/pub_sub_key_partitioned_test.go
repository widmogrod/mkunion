package projection

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChan(t *testing.T) {
	x := make(chan int, 10)
	x <- 1
	x <- 2
	x <- 3
	close(x)

	counter := 0
	for range x {
		counter++
	}

	assert.Equal(t, 3, counter)
}

//func TestPartitionedPubSub(t *testing.T) {
//	queue := NewPartitionedQueue()
//
//	for i := 0; i < 100; i++ {
//		queue.Publish(Item{
//			Key:       fmt.Sprintf("key-%d", i%10),
//			Data:      schema.MkInt(i),
//			EventTime: 0,
//			Window:    0,
//			finished:  false,
//		})
//	}
//
//	// operations on queue should result in Window or DoWindow operations per key group
//	// Window and DoWindow should be executed in parallel
//
//	mappedQueue := NewPartitionedQueue()
//
//	var mapWorkers = 3
//	var mapped = make(chan *Item)
//	queue.Subscribe(func(i *Item) {
//		mapped <- i
//	})
//
//	for i := 0; i < mapWorkers; i++ {
//		go TryMap(mapped, mappedQueue)
//	}
//
//	var groups = make(map[string]chan Item)
//	var newGroups = make(chan string)
//	queue.Subscribe(func(i *Item) {
//		if _, ok := groups[i.Key]; !ok {
//			groups[i.Key] = make(chan Item)
//			newGroups <- i.Key
//		}
//
//		groups[i.Key] <- *i
//	})
//
//	// when subscription is closed, all groups should be closed
//	// and all groups should be merged
//
//	for group := range newGroups {
//		go func(group string) {
//			TryMerge(groups[group])
//		}(group)
//	}
//
//}
//
//func TryMerge(items chan Item) {
//	for {
//		v1, ok1 := <-items
//		v2, ok2 := <-items
//
//		if !ok1 || !ok2 {
//			if ok1 {
//				fmt.Printf("v1: %v \n", v1)
//			} else if ok2 {
//				fmt.Printf("v2: %v \n", v2)
//			}
//			break
//		}
//	}
//}
