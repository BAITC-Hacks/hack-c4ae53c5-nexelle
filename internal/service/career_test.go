package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/service"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
)

func TestCompleteEventRecalculatesProgressAndPreventsDuplicate(t *testing.T) {
	dataset := domain.NewDataset()
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeMiddle)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeMiddle, RequiredSkills: map[string]int{"SK_GO": 3},
	}
	dataset.Employees["E0001"] = domain.Employee{
		EmployeeID: "E0001", Role: "Backend Engineer", Grade: domain.GradeJunior,
		LastReviewDate: "2026-09-01", Skills: map[string]int{"SK_GO": 1},
	}
	dataset.Events["EV_GO"] = domain.Event{
		EventID: "EV_GO", Title: "Go", Format: "self_paced", DurationHours: 2,
		TargetRoles: []string{"Backend Engineer"}, TargetGrades: []string{domain.GradeMiddle},
		DevelopsSkills: []domain.EventSkillGain{{SkillID: "SK_GO", Gain: 2, MaxLevel: 3}},
		Prerequisites:  map[string]int{},
	}
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}
	careerService, err := service.NewCareerService(memoryStore, mustDate(t, "2026-10-01"))
	if err != nil {
		t.Fatalf("NewCareerService() error = %v", err)
	}

	response, err := careerService.CompleteEvent(context.Background(), "E0001", "EV_GO")
	if err != nil {
		t.Fatalf("CompleteEvent() error = %v", err)
	}
	if response.Profile.Progress.EffectiveSkills["SK_GO"] != 3 {
		t.Fatalf("effective SK_GO = %d, want 3", response.Profile.Progress.EffectiveSkills["SK_GO"])
	}
	if len(response.Profile.Activities) != 1 || response.Profile.Activities[0].Status != domain.ActivityCompleted {
		t.Fatalf("unexpected activities: %#v", response.Profile.Activities)
	}

	_, err = careerService.CompleteEvent(context.Background(), "E0001", "EV_GO")
	if !errors.Is(err, store.ErrEventCompleted) {
		t.Fatalf("second CompleteEvent() error = %v, want ErrEventCompleted", err)
	}
}

func TestCompleteEventRejectsPrerequisites(t *testing.T) {
	dataset := domain.NewDataset()
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeMiddle)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeMiddle, RequiredSkills: map[string]int{},
	}
	dataset.Employees["E0001"] = domain.Employee{
		EmployeeID: "E0001", Role: "Backend Engineer", Grade: domain.GradeJunior,
		LastReviewDate: "2026-09-01", Skills: map[string]int{"SK_GO": 1},
	}
	dataset.Events["EV_ADVANCED_GO"] = domain.Event{
		EventID: "EV_ADVANCED_GO", Title: "Advanced Go", Format: "self_paced", DurationHours: 2,
		TargetRoles: []string{"Backend Engineer"}, TargetGrades: []string{domain.GradeMiddle},
		Prerequisites: map[string]int{"SK_GO": 3},
	}
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}
	careerService, err := service.NewCareerService(memoryStore, mustDate(t, "2026-10-01"))
	if err != nil {
		t.Fatalf("NewCareerService() error = %v", err)
	}

	_, err = careerService.CompleteEvent(context.Background(), "E0001", "EV_ADVANCED_GO")
	if !errors.Is(err, service.ErrPrerequisitesNotMet) {
		t.Fatalf("CompleteEvent() error = %v, want ErrPrerequisitesNotMet", err)
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
