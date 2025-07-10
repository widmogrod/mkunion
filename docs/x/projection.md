---
title: Projection Package
---

# x/projection - Event Processing and Stream Analytics

The `x/projection` package provides a comprehensive framework for event processing, stream analytics, and data projections. It supports windowing, watermarks, and complex event processing patterns with a focus on correctness and scalability.

## Overview

The projection package offers:
- **Event time processing** - Handle out-of-order events with watermarks
- **Flexible windowing** - Fixed, sliding, and session windows
- **Stream operations** - Map, filter, join, aggregate, and more
- **State management** - Snapshots and recovery for fault tolerance
- **DAG-based API** - Intuitive pipeline construction
- **Multiple execution modes** - In-memory and distributed processing

## Core Concepts

### Records and Items

The fundamental data units in projections:

```go
// Record with event time
type Record[A any] struct {
    Key       string
    Data      A
    EventTime time.Time
}

// Item in schemaless projection
type Item struct {
    Key       string
    Data      schema.Schema
    EventTime time.Time
    Window    Window
    Type      string
}
```

### Watermarks

Watermarks track progress in event time:

```go
type Watermark[A any] struct {
    EventTime time.Time
    Uptime    time.Time // Processing time
}

// Watermarks indicate that all events with 
// EventTime < watermark.EventTime have been seen
```

### Windows

Windows group events for processing:

```go
// Fixed windows of constant size
window := &FixedWindow{
    Width: time.Hour,
}

// Sliding windows with overlap
window := &SlidingWindow{
    Width:  time.Hour,
    Period: 30 * time.Minute,
}

// Session windows based on inactivity gaps
window := &SessionWindow{
    Gap: 5 * time.Minute,
}
```

## Getting Started

### Basic Stream Processing

```go
// Define your data types
type SensorReading struct {
    SensorID    string
    Temperature float64
    Timestamp   time.Time
}

type AvgTemp struct {
    SensorID string
    AvgTemp  float64
    Count    int
}

// Create a simple aggregation pipeline
dag := projection.NewDAGBuilder()

pipeline := dag.
    Load(sensorDataSource).
    Window(
        projection.WithFixedWindow(5 * time.Minute),
        projection.WithTriggers(&projection.AtWatermark{}),
    ).
    Map(&projection.MapHandler[SensorReading, AvgTemp]{
        F: func(reading SensorReading, returning func(string, AvgTemp)) error {
            // Group by sensor
            returning(reading.SensorID, AvgTemp{
                SensorID: reading.SensorID,
                AvgTemp:  reading.Temperature,
                Count:    1,
            })
            return nil
        },
    }).
    Map(&projection.MergeHandler[AvgTemp]{
        Combine: func(a, b AvgTemp) (AvgTemp, error) {
            total := a.AvgTemp*float64(a.Count) + b.AvgTemp*float64(b.Count)
            count := a.Count + b.Count
            return AvgTemp{
                SensorID: a.SensorID,
                AvgTemp:  total / float64(count),
                Count:    count,
            }, nil
        },
    })

// Execute the pipeline
interpreter := projection.NewInMemoryInterpreter()
results := interpreter.Run(ctx, pipeline.Build())
```

### Event Time Processing

Handle out-of-order events correctly:

```go
// Configure watermark generation
watermarkGen := &projection.WatermarkGenerator{
    // Allow 1 minute for late events
    MaxOutOfOrderness: 1 * time.Minute,
    
    // Extract event time from records
    ExtractTimestamp: func(record Record[Event]) time.Time {
        return record.Data.Timestamp
    },
}

// Process with watermarks
pipeline := dag.
    Load(eventSource).
    WithWatermarks(watermarkGen).
    Window(
        projection.WithFixedWindow(10 * time.Minute),
        projection.WithAllowedLateness(2 * time.Minute),
    )
```

## Advanced Features

### Complex Event Processing

#### Pattern Detection

