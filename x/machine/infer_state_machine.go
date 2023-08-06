package machine

import (
	"fmt"
	"reflect"
	"strings"
)

// NewInferTransition creates a helper to infer state machine transitions.
func NewInferTransition[Transition, State any]() *InferTransition[Transition, State] {
	return &InferTransition[Transition, State]{
		exists: make(map[string]bool),
	}
}

type InferTransition[Transition, State any] struct {
	name                 string
	exists               map[string]bool
	showErrorTransitions bool
	transitions          []transition
}

func (t *InferTransition[Transition, State]) WithTitle(name string) *InferTransition[Transition, State] {
	t.name = name
	return t
}

func (t *InferTransition[Transition, State]) WithErrorTransitions(flag bool) *InferTransition[Transition, State] {
	t.showErrorTransitions = flag
	return t
}

type transition [4]string

func (t transition) name() string {
	return t[0]
}

func (t transition) prev() string {
	return t[1]
}

func (t transition) curr() string {
	return t[2]
}

func (t transition) err() string {
	return t[3]
}

func (t transition) String() string {
	return fmt.Sprintf("(%s, %s, %s, %s)", t.name(), t.prev(), t.curr(), t.err())
}

// Record records a transition.
func (t *InferTransition[Transition, State]) Record(tr Transition, prev, curr State, errAfterTransition error) {

	var transitionName, prevStateName, currStateName string = "", "", ""
	if any(tr) != nil {
		transitionName = reflect.TypeOf(tr).String()
	}
	if any(prev) != nil {
		prevStateName = reflect.TypeOf(prev).String()
	}
	if any(curr) != nil {
		currStateName = reflect.TypeOf(curr).String()
	}

	err := ""
	if errAfterTransition != nil {
		err = errAfterTransition.Error()
	}

	tt := transition{
		transitionName,
		prevStateName,
		currStateName,
		err,
	}

	if t.exists[tt.String()] {
		return
	}
	t.exists[tt.String()] = true

	t.transitions = append(t.transitions, tt)
}

// ToMermaid returns a string in Mermaid format.
// https://mermaid-js.github.io/mermaid/#/stateDiagram
func (t *InferTransition[Transition, State]) ToMermaid() string {
	result := &strings.Builder{}
	if t.name != "" {
		fmt.Fprintf(result, "---\ntitle: %s\n---\n", t.name)
	}

	fmt.Fprint(result, "stateDiagram\n")
	for _, tt := range t.transitions {
		prev := tt.prev()
		if prev == "" {
			prev = "[*]"
		} else {
			prev = `"` + prev + `"`
		}
		curr := tt.curr()
		if curr == "" {
			curr = "[*]"
		} else {
			curr = `"` + curr + `"`
		}

		name := tt.name()
		if tt.err() != "" {
			fmt.Fprintf(result, " %%%% error=%s \n", strings.TrimSpace(strings.ReplaceAll(tt.err(), "\n", " ")))
			name = fmt.Sprintf("❌%s", name)
		}

		if tt.err() != "" && !t.showErrorTransitions {
			continue
		}

		fmt.Fprintf(result, "\t"+`%s --> %s: "%s"`+"\n", prev, curr, name)
	}

	return result.String()
}
