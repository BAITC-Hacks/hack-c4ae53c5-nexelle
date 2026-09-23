// Package service объединяет use cases профиля и completion, не привязываясь к HTTP-обработчикам.
package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/progress"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/recommendation"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
)

var ErrPrerequisitesNotMet = errors.New("event prerequisites are not met")

// CareerService собирает профиль, рассчитывает траекторию и сохраняет действия completion.
type CareerService struct {
	store      store.DatasetStore
	cutoffDate time.Time
}

// NewCareerService создаёт сервис с явной датой среза, одинаковой для всех расчётов текущей сессии.
func NewCareerService(datasetStore store.DatasetStore, cutoffDate time.Time) (*CareerService, error) {
	if cutoffDate.IsZero() {
		return nil, fmt.Errorf("cutoff date is required")
	}
	return &CareerService{store: datasetStore, cutoffDate: cutoffDate}, nil
}

// GetProfile возвращает API-контракт профиля с assessed skills, effective skills, gaps и рекомендациями.
func (s *CareerService) GetProfile(ctx context.Context, employeeID string) (domain.ProfileResponse, error) {
	dataset, employee, activities, err := s.profileData(ctx, employeeID)
	if err != nil {
		return domain.ProfileResponse{}, err
	}
	careerProgress, err := progress.Calculate(dataset, employee, activities, s.cutoffDate)
	if err != nil {
		return domain.ProfileResponse{}, fmt.Errorf("calculate progress: %w", err)
	}
	recommendations, err := recommendation.Recommend(recommendation.Input{
		Dataset: dataset, Employee: employee, Activities: activities, Progress: careerProgress, CutoffDate: s.cutoffDate,
	})
	if err != nil {
		return domain.ProfileResponse{}, fmt.Errorf("build recommendations: %w", err)
	}
	return domain.ProfileResponse{
		Employee:        employeeDTO(employee),
		Activities:      activityDTOs(dataset, activities),
		Progress:        careerProgress,
		Recommendations: recommendations,
	}, nil
}

// CompleteEvent проверяет prerequisites, создаёт completed record и возвращает полностью пересчитанный профиль.
func (s *CareerService) CompleteEvent(ctx context.Context, employeeID, eventID string) (domain.CompletionResponse, error) {
	dataset, employee, activities, err := s.profileData(ctx, employeeID)
	if err != nil {
		return domain.CompletionResponse{}, err
	}
	event, exists := dataset.Events[eventID]
	if !exists {
		return domain.CompletionResponse{}, store.ErrEventNotFound
	}
	if event.Mandatory {
		return domain.CompletionResponse{}, store.ErrMandatoryEvent
	}
	careerProgress, err := progress.Calculate(dataset, employee, activities, s.cutoffDate)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("calculate current progress: %w", err)
	}
	for skillID, level := range event.Prerequisites {
		if careerProgress.EffectiveSkills[skillID] < level {
			return domain.CompletionResponse{}, fmt.Errorf("%w: %s requires %d", ErrPrerequisitesNotMet, skillID, level)
		}
	}
	if _, err := s.store.AddCompletedActivity(ctx, employeeID, eventID, s.cutoffDate); err != nil {
		return domain.CompletionResponse{}, err
	}
	profile, err := s.GetProfile(ctx, employeeID)
	if err != nil {
		return domain.CompletionResponse{}, err
	}
	return domain.CompletionResponse{Profile: profile}, nil
}

// GetHRAnalytics строит агрегаты пробелов, отсутствия рекомендаций и участия по тому же состоянию данных,
// которое используется в профиле сотрудника.
func (s *CareerService) GetHRAnalytics(ctx context.Context) (domain.HRAnalytics, error) {
	dataset, err := s.store.GetDataset(ctx)
	if err != nil {
		return domain.HRAnalytics{}, err
	}
	employeeIDs := make([]string, 0, len(dataset.Employees))
	for employeeID := range dataset.Employees {
		employeeIDs = append(employeeIDs, employeeID)
	}
	sort.Strings(employeeIDs)

	gapCounts := make(map[string]int)
	withoutRecommendations := make([]domain.EmployeeWithoutRecommendation, 0)
	for _, employeeID := range employeeIDs {
		employee := dataset.Employees[employeeID]
		activities := dataset.ActivitiesByEmployee[employeeID]
		careerProgress, err := progress.Calculate(dataset, employee, activities, s.cutoffDate)
		if err != nil {
			return domain.HRAnalytics{}, fmt.Errorf("calculate employee %s progress: %w", employeeID, err)
		}
		for _, gap := range careerProgress.SkillGaps {
			if gap.Gap > 0 {
				gapCounts[gap.SkillID]++
			}
		}
		recommendations, err := recommendation.Recommend(recommendation.Input{
			Dataset: dataset, Employee: employee, Activities: activities, Progress: careerProgress, CutoffDate: s.cutoffDate,
		})
		if err != nil {
			return domain.HRAnalytics{}, fmt.Errorf("calculate employee %s recommendations: %w", employeeID, err)
		}
		if len(recommendations) == 0 {
			withoutRecommendations = append(withoutRecommendations, domain.EmployeeWithoutRecommendation{
				EmployeeID: employee.EmployeeID,
				FullName:   employee.FullName,
				Reason:     "no_available_recommendations",
			})
		}
	}

	topSkillGaps := make([]domain.SkillGapSummary, 0, len(gapCounts))
	for skillID, employeeCount := range gapCounts {
		skillName := skillID
		if skill, exists := dataset.Skills[skillID]; exists && skill.Name != "" {
			skillName = skill.Name
		}
		topSkillGaps = append(topSkillGaps, domain.SkillGapSummary{SkillID: skillID, SkillName: skillName, EmployeeCount: employeeCount})
	}
	sort.Slice(topSkillGaps, func(i, j int) bool {
		if topSkillGaps[i].EmployeeCount != topSkillGaps[j].EmployeeCount {
			return topSkillGaps[i].EmployeeCount > topSkillGaps[j].EmployeeCount
		}
		return topSkillGaps[i].SkillID < topSkillGaps[j].SkillID
	})
	if len(topSkillGaps) > 5 {
		topSkillGaps = topSkillGaps[:5]
	}

	participation := participationByEvent(dataset)
	return domain.HRAnalytics{
		EmployeeCount:                  len(dataset.Employees),
		TopSkillGaps:                   topSkillGaps,
		EmployeesWithoutRecommendation: withoutRecommendations,
		ParticipationByEvent:           participation,
	}, nil
}

