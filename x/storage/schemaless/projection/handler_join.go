package projection

var _ Handler = &JoinHandler[any]{}

type JoinHandler[T any] struct {
	F func(a, b T, returning func(T)) error
}

func (j *JoinHandler[T]) Process(x Item, returning func(Item)) error {
	panic("not implemented")
	//var result T
	//var resultSet = false
	//var first bool = true
	//var err error
	////Each(x.Data, func(value schema.Schema) {
	//	var elem T
	//	if err != nil {
	//		return
	//	}
	//
	//	elem, err = schema.ToGoG[T](value)
	//	if err != nil {
	//		return
	//	}
	//
	//	if first {
	//		first = false
	//		result = elem
	//		return
	//	}
	//
	//	err = j.F(result, elem, func(t T) {
	//		resultSet = true
	//		result = t
	//	})
	//	if err != nil {
	//		return
	//	}
	//})
	//
	//if err != nil {
	//	d, err2 := schema.ToJSON(x.Data)
	//	return fmt.Errorf("mergeHandler:Process(%s, err=%s) err %s", string(d), err, err2)
	//}
	//
	//if resultSet {
	//	returning(Item{
	//		Key:  x.Key,
	//		Data: schema.FromGo(result),
	//	})
	//}

	return nil
}

func (j *JoinHandler[T]) Retract(x Item, returning func(Item)) error {
	panic("not implemented")
	//return j.Process(x, returning)

}
