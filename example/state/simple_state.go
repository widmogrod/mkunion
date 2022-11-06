package state

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