```go
// Detect patterns in event streams
type Pattern struct {
    Start   Event
    Middle  []Event
    End     Event
}

patternDetector := &projection.PatternHandler[Event, Pattern]{
    // Define pattern matching logic
    Match: func(events []Event) (*Pattern, bool) {
        if len(events) < 3 {
            return nil, false
        }
        
        // Check if events match pattern
        if events[0].Type == "START" && 
           events[len(events)-1].Type == "END" {
            return &Pattern{
                Start:  events[0],
                Middle: events[1:len(events)-1],
                End:    events[len(events)-1],
            }, true
        }
        return nil, false
    },
}
```

#### Stream Joins

```go
// Join two streams
orders := dag.Load(orderSource)
payments := dag.Load(paymentSource)

joined := dag.Join(
    orders,
    payments,
    &projection.JoinHandler[Order, Payment, OrderPayment]{
        // Join condition
        JoinKey: func(order Order) string {
            return order.ID
        },
        PaymentKey: func(payment Payment) string {
            return payment.OrderID
        },
        
        // Join function
        Join: func(order Order, payment Payment) OrderPayment {
            return OrderPayment{
                Order:   order,
                Payment: payment,
            }
        },
        
        // Window for join
        Window: projection.WithFixedWindow(1 * time.Hour),
    },
)
```

### Windowing Strategies

#### Triggers

Control when window results are emitted:

```go
// Emit when watermark passes window end
trigger := &projection.AtWatermark{}

// Emit every 30 seconds
trigger := &projection.AtPeriod{
    Period: 30 * time.Second,
}

// Emit when window has 100 items
trigger := &projection.AtWindowItemSize{
    Size: 100,
}

// Combine triggers
trigger := &projection.AnyOf{
    Triggers: []projection.Trigger{
        &projection.AtWatermark{},
        &projection.AtPeriod{Period: 1 * time.Minute},
    },
}
```

#### Flush Modes

Handle window state after emission:

```go
// Discard window state after emission
projection.WithDiscard()

// Keep accumulating (for running totals)
projection.WithAccumulate()

// Support late data with retractions
projection.WithAccumulatingAndRetracting()
```

### State Management and Recovery

```go
// Enable snapshots for fault tolerance
pipeline := dag.
    Load(source).
    EnableSnapshots(
        projection.WithSnapshotInterval(5 * time.Minute),
        projection.WithSnapshotStorage(snapshotStore),
    ).
    Map(processor)

// Recover from snapshot
recovery := &projection.Recovery{
    SnapshotID: lastSnapshot.ID,
    Storage:    snapshotStore,
}

pipeline.RecoverFrom(recovery)
```

## Built-in Handlers

### Data Transformation

```go
// Map: Transform items
mapper := &projection.MapHandler[Input, Output]{
    F: func(input Input, emit func(key string, value Output)) error {
        output := transform(input)
        emit(input.Key, output)
        return nil
    },
}

// FlatMap: One-to-many transformation
flatMapper := &projection.MapHandler[User, Event]{
    F: func(user User, emit func(key string, value Event)) error {
        for _, activity := range user.Activities {
            emit(user.ID, Event{
                UserID: user.ID,
                Type:   activity.Type,
                Time:   activity.Time,
            })
        }
        return nil
    },
}
```

### Filtering

```go
// Filter based on predicate
filter := &projection.FilterHandler[Event]{
    Predicate: func(event Event) bool {
        return event.Type == "purchase" && event.Amount > 100
    },
}
```

### Aggregation

```go
// Count items
counter := &projection.CountHandler[Event]{}

// Calculate averages
averager := &projection.AvgHandler[Metric]{
    GetValue: func(m Metric) float64 {
        return m.Value
    },
}

// Custom aggregation
aggregator := &projection.MergeHandler[Stats]{
    // Initial state
    Zero: func() Stats {
        return Stats{Count: 0, Sum: 0}
    },
    
    // Combine function
    Combine: func(a, b Stats) (Stats, error) {
        return Stats{
            Count: a.Count + b.Count,
            Sum:   a.Sum + b.Sum,
        }, nil
    },
}
```

## DAG Builder API

The DAG builder provides a fluent API for constructing pipelines:

