// Package effect is a small effect system: programs are data, handlers give
// them meaning.
//
// A program is an Eff[Op, A]: it asks for operations of type Op and yields an
// A. It performs nothing by itself. Run walks it with a Handler[Op], which
// answers each operation: the real world in production, a fake in tests, a
// tape in a replay. Because every operation passes through one function, one
// Middleware sees every operation of every program, in order, as data: Trace,
// Record and Replay, Retry, StepKeys, Guard, Spans, fault injection, Chaos.
//
// Op is any type, usually a mkunion union whose variants embed f.Returns, so that
// operations and handlers are typed per variant. This package does not know
// any particular union; example/welcome is the walkthrough, and
// example/compose shows two unions from two packages in one program.
package effect
