package state

type (
	ID   = string
	Attr = map[string]any
)

//go:generate go run ../../cmd/mkunion/main.go --name=Command
type (
	CreateCandidateCMD struct {
		ID ID
	}
	MarkAsCanonicalCMD struct{}
	MarkAsDuplicateCMD struct{ CanonicalID ID }
	MarkAsUniqueCMD    struct{}
)

//go:generate go run ../../cmd/mkunion/main.go --name=State
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
