package schema

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"testing"
)

func TestMustDefineUnion_DataRace(t *testing.T) {
	RegisterUnionTypes(
		MustDefineUnion[ABUnion](&AStruct{}, &BStruct{}),
	)

	grouop := errgroup.Group{}
	for i := 0; i < 100; i++ {
		grouop.Go(func() error {
			if rand.Float64() > 0.5 {
				schemed := MkMap(
					MkField("AStruct", MkMap(
						MkField("Foo", MkFloat(123.3)),
					)))
				_, err := ToGo(schemed, WithUnionFormatter(FormatUnionNameUsingTypeName))
				return err
			} else {
				schemed := MkMap(
					MkField("schema.BStruct", MkMap(
						MkField("S", MkString("some string")),
					)))
				_, err := ToGo(schemed, WithUnionFormatter(FormatUnionNameUsingTypeNameWithPackage))
				return err
			}
		})
	}
	if err := grouop.Wait(); err != nil {
		assert.NoError(t, err)
	}
}
