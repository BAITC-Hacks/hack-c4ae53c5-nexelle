package progress_test

import (
	"testing"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/progress"
)

func TestCalculateAppliesOnlyCompletedActivitiesAfterReview(t *testing.T) {
	dataset := domain.NewDataset()
	dataset.Events["EV_GO"] = domain.Event{
		EventID:        "EV_GO",
		DevelopsSkills: []domain.EventSkillGain{{SkillID: "SK_GO", Gain: 2, MaxLevel: 3}},
	}
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeMiddle)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeMiddle,
		RequiredSkills: map[string]int{"SK_GO": 3}, CriticalSkills: []string{"SK_GO"},
	}
	employee := domain.Employee{
		EmployeeID: "E0001", Role: "Backend Engineer", Grade: domain.GradeJunior,
		LastReviewDate: "2026-09-01", Skills: map[string]int{"SK_GO": 1},
	}
	activities := []domain.ActivityRecord{
		{RecordID: "before-review", EventID: "EV_GO", Status: domain.ActivityCompleted, Date: mustDate(t, "2026-08-31")},
		{RecordID: "not-completed", EventID: "EV_GO", Status: domain.ActivityDropped, Date: mustDate(t, "2026-09-10")},
		{RecordID: "after-cutoff", EventID: "EV_GO", Status: domain.ActivityCompleted, Date: mustDate(t, "2026-10-02")},
		{RecordID: "on-cutoff", EventID: "EV_GO", Status: domain.ActivityCompleted, Date: mustDate(t, "2026-10-01")},
	}

	result, err := progress.Calculate(dataset, employee, activities, mustDate(t, "2026-10-01"))
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result.EffectiveSkills["SK_GO"] != 3 {
		t.Fatalf("effective SK_GO = %d, want 3", result.EffectiveSkills["SK_GO"])
	}
	if result.Target.Grade != domain.GradeMiddle || result.Target.Source != "nextGrade" {
		t.Fatalf("unexpected target: %#v", result.Target)
	}
	if result.ReadinessPercent != 100 {
		t.Fatalf("readiness = %v, want 100", result.ReadinessPercent)
	}
}

func TestCalculateKeepsLeadWithoutGoalOnLeadProfile(t *testing.T) {
	dataset := domain.NewDataset()
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeLead)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeLead, RequiredSkills: map[string]int{},
	}
	employee := domain.Employee{
		EmployeeID: "E0001", Role: "Backend Engineer", Grade: domain.GradeLead,
		LastReviewDate: "2026-09-01", Skills: map[string]int{},
	}

	result, err := progress.Calculate(dataset, employee, nil, mustDate(t, "2026-10-01"))
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result.Target.Grade != domain.GradeLead || result.Target.Source != "leadMaintenance" {
		t.Fatalf("unexpected target: %#v", result.Target)
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return date
}
