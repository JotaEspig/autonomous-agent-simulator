// Package rng provides counter-based random streams. Every draw is keyed by
// (seed, stream, tick, entity), so results never depend on how work is split
// between goroutines or in which order agents are processed.
package rng

import "math/rand/v2"

// Stream identifiers separate the purposes randomness is used for, so adding a
// draw in one place never shifts the numbers seen in another.
const (
	StreamPopulation uint64 = iota + 1
	StreamActivation
	StreamDecision
	StreamOrder
	StreamRule
	StreamMetric
	StreamStop
)

// Source is a reusable generator. It is not safe for concurrent use; keep one per worker.
type Source struct {
	pcg rand.PCG
	r   *rand.Rand
}

func New() *Source {
	s := &Source{}
	s.r = rand.New(&s.pcg)
	return s
}

// mix is the SplitMix64 finalizer, a fast bijective hash with good avalanche.
func mix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// Reset positions the source at the start of the stream identified by its coordinates.
func (s *Source) Reset(seed, stream, tick, entity uint64) {
	a := mix(seed ^ mix(stream))
	b := mix(tick ^ mix(entity^a))
	s.pcg.Seed(a, b)
}

// Hash maps coordinates to a uniformly distributed 64-bit value without a Source.
func Hash(seed, stream, entity uint64) uint64 {
	return mix(mix(seed^mix(stream)) ^ entity)
}

func (s *Source) Float() float64                     { return s.r.Float64() }
func (s *Source) IntN(n int) int                     { return s.r.IntN(n) }
func (s *Source) Norm() float64                      { return s.r.NormFloat64() }
func (s *Source) Shuffle(n int, swap func(i, j int)) { s.r.Shuffle(n, swap) }
