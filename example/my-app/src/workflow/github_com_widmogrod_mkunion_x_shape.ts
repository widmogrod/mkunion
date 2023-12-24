//generated by mkunion
export type Shape = {
	"$type"?: "shape.Any",
	"shape.Any": Any
} | {
	"$type"?: "shape.RefName",
	"shape.RefName": RefName
} | {
	"$type"?: "shape.AliasLike",
	"shape.AliasLike": AliasLike
} | {
	"$type"?: "shape.BooleanLike",
	"shape.BooleanLike": BooleanLike
} | {
	"$type"?: "shape.StringLike",
	"shape.StringLike": StringLike
} | {
	"$type"?: "shape.NumberLike",
	"shape.NumberLike": NumberLike
} | {
	"$type"?: "shape.ListLike",
	"shape.ListLike": ListLike
} | {
	"$type"?: "shape.MapLike",
	"shape.MapLike": MapLike
} | {
	"$type"?: "shape.StructLike",
	"shape.StructLike": StructLike
} | {
	"$type"?: "shape.UnionLike",
	"shape.UnionLike": UnionLike
}

export type Any = {}

export type RefName = {
	Name?: string,
	PkgName?: string,
	PkgImportName?: string,
	IsPointer?: boolean,
	Indexed?: Shape[],
}

export type AliasLike = {
	Name?: string,
	PkgName?: string,
	PkgImportName?: string,
	IsAlias?: boolean,
	Type?: Shape,
	Tags?: {[key: string]: Tag},
}

export type BooleanLike = {}

export type StringLike = {}

export type NumberLike = {
	Kind?: NumberKind,
}

export type ListLike = {
	Element?: Shape,
	ElementIsPointer?: boolean,
	ArrayLen?: number,
}

export type MapLike = {
	Key?: Shape,
	Val?: Shape,
	KeyIsPointer?: boolean,
	ValIsPointer?: boolean,
}

export type StructLike = {
	Name?: string,
	PkgName?: string,
	PkgImportName?: string,
	TypeParams?: TypeParam[],
	Fields?: FieldLike[],
	Tags?: {[key: string]: Tag},
}

export type UnionLike = {
	Name?: string,
	PkgName?: string,
	PkgImportName?: string,
	Variant?: Shape[],
	Tags?: {[key: string]: Tag},
}

export type NumberKind = {
	"$type"?: "shape.UInt8",
	"shape.UInt8": UInt8
} | {
	"$type"?: "shape.UInt16",
	"shape.UInt16": UInt16
} | {
	"$type"?: "shape.UInt32",
	"shape.UInt32": UInt32
} | {
	"$type"?: "shape.UInt64",
	"shape.UInt64": UInt64
} | {
	"$type"?: "shape.Int8",
	"shape.Int8": Int8
} | {
	"$type"?: "shape.Int16",
	"shape.Int16": Int16
} | {
	"$type"?: "shape.Int32",
	"shape.Int32": Int32
} | {
	"$type"?: "shape.Int64",
	"shape.Int64": Int64
} | {
	"$type"?: "shape.Float32",
	"shape.Float32": Float32
} | {
	"$type"?: "shape.Float64",
	"shape.Float64": Float64
}

export type UInt8 = {}

export type UInt16 = {}

export type UInt32 = {}

export type UInt64 = {}

export type Int8 = {}

export type Int16 = {}

export type Int32 = {}

export type Int64 = {}

export type Float32 = {}

export type Float64 = {}

export type Guard = {
	"$type"?: "shape.Enum",
	"shape.Enum": Enum
} | {
	"$type"?: "shape.Required",
	"shape.Required": Required
} | {
	"$type"?: "shape.AndGuard",
	"shape.AndGuard": AndGuard
}

export type Enum = {
	Val?: string[],
}

export type Required = {}

export type AndGuard = {
	L?: Guard[],
}

export type TypeParam = {
	Name?: string,
	Type?: Shape,
}
export type Tag = {
	Value?: string,
	Options?: string[],
}
export type FieldLike = {
	Name?: string,
	Type?: Shape,
	Desc?: string,
	Guard?: Guard,
	IsPointer?: boolean,
	Tags?: {[key: string]: Tag},
}
