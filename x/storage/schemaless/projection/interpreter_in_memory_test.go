package projection

//
//func TestDefaultInMemoryInterpreter(t *testing.T) {
//	log.SetLevel(log.DebugLevel)
//	log.SetFormatter(&log.TextFormatter{
//		ForceColors:     true,
//		TimestampFormat: "",
//		PadLevelText:    true,
//	})
//
//	dag := NewDAGBuilder()
//	_ = dag.
//		DoLoad(&GenerateHandler{DoLoad: func(push func(message Item)) error {
//			push(Item{Key: "1", Data: schema.FromGo(1)})
//			return nil
//		}}).
//		Window(&MapHandler[int, int]{
//			F: func(x int, returning func(key string, value int)) error {
//				returning("x", x+1)
//				return nil
//			},
//		}, WithName("DoSomething"))
//
//	t.Run("normal run, finishes", func(t *testing.T) {
//		interpreter := DefaultInMemoryInterpreter()
//		assert.NotNil(t, interpreter)
//
//		err := interpreter.Run(context.Background(), dag.Build())
//		assert.NoError(t, err)
//		//DEBUG  [0000] Window:  map(DoSomething, r=false) true false
//		//DEBUG  [0000] ✉️: {Offset:0 Key:1 Item:0x14000178d20 Watermark:<nil> finished:false} map(DoSomething, r=false)
//		//DEBUG  [0000] √ map(DoSomething, r=false)
//		//DEBUG  [0000] DoLoad: Finish DoLoad(root.Load0, r=false)
//		//DEBUG  [0000] Window: Finish map(DoSomething, r=false)
//		stats := interpreter.StatsSnapshotAndReset()
//		assert.Equal(t, 1, stats["load[root.Load0].returning"])
//		assert.Equal(t, 1, stats["map[DoSomething].returning.aggregate"])
//
//		// second time should return zero
//		stats = interpreter.StatsSnapshotAndReset()
//		assert.Equal(t, 0, stats["load[root.Load0].returning"])
//		assert.Equal(t, 0, stats["map[DoSomething].returning.aggregate"])
//	})
//
//	// should be able to run again the same DAG
//	t.Run("run on closed context should not execute, end return error", func(t *testing.T) {
//		// should not execute when context is cancelled
//		ctx, cancel := context.WithCancel(context.Background())
//		cancel()
//
//		interpreter := DefaultInMemoryInterpreter()
//		// TODO fix this, to proper error handling
//		err := interpreter.Run(ctx, dag.Build())
//		assert.NoError(t, err)
//
//		stats := interpreter.StatsSnapshotAndReset()
//		assert.Equal(t, 0, stats["load[root.Load0].returning"])
//		assert.Equal(t, 0, stats["map[DoSomething].returning.aggregate"])
//	})
//
//	t.Run("executing the same DAG twice should not execute twice", func(t *testing.T) {
//		interpreter := DefaultInMemoryInterpreter()
//		assert.NotNil(t, interpreter)
//
//		err := interpreter.Run(context.Background(), dag.Build())
//		assert.NoError(t, err)
//
//		err = interpreter.Run(context.Background(), dag.Build())
//		assert.ErrorIs(t, err, ErrInterpreterNotInNewState)
//	})
//}
