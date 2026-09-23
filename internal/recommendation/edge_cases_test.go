package recommendation_test

import (
	"reflect"
	"testing"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/recommendation"
)

func input(t *testing.T) recommendation.Input {
	t.Helper()
	dataset := domain.NewDataset()
	dataset.Skills["SK_SYSTEM_DESIGN"] = domain.Skill{SkillID: "SK_SYSTEM_DESIGN", Type: "hard"}
	dataset.Skills["SK_PUBLIC_SPEAKING"] = domain.Skill{SkillID: "SK_PUBLIC_SPEAKING", Type: "soft"}
	dataset.Events["system"] = event("system", "System Design", "online", []domain.EventSkillGain{{SkillID: "SK_SYSTEM_DESIGN", Gain: 1, MaxLevel: 5}})
	dataset.Events["speaking"] = event("speaking", "Public Speaking", "offline", []domain.EventSkillGain{{SkillID: "SK_PUBLIC_SPEAKING", Gain: 1, MaxLevel: 5}})
	return recommendation.Input{
		Dataset: dataset, Employee: domain.Employee{EmployeeID: "E1", Role: "Backend Engineer", Grade: domain.GradeSenior},
		CutoffDate: mustDate(t, "2026-10-01"),
		Progress: domain.CareerProgress{
			Target:          domain.CareerTarget{Role: "Backend Engineer", Grade: domain.GradeSenior},
			EffectiveSkills: map[string]int{"SK_SYSTEM_DESIGN": 1, "SK_PUBLIC_SPEAKING": 1},
			SkillGaps: []domain.SkillGap{
				{SkillID: "SK_SYSTEM_DESIGN", CurrentLevel: 1, RequiredLevel: 4, Gap: 3, IsCritical: true},
				{SkillID: "SK_PUBLIC_SPEAKING", CurrentLevel: 1, RequiredLevel: 2, Gap: 1},
			},
		},
	}
}

func TestCriticalGapWinsDespiteItsOwnNegativeHistory(t *testing.T) {
	in := input(t)
	for i := 0; i < 100; i++ {
		for _, status := range []string{domain.ActivityNoShow, domain.ActivityDropped, domain.ActivityDeclined} {
			in.Activities = append(in.Activities, domain.ActivityRecord{EventID: "system", Status: status, Date: mustDate(t, "2026-09-20")})
		}
	}
	result, err := recommendation.Recommend(in)
	if err != nil || len(result) != 2 {
		t.Fatalf("result: %+v, %v", result, err)
	}
	if result[0].EventID != "system" || result[0].Score.Total >= result[1].Score.Total {
		t.Fatalf("critical tier must override numeric score: %+v", result)
	}
	in.Limit = 1
	one, _ := recommendation.Recommend(in)
	if one[0].EventID != "system" {
		t.Fatal("limit must be applied after critical tier")
	}
}

func TestEligibilityBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Event, *recommendation.Input)
		want   int
	}{
		{"mandatory", func(e *domain.Event, _ *recommendation.Input) { e.Mandatory = true }, 0},
		{"wrong role", func(e *domain.Event, _ *recommendation.Input) { e.TargetRoles = []string{"Designer"} }, 0},
		{"implicit next grade is not explicit goal", func(e *domain.Event, in *recommendation.Input) { in.Employee.Grade = domain.GradeMiddle }, 0},
		{"explicit career goal", func(e *domain.Event, in *recommendation.Input) {
			in.Employee.Grade = domain.GradeMiddle
			in.Employee.CareerGoal = &domain.CareerGoal{TargetRole: "Backend Engineer", TargetGrade: domain.GradeSenior}
		}, 1},
		{"session on cutoff", func(e *domain.Event, _ *recommendation.Input) { e.UpcomingSessions = []string{"2026-10-01"} }, 0},
		{"session after cutoff", func(e *domain.Event, _ *recommendation.Input) { e.UpcomingSessions = []string{"2026-10-02"} }, 1},
		{"self paced", func(e *domain.Event, _ *recommendation.Input) { e.Format = "self_paced"; e.UpcomingSessions = nil }, 1},
		{"skill prerequisite", func(e *domain.Event, _ *recommendation.Input) {
			e.Prerequisites = map[string]int{"SK_SYSTEM_DESIGN": 2}
		}, 0},
		{"event prerequisite absent", func(e *domain.Event, _ *recommendation.Input) { e.PrerequisiteEvents = []string{"intro"} }, 0},
		{"event prerequisite completed", func(e *domain.Event, in *recommendation.Input) {
			e.PrerequisiteEvents = []string{"intro"}
			in.Activities = []domain.ActivityRecord{{EventID: "intro", Status: domain.ActivityCompleted, Date: in.CutoffDate}}
		}, 1},
		{"event prerequisite dropped", func(e *domain.Event, in *recommendation.Input) {
			e.PrerequisiteEvents = []string{"intro"}
			in.Activities = []domain.ActivityRecord{{EventID: "intro", Status: domain.ActivityDropped, Date: in.CutoffDate}}
		}, 0},
		{"event prerequisite future", func(e *domain.Event, in *recommendation.Input) {
			e.PrerequisiteEvents = []string{"intro"}
			in.Activities = []domain.ActivityRecord{{EventID: "intro", Status: domain.ActivityCompleted, Date: in.CutoffDate.AddDate(0, 0, 1)}}
		}, 0},
		{"already completed", func(e *domain.Event, in *recommendation.Input) {
			in.Activities = []domain.ActivityRecord{{EventID: e.EventID, Status: domain.ActivityCompleted, Date: in.CutoffDate}}
		}, 0},
		{"repeatable club", func(e *domain.Event, in *recommendation.Input) {
			e.EventID = "EV_036"
			in.Activities = []domain.ActivityRecord{{EventID: "EV_036", Status: domain.ActivityCompleted, Date: in.CutoffDate}}
		}, 1},
		{"at cap", func(e *domain.Event, _ *recommendation.Input) { e.DevelopsSkills[0].MaxLevel = 1 }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := input(t)
			e := in.Dataset.Events["system"]
			tc.change(&e, &in)
			in.Dataset.Events = map[string]domain.Event{e.EventID: e}
			result, err := recommendation.Recommend(in)
			if err != nil || len(result) != tc.want {
				t.Fatalf("got %d, want %d; %v", len(result), tc.want, err)
			}
		})
	}
}

func TestLeadFallbackAndDeterminism(t *testing.T) {
	in := input(t)
	in.Employee.Grade = domain.GradeLead
	in.Progress.Target.Grade = domain.GradeLead
	for i := range in.Progress.SkillGaps {
		in.Progress.SkillGaps[i].Gap = 0
		in.Progress.SkillGaps[i].CurrentLevel = 5
	}
	in.Dataset.Events["mentor"] = domain.Event{EventID: "mentor", Type: "mentoring", Format: "self_paced", TargetRoles: []string{"Backend Engineer"}, TargetGrades: []string{"Lead"}}
	for id, e := range in.Dataset.Events {
		e.TargetGrades = []string{"Lead"}
		in.Dataset.Events[id] = e
	}
	first, err := recommendation.Recommend(in)
	if err != nil || len(first) != 3 {
		t.Fatalf("fallback: %+v, %v", first, err)
	}
	for _, r := range first {
		if !r.Evidence.Fallback {
			t.Fatal("missing fallback evidence")
		}
	}
	for range 20 {
		result, _ := recommendation.Recommend(in)
		if !reflect.DeepEqual(first, result) {
			t.Fatal("nondeterministic result")
		}
	}
}
