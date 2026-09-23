package store

import (
	"context"
	"errors"
	"sync"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

var ErrDatasetNotLoaded = errors.New("dataset is not loaded")

type DatasetStore interface {
	GetDataset(ctx context.Context) (domain.Dataset, error)
	ReplaceDataset(ctx context.Context, dataset domain.Dataset) error
}

type MemoryStore struct {
	mu      sync.RWMutex
	dataset *domain.Dataset
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) GetDataset(ctx context.Context) (domain.Dataset, error) {
	if err := ctx.Err(); err != nil {
		return domain.Dataset{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dataset == nil {
		return domain.Dataset{}, ErrDatasetNotLoaded
	}
	return *s.dataset, nil
}

func (s *MemoryStore) ReplaceDataset(ctx context.Context, dataset domain.Dataset) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.dataset = &dataset
	return nil
}
