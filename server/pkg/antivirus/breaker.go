package antivirus

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	mu        sync.Mutex
	threshold int
	window    time.Duration
	openFor   time.Duration
	failures  []time.Time
	openUntil time.Time
}

func NewCircuitBreaker(threshold int, window, openFor time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	if window <= 0 {
		window = 30 * time.Second
	}
	if openFor <= 0 {
		openFor = 10 * time.Second
	}
	return &CircuitBreaker{
		threshold: threshold,
		window:    window,
		openFor:   openFor,
	}
}

func (b *CircuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return time.Now().After(b.openUntil)
}

func (b *CircuitBreaker) Record(status Status) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if status != StatusUnavailable {
		b.failures = nil
		return
	}

	now := time.Now()
	cutoff := now.Add(-b.window)
	kept := b.failures[:0]
	for _, ts := range b.failures {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	b.failures = append(kept, now)
	if len(b.failures) >= b.threshold {
		b.openUntil = now.Add(b.openFor)
		b.failures = nil
	}
}
