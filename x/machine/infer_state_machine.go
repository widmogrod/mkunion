package machine

import (
	"fmt"
	"reflect"
	"sort"
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

	// map only transitions with names
	if transitionName == "" {
		return
	}

	tt := transition{
		transitionName,
		prevStateName,
		currStateName,
		err,
	}

	name := tt.String()
	_ = name
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

	// sort transitions by name
	sort.SliceStable(t.transitions, func(i, j int) bool {
		return t.transitions[i].String() < t.transitions[j].String()
	})

	if t.name != "" {
		fmt.Fprintf(result, "---\ntitle: %s\n---\n", t.name)
	}

	fmt.Fprint(result, "stateDiagram\n")

	// Collect unique state names to create aliases
	stateAliases := make(map[string]string)
	for _, tt := range t.transitions {
		if prev := tt.prev(); prev != "" {
			if _, exists := stateAliases[prev]; !exists {
				// Create alias by removing special characters
				alias := createAlias(prev)
				stateAliases[prev] = alias
			}
		}
		if curr := tt.curr(); curr != "" {
			if _, exists := stateAliases[curr]; !exists {
				// Create alias by removing special characters
				alias := createAlias(curr)
				stateAliases[curr] = alias
			}
		}
	}

	// Write state aliases first (sorted for consistency)
	var stateNames []string
	for stateName := range stateAliases {
		stateNames = append(stateNames, stateName)
	}
	sort.Strings(stateNames)
	for _, stateName := range stateNames {
		fmt.Fprintf(result, "\t%s: %s\n", stateAliases[stateName], stateName)
	}
	if len(stateAliases) > 0 {
		fmt.Fprint(result, "\n") // Add empty line after aliases
	}

	// Write transitions using aliases
	for _, tt := range t.transitions {
		prev := tt.prev()
		if prev == "" {
			prev = "[*]"
		} else {
			prev = stateAliases[prev]
		}
		curr := tt.curr()
		if curr == "" {
			curr = "[*]"
		} else {
			curr = stateAliases[curr]
		}

		name := tt.name()
		if tt.err() != "" {
			if t.showErrorTransitions {
				fmt.Fprintf(result, "\t%%%% error=%s \n", strings.TrimSpace(strings.ReplaceAll(tt.err(), "\n", " ")))
				name = fmt.Sprintf("❌%s", name)
			} else {
				continue
			}
		}

		fmt.Fprintf(result, "\t%s --> %s: %s\n", prev, curr, name)
	}

	return result.String()
}

// createAlias creates a valid mermaid identifier from a state name
func createAlias(stateName string) string {
	if stateName == "" {
		return ""
	}

	// Remove pointer indicator
	alias := strings.TrimPrefix(stateName, "*")

	// Replace dots and slashes with underscores for valid mermaid identifiers
	alias = strings.ReplaceAll(alias, ".", "_")
	alias = strings.ReplaceAll(alias, "/", "_")

	// Simple approach: just return the cleaned up full path
	// This ensures uniqueness for all cases
	return alias
}

// ParsedTransition represents a transition parsed from a mermaid diagram
type ParsedTransition struct {
	FromState string
	ToState   string
	Command   string
	IsError   bool
}

// ParseMermaid parses a mermaid diagram and extracts transitions
func ParseMermaid(mermaidContent string) ([]ParsedTransition, error) {
	transitions := []ParsedTransition{}
	lines := strings.Split(mermaidContent, "\n")

	// Maps to store state aliases to full names
	stateAliases := make(map[string]string)

	// Flag to track if we're inside the stateDiagram section
	inStateDiagram := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// Check for stateDiagram marker
		if line == "stateDiagram" || line == "stateDiagram-v2" {
			inStateDiagram = true
			continue
		}

		// Skip lines until we find stateDiagram marker
		if !inStateDiagram {
			continue
		}

		// Parse state alias definitions (e.g., "State1: *example.StateType")
		if strings.Contains(line, ": ") && !strings.Contains(line, "-->") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				alias := strings.TrimSpace(parts[0])
				fullName := strings.TrimSpace(parts[1])
				stateAliases[alias] = fullName
			}
			continue
		}

		// Parse transitions (e.g., "State1 --> State2: Command")
		if strings.Contains(line, "-->") {
			// Check if this is an error transition
			isError := false
			if strings.Contains(line, "❌") {
				isError = true
			}

			// Split into from, to, and command parts
			arrowParts := strings.Split(line, "-->")
			if len(arrowParts) != 2 {
				continue
			}

			fromState := strings.TrimSpace(arrowParts[0])

			// Split the right side by colon to get state and command
			colonParts := strings.SplitN(arrowParts[1], ":", 2)
			if len(colonParts) != 2 {
				continue
			}

			toState := strings.TrimSpace(colonParts[0])
			command := strings.TrimSpace(colonParts[1])

			// Remove error indicator from command if present
			command = strings.TrimPrefix(command, "❌")

			// Resolve aliases to full state names
			if fromState != "[*]" {
				if fullName, ok := stateAliases[fromState]; ok {
					fromState = fullName
				}
			} else {
				fromState = ""
			}

			if toState != "[*]" {
				if fullName, ok := stateAliases[toState]; ok {
					toState = fullName
				}
			} else {
				toState = ""
			}

			transitions = append(transitions, ParsedTransition{
				FromState: fromState,
				ToState:   toState,
				Command:   command,
				IsError:   isError,
			})
		}
	}

	return transitions, nil
}
