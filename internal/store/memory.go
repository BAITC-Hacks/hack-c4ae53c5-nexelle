package store

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

var (
	ErrDatasetNotLoaded = errors.New("dataset is not loaded")
	ErrEmployeeNotFound = errors.New("employee not found")
	ErrEventNotFound    = errors.New("event not found")
	ErrMandatoryEvent   = errors.New("mandatory event cannot be completed as a recommendation")
	ErrEventCompleted   = errors.New("event is already completed")
)

type DatasetStore interface {
	GetDataset(ctx context.Context) (domain.Dataset, error)
	ReplaceDataset(ctx context.Context, dataset domain.Dataset) error
	GetEmployee(ctx context.Context, employeeID string) (domain.Employee, error)
	ListEmployees(ctx context.Context) ([]domain.Employee, error)
	ListEvents(ctx context.Context) ([]domain.Event, error)
	ListEmployeeActivities(ctx context.Context, employeeID string) ([]domain.ActivityRecord, error)
	AddCompletedActivity(ctx context.Context, employeeID, eventID string, completedAt time.Time) (domain.ActivityRecord, error)
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

// ListEmployees returns copies so API callers cannot mutate the in-memory dataset.
func (s *MemoryStore) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dataset == nil {
		return nil, ErrDatasetNotLoaded
	}
	result := make([]domain.Employee, 0, len(s.dataset.Employees))
	for _, employee := range s.dataset.Employees {
		result = append(result, copyEmployee(employee))
	}
	return result, nil
}

// ListEvents returns copies of every event in the imported catalog.
func (s *MemoryStore) ListEvents(ctx context.Context) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dataset == nil {
		return nil, ErrDatasetNotLoaded
	}
	result := make([]domain.Event, 0, len(s.dataset.Events))
	for _, event := range s.dataset.Events {
		result = append(result, copyEvent(event))
	}
	return result, nil
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

func copyEvent(event domain.Event) domain.Event {
	result := event
	result.TargetRoles = append([]string(nil), event.TargetRoles...)
	result.TargetGrades = append([]string(nil), event.TargetGrades...)
	result.DevelopsSkills = append([]domain.EventSkillGain(nil), event.DevelopsSkills...)
	result.UpcomingSessions = append([]string(nil), event.UpcomingSessions...)
	result.Prerequisites = make(map[string]int, len(event.Prerequisites))
	for skillID, level := range event.Prerequisites {
		result.Prerequisites[skillID] = level
	}
	return result
}

// AddCompletedActivity создаёт серверную запись completed и защищает историю от повторного завершения.
// Повтор допускается только для регулярного клуба EV_036; обязательные мероприятия не проходят этот сценарий.
func (s *MemoryStore) AddCompletedActivity(ctx context.Context, employeeID, eventID string, completedAt time.Time) (domain.ActivityRecord, error) {
	if err := ctx.Err(); err != nil {
		return domain.ActivityRecord{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dataset == nil {
		return domain.ActivityRecord{}, ErrDatasetNotLoaded
	}
	if _, exists := s.dataset.Employees[employeeID]; !exists {
		return domain.ActivityRecord{}, ErrEmployeeNotFound
	}
	event, exists := s.dataset.Events[eventID]
	if !exists {
		return domain.ActivityRecord{}, ErrEventNotFound
	}
	if event.Mandatory {
		return domain.ActivityRecord{}, ErrMandatoryEvent
	}
	for _, record := range s.dataset.ActivitiesByEmployee[employeeID] {
		if record.EventID == eventID && record.Status == domain.ActivityCompleted && eventID != "EV_036" {
			return domain.ActivityRecord{}, ErrEventCompleted
		}
	}

	record := domain.ActivityRecord{
		RecordID:      nextDemoRecordID(s.dataset.ActivityRecords, employeeID, eventID),
		EmployeeID:    employeeID,
		EventID:       eventID,
		Date:          completedAt,
		Status:        domain.ActivityCompleted,
		CompletionPct: 100,
		AssignedBy:    "self",
	}
	s.dataset.ActivityRecords[record.RecordID] = record
	s.dataset.ActivitiesByEmployee[employeeID] = append(s.dataset.ActivitiesByEmployee[employeeID], record)
	return record, nil
}

func nextDemoRecordID(records map[string]domain.ActivityRecord, employeeID, eventID string) string {
	for sequence := 1; ; sequence++ {
		candidate := "DEMO_" + employeeID + "_" + eventID + "_" + strconv.Itoa(sequence)
		if _, exists := records[candidate]; !exists {
			return candidate
		}
	}
}
