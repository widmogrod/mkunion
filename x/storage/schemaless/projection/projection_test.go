package projection

import (
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
)

var generateData []Item

func init() {
	generateData = []Item{
		Item{
			Key: "game:1",
			Data: schema.FromGo(Game{
				Players: []string{"a", "b"},
				Winner:  "a",
			}),
		},
		Item{
			Key: "game:2",
			Data: schema.FromGo(Game{
				Players: []string{"a", "b"},
				Winner:  "b",
			}),
		},
		Item{
			Key: "game:3",
			Data: schema.FromGo(Game{
				Players: []string{"a", "b"},
				IsDraw:  true,
			}),
		},
	}
}

func GenerateData() *GenerateHandler {
	return &GenerateHandler{
		Load: func(returning func(message Item)) error {
			for _, msg := range generateData {
				returning(msg)
			}
			return nil
		},
	}
}

func MapGameToStats() *MapHandler[Game, SessionsStats] {
	return &MapHandler[Game, SessionsStats]{
		F: func(x Game, returning func(key string, value SessionsStats)) error {
			for _, player := range x.Players {
				wins := 0
				draws := 0
				loose := 0

				if x.IsDraw {
					draws = 1
				} else if x.Winner == player {
					wins = 1
				} else {
					loose = 1
				}

				returning("session-stats-by-player:"+player, SessionsStats{
					Wins:  wins,
					Draws: draws,
					Loose: loose,
				})
			}

			return nil
		},
	}
}

func MergeSessionStats() *MergeHandler[SessionsStats] {
	return &MergeHandler[SessionsStats]{
		Combine: func(base, x SessionsStats) (SessionsStats, error) {
			return SessionsStats{
				Wins:  base.Wins + x.Wins,
				Draws: base.Draws + x.Draws,
				Loose: base.Loose + x.Loose,
			}, nil
		},
		DoRetract: func(base, x SessionsStats) (SessionsStats, error) {
			panic("retraction on SessionStats should not happen")
		},
	}
}

func CountTotalSessionsStats(b Builder) Builder {
	return b.
		Map(&MergeHandler[int]{
			Combine: func(base, x int) (int, error) {
				log.Debugln("counting(+)", base+x, base, x)
				return base + x, nil
			},
			DoRetract: func(base int, x int) (int, error) {
				log.Debugln("counting(-)", base+x, base, x)
				return base - x, nil
			},
		}, WithName("CountTotalSessionsStats:Count"))
}

//func TestProjection(t *testing.T) {
//	log.SetLevel(log.DebugLevel)
//	log.SetFormatter(&log.TextFormatter{
//		ForceColors:     true,
//		TimestampFormat: "",
//		PadLevelText:    true,
//	})
//	store := schemaless.NewInMemoryRepository()
//	sessionStatsRepo := typedful.NewTypedRepository[SessionsStats](store)
//	totalRepo := typedful.NewTypedRepository[int](store)
//
//	dag := NewDAGBuilder()
//	games := dag.
//		DoLoad(GenerateData(), WithName("GenerateData"))
//	gameStats := games.
//		Window(MapGameToStats(), WithName("MapGameToStats"))
//	gameStatsBySession := gameStats.
//		DoWindow(MergeSessionStats(), WithName("MergeSessionStats"))
//
//	_ = CountTotalSessionsStats(gameStatsBySession).
//		Window(NewRepositorySink("total", store), WithName("Sink ⚽️TotalCount"))
//
//	_ = gameStatsBySession.
//		Window(NewRepositorySink("session", store), WithName("NewRepositorySink"))
//
//	interpretation := DefaultInMemoryInterpreter()
//	err := interpretation.Run(context.Background(), dag.Build())
//	assert.NoError(t, err)
//
//	result, err := sessionStatsRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[SessionsStats]]{
//		RecordType: "session",
//	})
//	assert.NoError(t, err)
//	assert.Len(t, result.Items, 2)
//	for _, x := range result.Items {
//		v, err := schema.ToJSON(schema.FromGo(x.Data))
//		assert.NoError(t, err)
//		fmt.Printf("item: id=%s type-%s %s\n", x.ID, x.Type, string(v))
//	}
//
//	stats, err := sessionStatsRepo.Get("session-stats-by-player:a", "session")
//	assert.NoError(t, err)
//	assert.Equal(t, SessionsStats{
//		Wins:  1,
//		Loose: 1,
//		Draws: 1,
//	}, stats.Data)
//
//	stats, err = sessionStatsRepo.Get("session-stats-by-player:b", "session")
//	assert.NoError(t, err)
//	assert.Equal(t, SessionsStats{
//		Wins:  1,
//		Loose: 1,
//		Draws: 1,
//	}, stats.Data)
//
//	total, err := totalRepo.Get("total", "total")
//	assert.NoError(t, err)
//	assert.Equal(t, 2, total.Data)
//}

