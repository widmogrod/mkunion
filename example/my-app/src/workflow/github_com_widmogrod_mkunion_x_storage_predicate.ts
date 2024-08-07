//generated by mkunion
export type WherePredicates = {
	Predicate?: Predicate,
	Params?: ParamBinds,
	Shape?: shape.Shape,
}

export type Predicate = {
	"$type"?: "predicate.And",
	"predicate.And": And
} | {
	"$type"?: "predicate.Or",
	"predicate.Or": Or
} | {
	"$type"?: "predicate.Not",
	"predicate.Not": Not
} | {
	"$type"?: "predicate.Compare",
	"predicate.Compare": Compare
}

export type And = {
	L?: Predicate[],
}

export type Or = {
	L?: Predicate[],
}

export type Not = {
	P?: Predicate,
}

export type Compare = {
	Location?: string,
	Operation?: string,
	BindValue?: Bindable,
}

export type Bindable = {
	"$type"?: "predicate.BindValue",
	"predicate.BindValue": BindValue
} | {
	"$type"?: "predicate.Literal",
	"predicate.Literal": Literal
} | {
	"$type"?: "predicate.Locatable",
	"predicate.Locatable": Locatable
}

export type BindValue = {
	BindName?: BindName,
}

export type BindName = string

export type Literal = {
	Value?: schema.Schema,
}

export type Locatable = {
	Location?: string,
}

export type ParamBinds = {[key: BindName]: schema.Schema}


//eslint-disable-next-line
import * as schema from './github_com_widmogrod_mkunion_x_schema'
//eslint-disable-next-line
import * as shape from './github_com_widmogrod_mkunion_x_shape'