func (s *CareerService) profileData(ctx context.Context, employeeID string) (domain.Dataset, domain.Employee, []domain.ActivityRecord, error) {
	dataset, err := s.store.GetDataset(ctx)
	if err != nil {
		return domain.Dataset{}, domain.Employee{}, nil, err
	}
	employee, err := s.store.GetEmployee(ctx, employeeID)
	if err != nil {
		return domain.Dataset{}, domain.Employee{}, nil, err
	}
	activities, err := s.store.ListEmployeeActivities(ctx, employeeID)
	if err != nil {
		return domain.Dataset{}, domain.Employee{}, nil, err
	}
	return dataset, employee, activities, nil
}

func employeeDTO(employee domain.Employee) domain.EmployeeProfileDTO {
	result := domain.EmployeeProfileDTO{
		EmployeeID:        employee.EmployeeID,
		FullName:          employee.FullName,
		Department:        employee.Department,
		Role:              employee.Role,
		Grade:             employee.Grade,
		ManagerID:         employee.ManagerID,
		HireDate:          employee.HireDate,
		TenureMonths:      employee.TenureMonths,
		WorkFormat:        employee.WorkFormat,
		PreferredLanguage: employee.PreferredLanguage,
		AssessedSkills:    copySkills(employee.Skills),
		LastReviewDate:    employee.LastReviewDate,
	}
	if employee.CareerGoal != nil {
		result.CareerGoal = &domain.CareerGoalDTO{
			TargetRole:  employee.CareerGoal.TargetRole,
			TargetGrade: employee.CareerGoal.TargetGrade,
		}
	}
	return result
}

func activityDTOs(dataset domain.Dataset, activities []domain.ActivityRecord) []domain.ActivityDTO {
	sorted := append([]domain.ActivityRecord(nil), activities...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Date.Equal(sorted[j].Date) {
			return sorted[i].RecordID < sorted[j].RecordID
		}
		return sorted[i].Date.Before(sorted[j].Date)
	})
	result := make([]domain.ActivityDTO, 0, len(sorted))
	for _, activity := range sorted {
		eventTitle := activity.EventID
		eventFormat := ""
		if event, exists := dataset.Events[activity.EventID]; exists {
			eventTitle = event.Title
			eventFormat = event.Format
		}
		item := domain.ActivityDTO{
			RecordID:       activity.RecordID,
			EmployeeID:     activity.EmployeeID,
			EventID:        activity.EventID,
			EventTitle:     eventTitle,
			EventFormat:    eventFormat,
			Date:           activity.Date.Format(time.DateOnly),
			Status:         activity.Status,
			CompletionPct:  activity.CompletionPct,
			Score:          activity.Score,
			FeedbackRating: activity.FeedbackRating,
			AssignedBy:     activity.AssignedBy,
		}
		if activity.DueDate != nil {
			item.DueDate = activity.DueDate.Format(time.DateOnly)
		}
		result = append(result, item)
	}
	return result
}

func copySkills(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for skillID, level := range source {
		result[skillID] = level
	}
	return result
}

func participationByEvent(dataset domain.Dataset) []domain.EventParticipation {
	byEvent := make(map[string]domain.EventParticipation, len(dataset.Events))
	for eventID := range dataset.Events {
		byEvent[eventID] = domain.EventParticipation{EventID: eventID}
	}
	for _, activity := range dataset.ActivityRecords {
		item := byEvent[activity.EventID]
		item.Registrations++
		switch activity.Status {
		case domain.ActivityCompleted:
			item.Completed++
		case domain.ActivityNoShow:
			item.NoShow++
		case domain.ActivityDropped:
			item.Dropped++
		case domain.ActivityDeclined:
			item.Declined++
		}
		byEvent[activity.EventID] = item
	}
	result := make([]domain.EventParticipation, 0, len(byEvent))
	for _, item := range byEvent {
		if item.Registrations > 0 {
			item.CompletionRate = math.Round(float64(item.Completed)*10000/float64(item.Registrations)) / 100
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Registrations != result[j].Registrations {
			return result[i].Registrations > result[j].Registrations
		}
		return result[i].EventID < result[j].EventID
	})
	return result
}
