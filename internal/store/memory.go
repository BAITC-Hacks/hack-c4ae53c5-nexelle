package store

import (
	"context"
	"errors"
	"sync"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

var (
	ErrDatasetNotLoaded = errors.New("dataset is not loaded")
	ErrEmployeeNotFound = errors.New("employee not found")
)

type DatasetStore interface {
	GetDataset(ctx context.Context) (domain.Dataset, error)
	ReplaceDataset(ctx context.Context, dataset domain.Dataset) error
	GetEmployee(ctx context.Context, employeeID string) (domain.Employee, error)
	ListEmployeeActivities(ctx context.Context, employeeID string) ([]domain.ActivityRecord, error)
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

func (s *MemoryStore) GetEmployee(ctx context.Context, employeeID string) (domain.Employee, error) {
	if err := ctx.Err(); err != nil {
		return domain.Employee{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dataset == nil {
		return domain.Employee{}, ErrDatasetNotLoaded
	}
	employee, exists := s.dataset.Employees[employeeID]
	if !exists {
		return domain.Employee{}, ErrEmployeeNotFound
	}
	return copyEmployee(employee), nil
}

func (s *MemoryStore) ListEmployeeActivities(ctx context.Context, employeeID string) ([]domain.ActivityRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dataset == nil {
		return nil, ErrDatasetNotLoaded
	}
	if _, exists := s.dataset.Employees[employeeID]; !exists {
		return nil, ErrEmployeeNotFound
	}
	records := s.dataset.ActivitiesByEmployee[employeeID]
	result := make([]domain.ActivityRecord, len(records))
	copy(result, records)
	return result, nil
}

func copyEmployee(employee domain.Employee) domain.Employee {
	result := employee
	result.Skills = make(map[string]int, len(employee.Skills))
	for skillID, level := range employee.Skills {
		result.Skills[skillID] = level
	}
	if employee.ManagerID != nil {
		managerID := *employee.ManagerID
		result.ManagerID = &managerID
	}
	if employee.CareerGoal != nil {
		careerGoal := *employee.CareerGoal
		result.CareerGoal = &careerGoal
	}
	return result
}
