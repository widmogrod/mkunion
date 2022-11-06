package state

//go:generate go run ../../cmd/mkunion/main.go --name=Transition --types=CreateCandidate,MarkAsCanonical,MarkAsDuplicate,MarkAsUnique
type (
	CreateCandidate struct {
		ID ID
	}
	MarkAsCanonical struct{}
	MarkAsDuplicate struct{ CanonicalID ID }
	MarkAsUnique    struct{}
)
