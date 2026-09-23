// Package progress вычисляет эффективные навыки и карьерные разрывы без HTTP, БД и внешних сервисов.
package progress

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

// Calculate строит карьерный прогресс сотрудника на указанную дату среза.
// В расчёт попадают завершения после последней оценки и не позднее даты среза.
// Включение самой даты среза необходимо, чтобы completion, созданный в день демо,
// немедленно менял прогресс в ответе API.
func Calculate(dataset domain.Dataset, employee domain.Employee, activities []domain.ActivityRecord, cutoffDate time.Time) (domain.CareerProgress, error) {
	if cutoffDate.IsZero() {
		return domain.CareerProgress{}, fmt.Errorf("cutoff date is required")
	}
	lastReviewDate, err := time.Parse(time.DateOnly, employee.LastReviewDate)
	if err != nil {
		return domain.CareerProgress{}, fmt.Errorf("parse last_review_date: %w", err)
	}

	effectiveSkills := copySkills(employee.Skills)
	sortedActivities := append([]domain.ActivityRecord(nil), activities...)
	sort.SliceStable(sortedActivities, func(i, j int) bool {
		if sortedActivities[i].Date.Equal(sortedActivities[j].Date) {
			return sortedActivities[i].RecordID < sortedActivities[j].RecordID
		}
		return sortedActivities[i].Date.Before(sortedActivities[j].Date)
	})
	for _, activity := range sortedActivities {
		if activity.Status != domain.ActivityCompleted || !activity.Date.After(lastReviewDate) || activity.Date.After(cutoffDate) {
			continue
		}
		event, exists := dataset.Events[activity.EventID]
		if !exists {
			return domain.CareerProgress{}, fmt.Errorf("completed activity references unknown event %q", activity.EventID)
		}
		for _, gain := range event.DevelopsSkills {
			currentLevel := effectiveSkills[gain.SkillID]
			effectiveSkills[gain.SkillID] = min(currentLevel+gain.Gain, gain.MaxLevel)
		}
	}

	target, targetProfile, err := resolveTarget(dataset, employee)
	if err != nil {
		return domain.CareerProgress{}, err
	}
	gaps := makeSkillGaps(targetProfile, effectiveSkills)
	return domain.CareerProgress{
		Target:           target,
		EffectiveSkills:  effectiveSkills,
		SkillGaps:        gaps,
		ReadinessPercent: readinessPercent(targetProfile, effectiveSkills),
	}, nil
}

func resolveTarget(dataset domain.Dataset, employee domain.Employee) (domain.CareerTarget, domain.RoleProfile, error) {
	if employee.CareerGoal != nil {
		profile, exists := dataset.RoleProfiles[domain.RoleGradeKey(employee.CareerGoal.TargetRole, employee.CareerGoal.TargetGrade)]
		if !exists {
			return domain.CareerTarget{}, domain.RoleProfile{}, fmt.Errorf("career goal profile %q/%q does not exist", employee.CareerGoal.TargetRole, employee.CareerGoal.TargetGrade)
		}
		return domain.CareerTarget{Role: profile.Role, Grade: profile.Grade, Source: "careerGoal"}, profile, nil
	}

	nextGrade := nextGrade(employee.Grade)
	profile, exists := dataset.RoleProfiles[domain.RoleGradeKey(employee.Role, nextGrade)]
	if !exists {
		return domain.CareerTarget{}, domain.RoleProfile{}, fmt.Errorf("target profile %q/%q does not exist", employee.Role, nextGrade)
	}
	source := "nextGrade"
	if employee.Grade == domain.GradeLead {
		source = "leadMaintenance"
	}
	return domain.CareerTarget{Role: profile.Role, Grade: profile.Grade, Source: source}, profile, nil
}

func nextGrade(grade string) string {
	switch grade {
	case domain.GradeJunior:
		return domain.GradeMiddle
	case domain.GradeMiddle:
		return domain.GradeSenior
	case domain.GradeSenior:
		return domain.GradeLead
	default:
		return domain.GradeLead
	}
}

func makeSkillGaps(profile domain.RoleProfile, effectiveSkills map[string]int) []domain.SkillGap {
	critical := make(map[string]bool, len(profile.CriticalSkills))
	for _, skillID := range profile.CriticalSkills {
		critical[skillID] = true
	}
	gaps := make([]domain.SkillGap, 0, len(profile.RequiredSkills))
	for skillID, requiredLevel := range profile.RequiredSkills {
		currentLevel := effectiveSkills[skillID]
		gaps = append(gaps, domain.SkillGap{
			SkillID:       skillID,
			CurrentLevel:  currentLevel,
			RequiredLevel: requiredLevel,
			Gap:           max(requiredLevel-currentLevel, 0),
			IsCritical:    critical[skillID],
		})
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].IsCritical != gaps[j].IsCritical {
			return gaps[i].IsCritical
		}
		if gaps[i].Gap != gaps[j].Gap {
			return gaps[i].Gap > gaps[j].Gap
		}
		return gaps[i].SkillID < gaps[j].SkillID
	})
	return gaps
}

func readinessPercent(profile domain.RoleProfile, effectiveSkills map[string]int) float64 {
	requiredTotal := 0
	matchedTotal := 0
	for skillID, requiredLevel := range profile.RequiredSkills {
		requiredTotal += requiredLevel
		matchedTotal += min(effectiveSkills[skillID], requiredLevel)
	}
	if requiredTotal == 0 {
		return 100
	}
	return math.Round(float64(matchedTotal)*10000/float64(requiredTotal)) / 100
}

func copySkills(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for skillID, level := range source {
		result[skillID] = level
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
