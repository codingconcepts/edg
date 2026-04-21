package seq

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"

	"github.com/codingconcepts/edg/pkg/random"
)

type Config struct {
	Name  string `json:"name" yaml:"name"`
	Start int64  `json:"start" yaml:"start"`
	Step  int64  `json:"step" yaml:"step"`
}

type sequence struct {
	start   int64
	step    int64
	counter atomic.Int64
}

func (s *sequence) Next() int64 {
	c := s.counter.Add(1) - 1
	return s.start + c*s.step
}

func (s *sequence) value(idx int64) int64 {
	return s.start + idx*s.step
}

func (s *sequence) count() int64 {
	return s.counter.Load()
}

func (s *sequence) Rand() (int64, error) {
	n := s.count()
	if n == 0 {
		return 0, fmt.Errorf("sequence has no values yet")
	}
	return s.value(int64(random.Rng.IntN(int(n)))), nil
}

func (s *sequence) Zipf(sv, v float64) (int64, error) {
	n := s.count()
	if n == 0 {
		return 0, fmt.Errorf("sequence has no values yet")
	}
	src := rand.NewPCG(random.Rng.Uint64(), random.Rng.Uint64())
	r := rand.New(src)
	z := rand.NewZipf(r, sv, v, uint64(n-1))
	if z == nil {
		return 0, fmt.Errorf("zipf: invalid parameters s=%g v=%g (requires s > 1 and v >= 1)", sv, v)
	}
	return s.value(int64(z.Uint64())), nil
}

func (s *sequence) Norm(mean, stddev float64) (int64, error) {
	n := s.count()
	if n == 0 {
		return 0, fmt.Errorf("sequence has no values yet")
	}
	idx, err := random.Norm(mean, stddev, 0, float64(n-1))
	if err != nil {
		return 0, err
	}
	return s.value(int64(idx)), nil
}

func (s *sequence) Exp(rate float64) (int64, error) {
	n := s.count()
	if n == 0 {
		return 0, fmt.Errorf("sequence has no values yet")
	}
	idx, err := random.Exp(rate, 0, float64(n-1))
	if err != nil {
		return 0, err
	}
	return s.value(int64(idx)), nil
}

func (s *sequence) Lognorm(mu, sigma float64) (int64, error) {
	n := s.count()
	if n == 0 {
		return 0, fmt.Errorf("sequence has no values yet")
	}
	idx, err := random.LogNorm(mu, sigma, 0, float64(n-1))
	if err != nil {
		return 0, err
	}
	return s.value(int64(idx)), nil
}

type Manager struct {
	seqs map[string]*sequence
}

func NewManager(defs []Config) *Manager {
	m := &Manager{seqs: make(map[string]*sequence, len(defs))}
	for _, d := range defs {
		m.seqs[d.Name] = &sequence{start: d.Start, step: d.Step}
	}
	return m
}

// HasSequence reports whether a sequence with the given name is defined.
func (m *Manager) HasSequence(name string) bool {
	_, ok := m.seqs[name]
	return ok
}

func (m *Manager) Next(name string) (int64, error) {
	s, ok := m.seqs[name]
	if !ok {
		return 0, fmt.Errorf("seq_global: unknown sequence %q", name)
	}
	return s.Next(), nil
}

func (m *Manager) Rand(name string) (int64, error) {
	s, ok := m.seqs[name]
	if !ok {
		return 0, fmt.Errorf("seq_rand: unknown sequence %q", name)
	}
	return s.Rand()
}

func (m *Manager) Zipf(name string, sv, v float64) (int64, error) {
	s, ok := m.seqs[name]
	if !ok {
		return 0, fmt.Errorf("seq_zipf: unknown sequence %q", name)
	}
	return s.Zipf(sv, v)
}

func (m *Manager) Norm(name string, mean, stddev float64) (int64, error) {
	s, ok := m.seqs[name]
	if !ok {
		return 0, fmt.Errorf("seq_norm: unknown sequence %q", name)
	}
	return s.Norm(mean, stddev)
}

func (m *Manager) Exp(name string, rate float64) (int64, error) {
	s, ok := m.seqs[name]
	if !ok {
		return 0, fmt.Errorf("seq_exp: unknown sequence %q", name)
	}
	return s.Exp(rate)
}

func (m *Manager) Lognorm(name string, mu, sigma float64) (int64, error) {
	s, ok := m.seqs[name]
	if !ok {
		return 0, fmt.Errorf("seq_lognorm: unknown sequence %q", name)
	}
	return s.Lognorm(mu, sigma)
}
