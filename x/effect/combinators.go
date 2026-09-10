package effect

// Programs are values, so they compose with functions. Then and Map (eff.go)
// continue on success. Catch continues on failure. Together with Return and
// Throw they are enough to build retries, fallbacks and waits as programs,
// with no body and no middleware. See example/compose/billing.

// --8<-- [start:catch]

// Catch runs e and, when it fails, runs handle with the error instead.
// A success passes through untouched. It is Then for the failure side.
func Catch[Op, A any](e Eff[Op, A], handle func(error) Eff[Op, A]) Eff[Op, A] {
	return MatchEffR1(e,
		func(x *Pure[Op, A]) Eff[Op, A] { return x },
		func(x *Fail[Op, A]) Eff[Op, A] { return handle(x.Err) },
		func(x *Bind[Op, A]) Eff[Op, A] {
			return &Bind[Op, A]{
				Op:   x.Op,
				Cont: func(answer any, err error) Eff[Op, A] { return Catch(x.Cont(answer, err), handle) },
			}
		},
		func(x *Suspend[Op, A]) Eff[Op, A] {
			return &Suspend[Op, A]{Resume: func() Eff[Op, A] { return Catch(x.Resume(), handle) }}
		},
	)
}

// OrElse runs e and, when it fails, runs fallback. The error is dropped.
func OrElse[Op, A any](e, fallback Eff[Op, A]) Eff[Op, A] {
	return Catch(e, func(error) Eff[Op, A] { return fallback })
}

// --8<-- [end:catch]
