// Code generated by mkunion. DO NOT EDIT.
package workflow

import (
	"github.com/widmogrod/mkunion/x/shape"
)

func init() {
	shape.Register(ToStrContextShape())
	shape.Register(ToStrErrInfoShape())
}

//shape:shape
func ToStrContextShape() shape.Shape {
	return &shape.StructLike{
		Name:          "ToStrContext",
		PkgName:       "workflow",
		PkgImportName: "github.com/widmogrod/mkunion/x/workflow",
		Fields: []*shape.FieldLike{
			{
				Name: "Errors",
				Type: &shape.MapLike{
					Key: &shape.RefName{
						Name:          "StepID",
						PkgName:       "workflow",
						PkgImportName: "github.com/widmogrod/mkunion/x/workflow",
					},
					Val: &shape.RefName{
						Name:          "ToStrErrInfo",
						PkgName:       "workflow",
						PkgImportName: "github.com/widmogrod/mkunion/x/workflow",
					},
				},
			},
		},
	}
}

//shape:shape
func ToStrErrInfoShape() shape.Shape {
	return &shape.StructLike{
		Name:          "ToStrErrInfo",
		PkgName:       "workflow",
		PkgImportName: "github.com/widmogrod/mkunion/x/workflow",
		Fields: []*shape.FieldLike{
			{
				Name: "StepID",
				Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
			},
			{
				Name: "Code",
				Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
			},
			{
				Name: "Message",
				Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
			},
		},
	}
}
