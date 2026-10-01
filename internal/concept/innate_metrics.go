package concept

import (
	"sync"
	"time"
)

// Opt-in internal qualification data. No metrics or clock reads are collected
// by ordinary compilation, and the shared evaluator lock remains unchanged.
type evt1ComptimeUsage struct {
	Fuel, Depth, Loop, Array int
}

type evt1InnateSample struct {
	Module, Predicate, Subject string
	Usage                      evt1ComptimeUsage
	Wait, Execution            time.Duration
}

type evt1InnateMetrics struct {
	mu      sync.Mutex
	samples []evt1InnateSample
}

func (m *evt1InnateMetrics) record(sample evt1InnateSample) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.samples = append(m.samples, sample)
}

func (m *evt1InnateMetrics) snapshot() []evt1InnateSample {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]evt1InnateSample(nil), m.samples...)
}
