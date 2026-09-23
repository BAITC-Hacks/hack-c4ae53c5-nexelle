package recommendation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/progress"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/recommendation"
)

func TestRecommendPrioritizesCriticalSystemDesignOverPublicSpeaking(t *testing.T) {
	cutoff := mustDate(t, "2026-10-01")
	dataset := domain.NewDataset()
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeSenior)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeSenior,
		RequiredSkills: map[string]int{"SK_SYSTEM_DESIGN": 4, "SK_PUBLIC_SPEAKING": 2},
		CriticalSkills: []string{"SK_SYSTEM_DESIGN"},
	}
	dataset.Events["EV_SYSTEM_DESIGN"] = event("EV_SYSTEM_DESIGN", "System Design Lab", "online", []domain.EventSkillGain{{SkillID: "SK_SYSTEM_DESIGN", Gain: 1, MaxLevel: 5}})
	dataset.Events["EV_PUBLIC_SPEAKING"] = event("EV_PUBLIC_SPEAKING", "Public Speaking Club", "offline", []domain.EventSkillGain{{SkillID: "SK_PUBLIC_SPEAKING", Gain: 1, MaxLevel: 5}})
	dataset.Events["EV_TECH_HISTORY"] = event("EV_TECH_HISTORY", "Technical Course", "online", nil)
	dataset.Events["EV_PUBLIC_HISTORY_1"] = event("EV_PUBLIC_HISTORY_1", "Public History 1", "offline", nil)
	dataset.Events["EV_PUBLIC_HISTORY_2"] = event("EV_PUBLIC_HISTORY_2", "Public History 2", "offline", nil)
	dataset.Events["EV_PUBLIC_HISTORY_3"] = event("EV_PUBLIC_HISTORY_3", "Public History 3", "offline", nil)

	employee := domain.Employee{
		EmployeeID: "E0001", Role: "Backend Engineer", Grade: domain.GradeMiddle,
		CareerGoal:     &domain.CareerGoal{TargetRole: "Backend Engineer", TargetGrade: domain.GradeSenior},
		LastReviewDate: "2026-09-01",
		Skills:         map[string]int{"SK_SYSTEM_DESIGN": 2, "SK_PUBLIC_SPEAKING": 1},
	}
	activities := []domain.ActivityRecord{
		{RecordID: "tech-completed", EventID: "EV_TECH_HISTORY", Status: domain.ActivityCompleted, Date: mustDate(t, "2026-08-20")},
		{RecordID: "public-no-show", EventID: "EV_PUBLIC_HISTORY_1", Status: domain.ActivityNoShow, Date: mustDate(t, "2026-09-10")},
		{RecordID: "public-dropped", EventID: "EV_PUBLIC_HISTORY_2", Status: domain.ActivityDropped, Date: mustDate(t, "2026-09-12")},
		{RecordID: "public-declined", EventID: "EV_PUBLIC_HISTORY_3", Status: domain.ActivityDeclined, Date: mustDate(t, "2026-09-15")},
	}
	careerProgress, err := progress.Calculate(dataset, employee, activities, cutoff)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	recommendations, err := recommendation.Recommend(recommendation.Input{
		Dataset: dataset, Employee: employee, Activities: activities, Progress: careerProgress, CutoffDate: cutoff,
	})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}
	if len(recommendations) != 2 {
		t.Fatalf("recommendations count = %d, want 2", len(recommendations))
	}
	if recommendations[0].EventID != "EV_SYSTEM_DESIGN" {
		t.Fatalf("first recommendation = %s, want EV_SYSTEM_DESIGN; scores: %#v", recommendations[0].EventID, recommendations)
	}
	if recommendations[0].Score.GapScore <= recommendations[1].Score.GapScore {
		t.Fatalf("critical gap score must be higher: %#v", recommendations)
	}
	if recommendations[1].Evidence.History.NegativeCount != 3 {
		t.Fatalf("public speaking negative count = %d, want 3", recommendations[1].Evidence.History.NegativeCount)
	}
	if recommendations[0].Explanation == "" {
		t.Fatal("recommendation explanation must not be empty")
	}
	if !strings.Contains(recommendations[0].Explanations["ru"], "для Senior") {
		t.Fatalf("Russian explanation = %q", recommendations[0].Explanations["ru"])
	}
	if !strings.Contains(recommendations[0].Explanations["kk"], "Senior үшін") {
		t.Fatalf("Kazakh explanation = %q", recommendations[0].Explanations["kk"])
	}
}

func event(id, title, format string, gains []domain.EventSkillGain) domain.Event {
	return domain.Event{
		EventID: id, Title: title, Format: format, DurationHours: 2,
		TargetRoles: []string{"Backend Engineer"}, TargetGrades: []string{domain.GradeSenior},
		DevelopsSkills: gains, Prerequisites: map[string]int{}, UpcomingSessions: []string{"2026-10-15"},
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return date
}
