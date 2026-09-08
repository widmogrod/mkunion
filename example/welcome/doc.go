// Package effect is a small effect system built from mkunion unions.
//
// The idea in one line: a program is data, and a handler gives that data meaning.
// A program asks for operations such as "read this file" or "send this mail".
// It never performs them. A handler performs them: the real one in production,
// a fake one in tests, a recording one when you want a tape to replay.
//
// Read it in this order. Each file builds on the one before, and each part has
// a test file with the same number that shows the behaviour with real data.
//
//	Part 1: the basics (start here)
//	  ops.go        the operations, as a union, each with its answer type in f.Returns
//	                (the typed handler is generated from it, see ops_union_gen.go)
//	  program.go    programs written as plain Go against Fx
//	  handlers.go   Live for production, Fake and Defaults for tests, Mailbox
//	  part1_basics_test.go
//
//	Part 2: under the hood (the core lives in x/effect)
//	  x/effect/eff.go    the program union (Pure, Fail, Bind, Suspend), Then, Run
//	  x/effect/proc.go   the coroutine that turns a plain body into that union
//	  part2_under_the_hood_test.go
//
//	Part 3: testing with tapes
//	  x/effect/recording.go   Record, Replay
//	  recording_json.go       a tape as JSON, typed per operation
//	  part3_testing_test.go
//
//	Part 4: failure is normal
//	  x/effect/middleware.go   retry policies, idempotency keys, fault injection, chaos
//	  part4_failures_test.go
//
//	Part 5: seeing what happened
//	  x/effect/observe.go   trace diff, policy guard, spans
//	  part5_observability_test.go
//
// The docs page docs/examples/effect_system.md walks through the same order.
//
// The type registry is off for this package, for the reasons given in x/effect/doc.go.
//
//go:tag mkunion:",no-type-registry"
package welcome
