package shape

import "sync/atomic"

// Counters that describe how much work shape inference does.
// They help to quantify performance changes (see MKUNION_STATS=1).
var (
	statFileParses   atomic.Int64
	statFileParseHit atomic.Int64
	statPkgWalks     atomic.Int64
	statPkgWalksHit  atomic.Int64
)

// StatsSnapshot returns counters that describe how much work shape inference did so far.
func StatsSnapshot() map[string]int64 {
	return map[string]int64{
		"file_parses":     statFileParses.Load(),
		"file_parse_hits": statFileParseHit.Load(),
		"pkg_walks":       statPkgWalks.Load(),
		"pkg_walk_hits":   statPkgWalksHit.Load(),
	}
}