//func TestLiveSelect(t *testing.T) {
//	log.SetLevel(log.DebugLevel)
//	log.SetFormatter(&log.TextFormatter{
//		ForceColors:     true,
//		TimestampFormat: "",
//		PadLevelText:    true,
//	})
//
//	// setup type registry
//	schema.RegisterRules([]schema.RuleMatcher{
//		schema.WhenPath(nil, schema.UseStruct(&schemaless.Record[Game]{})),
//		// mkunion should be able to deduce this type
//		// todo add this feature!
//		schema.WhenPath([]string{"Data"}, schema.UseStruct(Game{})),
//	})
//
//	// This is example that is aiming to explore concept of live select.
//	// Example use case in context of tic-tac-toe game:
//	// - As a player I want to see my stats in session in real time
//	// - As a player I want to see tic-tac-toe game updates in real time
//	//   (I wonder how live select would compete vs current implementation on websockets)
//	// - (some other) As a player I want to see achivements, in-game messages, in real time
//	//
//	// 	LIVE SELECT
//	//      sessionID,
//	//		COUNT_BY_VALUE(winner, WHERE winnerID NOT NULL) as wins, // {"a": 1, "b": 2}
//	//		COUNT(WHERE isDraw = TRUE) as draws,
//	//      COUNT() as total
//	//  GROUP BY sessionID as group
//	//  WHERE sessionID = :sessionID
//	//    AND gameState = "GameFinished"
//	//
//	// Solving live select model with DAG, can solve also MATERIALIZED VIEW problem with ease.
//	//
//	// At some point it would be nice to have benchmarks that show where is breaking point of
//	// doing ad hock live selects vs precalculating materialized view.
//	// ---
//	// This example works, and there are few things to solve
//	// - detect when there is no updates [v]
//	//   - let's data producer send a signal that it finished, and no frther updates will sent [v]
//	//   - add watermarking, to detect, what are latest events in system [TODO when there will be windowing]
//	// - closing live select, when connection is closed [TODO add context]
//	//------------------------------------
//	// - DAG compilation, like loading data from stream,
//	//   - if steam is Kinesis, there is limit of consumers that can be attached to stream.
//	//     this means, that when there can be few milion of live select, there will be need to have some other way to DoLoad data to DAG
//	//     - one way is to have part of DAG to recognise this limitation, and act within limit of kinesis, an have only few consumers
//	//       that push data to a solution, that can handle millions of lightweight consumers,
//	//			- RabbitMQ, it's all about topology of messages, few thousen of consumers should be fine
//	//          - Redis?
//	//     	    - In memeory, since DAG for live select is already in memeory, to could be able to route messaged to at lewast few thousend of consumers
//	//          - if DAG would push message to API Gateway Websocket, then state on one node is not concerned,
//	//            but what is, is that each node may have some data, and then it will need to re route them to other nodeDAGBuilders to create final aggregate
//	//            which means that each node needs to have knowladge which node process process which kays ranges
//	//
//	// - DoLoad node1
//	//   - Select repository (Optimise and cache)
//	//   - Take events related to a filter from stream (steam reads from a partition, so it has olny potion of data)
//	//   - Window & DoWindow
//	//   - Push to web socket
//	//
//	//   since every above optimisation would require some kind of cluster, and mitigates some limitations,
//	//   but since live select is always from a point of time, and later is interested in having only latest data pushes, maybe it make sense
//	//   to have only data for that window in memory. Then all steam data whenever live select request for it or now, would be computer for recently change data
//	//   that way, cluster only works for time horizon. Time horizon is smaller than all data,
//	//     it still could be horizontly scaled, each node would have it's own range of keys
//	//
//	//  Framing problem of live select as select on record with only one element that exists in database (no joins)
//	//  when connected with RepositoryWithAggregate, solves live select by only working with stream and waiting for updates, no need to past data, only updates
//	//  that way, select to DynamoDB won't be needed, and thise other otimisations (like caching DAX or Reads from OpenSearch) won't be needed
//	//
//	//
//	//
//	//
//	//---------------------------------
//	// - optimiastion of DAGs, few edges in line, withotu forks, can be executed in memory, without need of streams between them
//	// - what if different partitions needs to merge? like count total,
//	//   data from different counting nodeDAGBuilders, should be send to one selected node
//	//   - How to sove such partitioning? would RabbitMQ help or make things harder?
//	// - DynamoDB loader, can have information on how many RUs to use, like 5% percent
//	// - when system is on production, and there will be more live select DAGs,
//	//   - loading subset of records from db, may be fine for live select
//	//   - but what if there will be a lot of new DAGs, that need to process all data fron whole db?
//	//      my initial assumption, was that DAGs can be lightwaight, so that I can add new forks on runtime,
//	//      but fork on "joined" will be from zero oldest offset, and may not have data from DB, so it's point in time
//	//      maybe this means that instead of having easy way of forking, just DAGs can be deployed with full DoLoad from DB
//	//      since such situation can happen multiple times, that would mean that database needs to be optimised for massive parallel reads
//	//
//	//      	Premature optimisation: In context of DDB, this will consume a lot of RCUs,
//	//       	so that could be solved by creating a data (delta) lake on object storage like S3,
//	//      	Where there is DAG that use DDB and stream to keep S3 data up to date, and always with the latest representation
//	//
//	//		Thinking in a way that each DAG is separate deployment, that tracks it's process
//	// 		Means that change is separates, deployments can be separate, scaling needs can be separate, blast radius and ownership as well
//	//      More teams can work in parallel, and with uniform language of describing DAGs, means that domain concepts can be included as library
//	//
//	//		From that few interesing patterns can happed, (some described in Data Architecture at Scale)
//	//		- Read-only Data Stores. Sharing read RDS, each team can gen a database that other team has,
//	//	      deployed to their account, and keep up to date by data system (layer)
//	//		  which means, each system, can do reads as much as they can with once proximity to data (different account can be in different geo regions)
//	//		  which means, each system, can share libraries that perform domain specific queries, and those libraries can use RDS in their account
//	//		  which means, that those libraries, can have also catching, and catch layer can be deployed on reader account,
//	//
//	//	How live select architecture can be decomposed?
//	//  - Fast message and reliable message delivery platform
//	//  - Fast change detection
//	//
//	dag := NewDAGBuilder()
//	// Only latest records from database that match live select criteria are used
//	lastState := dag.
//		DoLoad(&GenerateHandler{
//			DoLoad: func(push func(message Item)) error {
//				push(Item{
//					Key: "game-1",
//					Data: schema.FromGo(schemaless.Record[Game]{
//						ID:      "game-1",
//						Version: 3,
//						Data: Game{
//							SessionID: "session-1",
//							Players:   []string{"a", "b"},
//							Winner:    "a",
//						},
//					}),
//				})
//				push(Item{
//					Key: "game-2",
//					Data: schema.FromGo(schemaless.Record[Game]{
//						ID:      "game-2",
//						Version: 3,
//						Data: Game{
//							SessionID: "session-2",
//							Players:   []string{"a", "b"},
//							Winner:    "a",
//						},
//					}),
//				})
//
//				return nil
//			},
//		}, WithName("DynamoDB LastState Filtered"))
//	// Only streamed records that match live select criteria are used
//	streamState := dag.
//		DoLoad(&GenerateHandler{
//			DoLoad: func(push func(message Item)) error {
//				// This is where we would get data from stream
//				push(Item{
//					Key: "game-1",
//					Data: schema.FromGo(schemaless.Record[Game]{
//						ID:      "game-1",
//						Version: 2,
//						Data: Game{
//							SessionID: "session-1",
//							Players:   []string{"a", "b"},
//							Winner:    "a",
//						},
//					}),
//				})
//				return nil
//			},
//		}, WithName("DynamoDB Filtered Stream"))
//	// Joining make sure that newest version is published
//
//	joined := dag.
//		// DoJoin by key, so if db and stream has the same key, then it will be joined.
//		DoJoin(lastState, streamState, WithName("DoJoin")).
//		Window(&FilterHandler{
//			Where: predicate.MustWhere(
//				"Data.SessionID = :sessionID",
//				predicate.ParamBinds{
//					":sessionID": schema.MkString("session-1"),
//				}),
//		}).
//		// Joining by key and producing a new key is like merging!
//		DoWindow(&JoinHandler[schemaless.Record[Game]]{
//			F: func(a, b schemaless.Record[Game], returning func(schemaless.Record[Game])) error {
//				if a.Version < b.Version {
//					returning(b)
//				}
//				return nil
//			},
//		})
//
//	gameStats := joined.
//		Window(Log("gameStats"), WithName("MapGameToStats")).
//		Window(&MapHandler[schemaless.Record[Game], SessionsStats]{
//			F: func(x schemaless.Record[Game], returning func(key string, value SessionsStats)) error {
//				y := x.Data
//				for _, player := range y.Players {
//					wins := 0
//					draws := 0
//					loose := 0
//
//					if y.IsDraw {
//						draws = 1
//					} else if y.Winner == player {
//						wins = 1
//					} else {
//						loose = 1
//					}
//
//					returning("session-stats-by-player:"+player, SessionsStats{
//						Wins:  wins,
//						Draws: draws,
//						Loose: loose,
//					})
//				}
//
//				return nil
//			},
//		})
//
//	gameStatsBySession := gameStats.
//		DoWindow(MergeSessionStats(), WithName("MergeSessionStats"))
//
//	//// Storing in database those updates is like creating materialized view
//	//// For live select this can be skipped.
//	//store := schemaless.NewInMemoryRepository()
//	//gameStatsBySession.
//	//	WithName("Store in database").
//	//	Window(NewRepositorySink("session", store), IgnoreRetractions())
//
//	gameStatsBySession.
//		Window(Log("publish-web-socket"), WithName("Publish to websocket"))
//	//Window(NewWebsocketSink())
//
//	interpretation := DefaultInMemoryInterpreter()
//	err := interpretation.Run(context.Background(), dag.Build())
//	assert.NoError(t, err)
//}

