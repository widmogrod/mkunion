package projection

import "sync"

type StatsCollector interface {
	Snapshot() Stats
	Incr(key string, increment int)
}

func NewStatsCollector() *statsCollector {
	return &statsCollector{
		stats: Stats{},
	}
}

var _ StatsCollector = (*statsCollector)(nil)

type statsCollector struct {
	lock  sync.Mutex
	stats Stats
}

func (s *statsCollector) Snapshot() Stats {
	s.lock.Lock()
	defer s.lock.Unlock()

	result := s.stats
	s.stats = Stats{}

	return result
}

func (s *statsCollector) Incr(key string, increment int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stats[key] += increment
}
