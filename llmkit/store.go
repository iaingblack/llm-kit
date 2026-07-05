package llmkit

import (
	"context"
	"sync"
)

type Store interface {
	GetLLMConfig(context.Context) (Config, error)
	SaveLLMConfig(context.Context, Config) (Config, error)
}

type MemoryStore struct {
	mu  sync.Mutex
	cfg Config
}

func NewMemoryStore(cfg Config) *MemoryStore {
	return &MemoryStore{cfg: NormalizeConfig(cfg)}
}

func (s *MemoryStore) GetLLMConfig(context.Context) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return NormalizeConfig(s.cfg), nil
}

func (s *MemoryStore) SaveLLMConfig(_ context.Context, cfg Config) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = NormalizeConfig(cfg)
	return s.cfg, nil
}
