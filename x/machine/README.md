# mkunion and state machines
Package models state machines as a union of **states**, and transition functions as a union of **commands**.
Package provides an inferring method to visualize state machines as a mermaid diagram.

## Example
Look into [simple_machine_test.go](../../example/state/simple_machine_test.go) directory for a complete example.

```mermaid
---
title: Canonical question transition
---
stateDiagram
	[*] --> "*state.Candidate": "*state.CreateCandidateCMD"
	"*state.Candidate" --> "*state.Duplicate": "*state.MarkAsDuplicateCMD"
	"*state.Candidate" --> "*state.Canonical": "*state.MarkAsCanonicalCMD"
	"*state.Candidate" --> "*state.Unique": "*state.MarkAsUniqueCMD"
	[*] --> [*]: "❌*state.MarkAsDuplicateCMD"
	"*state.Canonical" --> "*state.Canonical": "❌*state.MarkAsDuplicateCMD"
```