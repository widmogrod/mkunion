package projection

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
	"time"
)

func TestDabBuilderTest(t *testing.T) {
	dag := NewDAGBuilder()
	found, err := dag.GetByName("a")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, found)

	//found, err = dag.GetByName("root")
	//assert.NoError(t, err)
	//assert.Equal(t, dag, found)

	log := &LogHandler{}
	//m := &LogHandler{}

	/*
		mermaid
		graph TD
			a[DoLoad]
			b[Window]
			c[DoLoad]
			d[Window]
			e[DoJoin]
			f[Window]
			a --> b
			c --> d
			b --> e
			d --> e
			e --> f
	*/
	mapped1 := dag.
		Load(log, WithName("a")).
		Window(WithName("b"))

	mapped2 := dag.
		Load(log, WithName("c")).
		Window(WithName("d"))

	dag.
		Join(mapped1, mapped2, WithName("e")).
		Window(WithName("f"))

	found, err = dag.GetByName("a")
	assert.NoError(t, err)
	assert.Equal(t, log, found.dag.(*DoLoad).OnLoad)

	found, err = dag.GetByName("b")
	assert.NoError(t, err)
	//assert.Equal(t, m, found.dag.(*DoWindow).OnMap)

	nodes := dag.Build()
	assert.Equal(t, 6, len(nodes))

	//assert.Equal(t, "a", GetCtx(nodesFromTo[0]).Name())
	//assert.Equal(t, "b", GetCtx(nodesFromTo[1]).Name())

	fmt.Println(ToMermaidGraph(dag))

	fmt.Println(ToMermaidGraphWithOrder(dag, ReverseSort(Sort(dag))))
}

func TestNewContextBuilder(t *testing.T) {
	useCases := map[string]struct {
		in  *DefaultContext
		out *DefaultContext
	}{
		"should set default values": {
			in: NewContextBuilder(),
			out: &DefaultContext{
				wd: &FixedWindow{
					// infinite window,
					Width: math.MaxInt64,
				},
				td: &AtWatermark{},
				fm: &Discard{},
			},
		},
		"should set window duration": {
			in: NewContextBuilder(
				WithFixedWindow(100*time.Millisecond),
				WithTriggers(&AtPeriod{Duration: 10 * time.Millisecond}, &AtWatermark{}),
				WithAccumulatingAndRetracting(),
			),
			out: &DefaultContext{
				wd: &FixedWindow{
					Width: 100 * time.Millisecond,
				},
				td: &AllOf{
					Triggers: []TriggerDescription{
						&AtPeriod{Duration: 10 * time.Millisecond},
						&AtWatermark{},
					},
				},
				fm: &AccumulatingAndRetracting{},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.out, uc.in)
		})
	}

}
