package predicate

func Optimize(p Predicate) Predicate {
	return MatchPredicateR1(
		p,
		func(x *And) Predicate {
			// flatten nested predicates
			if len(x.L) == 1 {
				return x.L[0]
			}
			return x
		},
		func(x *Or) Predicate {
			// flatten nested predicates
			if len(x.L) == 1 {
				return x.L[0]
			}
			return x
		},
		func(x *Not) Predicate {
			y, ok := x.P.(*Not)
			if ok {
				// double negation is the same as the original
				// !(!x) == x
				return Optimize(y.P)
			}
			return x
		},
		func(x *Compare) Predicate {
			return x
		},
	)
}
