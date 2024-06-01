package projection

//go:tag serde:"json"
type Game struct {
	SessionID string
	Players   []string
	Winner    string
	IsDraw    bool
}

//go:tag serde:"json"
type SessionsStats struct {
	Wins  int
	Draws int
	Loose int
}
