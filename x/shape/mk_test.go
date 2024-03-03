package shape

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestMkRefNameFromString(t *testing.T) {
	useCases := []struct {
		name string
		want *RefName
	}{
		{
			name: "string",
			want: &RefName{
				Name: "string",
			},
		},
		{
			name: "github.com/widmogrod/mkunion.Record[string]",
			want: &RefName{
				Name:          "Record",
				PkgName:       "mkunion",
				PkgImportName: "github.com/widmogrod/mkunion",
				Indexed: []Shape{
					&RefName{
						Name: "string",
					},
				},
			},
		},
		{
			name: "github.com/widmogrod/mkunion.Record[string,int]",
			want: &RefName{
				Name:          "Record",
				PkgName:       "mkunion",
				PkgImportName: "github.com/widmogrod/mkunion",
				Indexed: []Shape{
					&RefName{
						Name: "string",
					},
					&RefName{
						Name: "int",
					},
				},
			},
		},
		{
			name: "github.com/widmogrod/mkunion.Record[string,github.com/widmogrod/mkunion.Record[bool,int]]",
			want: &RefName{
				Name:          "Record",
				PkgName:       "mkunion",
				PkgImportName: "github.com/widmogrod/mkunion",
				Indexed: []Shape{
					&RefName{
						Name: "string",
					},
					&RefName{
						Name:          "Record",
						PkgName:       "mkunion",
						PkgImportName: "github.com/widmogrod/mkunion",
						Indexed: []Shape{
							&RefName{
								Name: "bool",
							},
							&RefName{
								Name: "int",
							},
						},
					},
				},
			},
		},
		{
			name: "projection.Record[github.com/widmogrod/mkunion/x/projection.Either[int,float64]]",
			want: &RefName{
				Name:          "Record",
				PkgName:       "projection",
				PkgImportName: "projection",
				Indexed: []Shape{
					&RefName{
						Name:          "Either",
						PkgName:       "projection",
						PkgImportName: "github.com/widmogrod/mkunion/x/projection",
						Indexed: []Shape{
							&RefName{
								Name: "int",
							},
							&RefName{
								Name: "float64",
							},
						},
					},
				},
			},
		},
	}

	for _, useCase := range useCases {
		t.Run(useCase.name, func(t *testing.T) {
			got := MkRefNameFromString(useCase.name)
			if diff := cmp.Diff(useCase.want, got); diff != "" {
				t.Fatalf("MkRefNameFromReflect2: diff: (-want +got)\n%s", diff)
			}
		})
	}
}

type someOf2[T1, T2 any] struct{}

func TestMkRefNameFromReflect(t *testing.T) {
	subject := someOf2[string, someOf2[int, float64]]{}
	got := MkRefNameFromReflect(reflect.TypeOf(subject))
	want := &RefName{
		Name:          "someOf2",
		PkgName:       "shape",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape",
		Indexed: []Shape{
			&RefName{
				Name: "string",
			},
			&RefName{
				Name:          "someOf2",
				PkgName:       "shape",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape",
				Indexed: []Shape{
					&RefName{
						Name: "int",
					},
					&RefName{
						Name: "float64",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("MkRefNameFromReflect: diff: (-want +got)\n%s", diff)
	}

	typeName := ToGoTypeName(got, WithPkgImportName())
	assert.Equal(t,
		"github.com/widmogrod/mkunion/x/shape.someOf2[string,someOf2[int,float64]]",
		typeName,
	)
}
