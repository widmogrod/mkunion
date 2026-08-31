package predicate

func Optimize(p Predicate) Predicate {
	return MatchPredicateR1(
		p,
		func(x *And) Predicate {
			l := make([]Predicate, 0, len(x.L))
			for _, c := range x.L {
				l = append(l, Optimize(c))
			}
			// flatten nested predicates
			if len(l) == 1 {
				return l[0]
			}
			return &And{L: l}
		},
		func(x *Or) Predicate {
			l := make([]Predicate, 0, len(x.L))
			for _, c := range x.L {
				l = append(l, Optimize(c))
			}
			// flatten nested predicates
			if len(l) == 1 {
				return l[0]
			}
			return &Or{L: l}
		},
		func(x *Not) Predicate {
			inner := Optimize(x.P)
			if y, ok := inner.(*Not); ok {
				// double negation is the same as the original
				// !(!x) == x
				return y.P
			}
			return &Not{P: inner}
		},
		func(x *Compare) Predicate {
			return x
		},
	)
}
