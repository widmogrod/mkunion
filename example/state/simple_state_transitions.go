package state

type (
	ID   = string
	Attr = map[string]any
)

//go:generate go run ../../cmd/mkunion/main.go --name=State --types=Candidate,Canonical,Duplicate,Unique
type (
	Candidate struct {
		ID         ID
		Attributes Attr
	}
	Canonical struct {
		ID ID
	}
	Duplicate struct {
		ID          ID
		CanonicalID ID
	}
	Unique struct {
		ID ID
	}
)

//go:generate go run ../../cmd/mkunion/main.go --name=Transition --types=CreateCandidate,MarkAsCanonical,MarkAsDuplicate,MarkAsUnique
type (
	CreateCandidate struct {
		ID ID
	}
	MarkAsCanonical struct{}
	MarkAsDuplicate struct{ CanonicalID ID }
	MarkAsUnique    struct{}
)