//func TestMergeDifferentInputsTypes(t *testing.T) {
//	log.SetLevel(log.DebugLevel)
//	log.SetFormatter(&log.TextFormatter{
//		ForceColors:     true,
//		TimestampFormat: "",
//		PadLevelText:    true,
//	})
//
//	dag := NewDAGBuilder()
//
//	ints := dag.DoLoad(&GenerateHandler{
//		DoLoad: func(push func(message Item)) error {
//			push(Item{
//				Key:  "int-1",
//				Data: schema.FromGo(1),
//			})
//			return nil
//		},
//	})
//
//	strings := dag.DoLoad(&GenerateHandler{
//		DoLoad: func(push func(message Item)) error {
//			push(Item{
//				Key:  "string-1",
//				Data: schema.FromGo("string-1"),
//			})
//			return nil
//		},
//	})
//
//	_ = dag.
//		// Push to the same channel different keys
//		DoJoin(ints, strings).
//		// Window, don't look at keys, so it can squash them into one
//		Window(&MapHandler[any, any]{
//			F: func(x any, returning func(key string, value any)) error {
//				switch y := x.(type) {
//				case int:
//					returning("key", strconv.Itoa(y))
//				case float64:
//					returning("key", strconv.FormatFloat(y, 'f', -1, 64))
//				case string:
//					returning("key", y)
//				default:
//					return fmt.Errorf("unknown type %T", x)
//				}
//				return nil
//			},
//		}).
//		// DoWindow is always MergeByKey, and since we have only one key, it will merge all incoming data
//		DoWindow(&MergeHandler[string]{
//			Combine: func(a, b string) (string, error) {
//				return a + b, nil
//			},
//		}).
//		//Window(&DebounceHandler{
//		//	MaxSize: 10,
//		//	MaxTime: 10 * time.Millisecond,
//		//}).
//		Window(Log("merged"))
//
//	interpretation := DefaultInMemoryInterpreter()
//	err := interpretation.Run(context.Background(), dag.Build())
//	assert.NoError(t, err)
//}