```go
dag := projection.NewDAGBuilder()

// Source nodes
events := dag.Load(eventSource, projection.WithName("events"))
configs := dag.Load(configSource, projection.WithName("configs"))

// Transform and enrich
enriched := events.
    Join(configs, enrichmentJoin).
    Filter(validationFilter).
    Map(enrichmentMapper)

// Window and aggregate
hourlyStats := enriched.
    Window(
        projection.WithFixedWindow(1 * time.Hour),
        projection.WithTriggers(&projection.AtWatermark{}),
    ).
    GroupByKey().
    Aggregate(statsAggregator)

// Multiple outputs
hourlyStats.
    Branch(
        projection.Branch{
            Name: "alerts",
            Filter: alertFilter,
            Handler: alertHandler,
        },
        projection.Branch{
            Name: "metrics",
            Handler: metricsHandler,
        },
    )

// Build the execution plan
nodes := dag.Build()
```

## Integration Examples

### With x/storage

Store projection results:

```go
// Create storage sink
sink := &projection.StorageSink[Stats]{
    Repository: typedful.NewTypedRepository[Stats](storage),
    RecordType: "hourly_stats",
}

// Add to pipeline
pipeline.
    Window(projection.WithFixedWindow(1 * time.Hour)).
    Aggregate(aggregator).
    Sink(sink)
```

### With x/stream

Use different stream backends:

```go
// Kafka stream source
kafkaStream := stream.NewKafkaStream(kafkaConfig)
source := projection.NewStreamSource(kafkaStream)

// In-memory stream for testing
memStream := stream.NewInMemoryStream()
testSource := projection.NewStreamSource(memStream)
```

## Performance Optimization

### Parallelism

```go
// Configure parallel execution
interpreter := projection.NewInMemoryInterpreter(
    projection.WithParallelism(4),
    projection.WithBufferSize(1000),
)

// Parallel processing in handlers
mapper := &projection.MapHandler[Input, Output]{
    Parallel: true,
    F: func(input Input, emit func(string, Output)) error {
        // Thread-safe processing
    },
}
```

### Memory Management

```go
// Configure memory limits
projection.WithMaxWindowSize(10000)
projection.WithEvictionPolicy(projection.LRU)

// Use disk-backed state
projection.WithStateBackend(diskBackend)
```

## Testing

### Unit Testing Handlers

```go
func TestAggregator(t *testing.T) {
    handler := &MergeHandler[Stats]{...}
    
    // Test combine function
    result, err := handler.Combine(
        Stats{Count: 5, Sum: 50},
        Stats{Count: 3, Sum: 30},
    )
    
    assert.NoError(t, err)
    assert.Equal(t, Stats{Count: 8, Sum: 80}, result)
}
```

### Integration Testing

```go
func TestPipeline(t *testing.T) {
    // Create test source
    source := projection.NewTestSource([]Record[Event]{
        {Key: "1", Data: event1, EventTime: time1},
        {Key: "2", Data: event2, EventTime: time2},
    })
    
    // Build and run pipeline
    dag := projection.NewDAGBuilder()
    pipeline := dag.Load(source).Window(...).Aggregate(...)
    
    results := projection.RunTest(t, pipeline)
    
    // Assert results
    assert.Len(t, results, expectedCount)
}
```

## Best Practices

1. **Event Time vs Processing Time**: Always use event time for business logic
2. **Watermark Strategy**: Balance lateness tolerance with result timeliness
3. **Window Size**: Consider memory usage and business requirements
4. **Error Handling**: Implement proper error handling in all handlers
5. **State Size**: Keep window state manageable, use appropriate flush modes

## Troubleshooting

Common issues and solutions:

1. **Late Events**: Increase allowed lateness or adjust watermark strategy
2. **Memory Issues**: Reduce window size or use disk-backed state
3. **Performance**: Enable parallelism, optimize handlers
4. **Incorrect Results**: Check event time extraction and window configuration

## Further Reading

- [Stream Processing Concepts](https://www.oreilly.com/library/view/streaming-systems/9781491983867/)
- [x/storage Integration](./storage.md#projection-system)
- [x/stream Package](./stream.md)
- [Windowing Strategies](../examples/windowing.md)