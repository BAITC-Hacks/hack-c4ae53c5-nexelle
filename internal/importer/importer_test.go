package importer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/importer"
)

func TestLoadDirLoadsValidDataset(t *testing.T) {
	dir := writeDataset(t, "")

	dataset, report, err := importer.LoadDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v, report = %#v", err, report)
	}
	if !report.IsValid() {
		t.Fatalf("report should be valid: %#v", report.Errors)
	}

	stats := dataset.Stats()
	if stats.Skills != 1 || stats.RoleProfiles != 1 || stats.Employees != 1 || stats.Events != 1 || stats.ActivityRecords != 0 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestLoadDirReadsOfficialEventFieldNames(t *testing.T) {
	dir := writeDataset(t, "")
	dataset, report, err := importer.LoadDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v, report = %#v", err, report)
	}

	event := dataset.Events["EV_001"]
	if len(event.TargetRoles) != 1 || event.TargetRoles[0] != "Backend Engineer" {
		t.Fatalf("target_roles = %#v", event.TargetRoles)
	}
	if len(event.TargetGrades) != 1 || event.TargetGrades[0] != "Junior" {
		t.Fatalf("target_grades = %#v", event.TargetGrades)
	}
	if len(event.DevelopsSkills) != 1 || event.DevelopsSkills[0].SkillID != "SK_GO" || event.DevelopsSkills[0].MaxLevel != 3 {
		t.Fatalf("develops_skills = %#v", event.DevelopsSkills)
	}
	if event.Prerequisites == nil || len(event.UpcomingSessions) != 0 {
		t.Fatalf("prerequisites = %#v, upcoming_sessions = %#v", event.Prerequisites, event.UpcomingSessions)
	}
}

func TestLoadDirRejectsUnknownEmployeeInActivityHistory(t *testing.T) {
	dir := writeDataset(t, "R000001,E404,EV_001,2026-09-01,,completed,100,95,5,self\n")

	_, report, err := importer.LoadDir(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), importer.ErrDatasetInvalid.Error()) {
		t.Fatalf("LoadDir() error = %v, want validation error", err)
	}
	if len(report.Errors) == 0 {
		t.Fatal("expected validation issues")
	}

	issue := report.Errors[0]
	if issue.File != "activity_history.csv" || issue.Field != "employee_id" {
		t.Fatalf("unexpected issue: %#v", issue)
	}
}

func TestLoadDirContinuesHistoryValidationAfterEarlierFileErrors(t *testing.T) {
	dir := writeDataset(t, "R000001,E404,EV_001,2026-09-01,,completed,100,95,5,self\n")
	writeFile(t, dir, "skills.json", `{
  "proficiency_scale": {"0": "No knowledge"},
  "skills": [],
  "role_profiles": [{
    "role": "Backend Engineer", "grade": "Junior", "required_skills": {"SK_GO": 1}, "critical_skills": ["SK_GO"]
  }]
}`)

	_, report, err := importer.LoadDir(context.Background(), dir)
	if err == nil {
		t.Fatal("LoadDir() should fail validation")
	}

	foundHistoryIssue := false
	for _, issue := range report.Errors {
		if issue.File == "activity_history.csv" && issue.Field == "employee_id" {
			foundHistoryIssue = true
			break
		}
	}
	if !foundHistoryIssue {
		t.Fatalf("history issue was lost after earlier validation errors: %#v", report.Errors)
	}
}

func writeDataset(t *testing.T, historyRows string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "skills.json", `{
  "proficiency_scale": {"0": "No knowledge", "1": "Basic awareness"},
  "skills": [{
    "skill_id": "SK_GO", "name": "Go", "type": "hard", "category": "engineering", "description": "Go language"
  }],
  "role_profiles": [{
    "role": "Backend Engineer", "grade": "Junior", "required_skills": {"SK_GO": 1}, "critical_skills": ["SK_GO"]
  }]
}`)
	writeFile(t, dir, "employees.json", `{
  "employees": [{
    "employee_id": "E0001", "full_name": "Test Employee", "department": "Engineering",
    "role": "Backend Engineer", "grade": "Junior", "manager_id": null,
    "hire_date": "2026-01-01", "tenure_months": 9, "work_format": "hybrid",
    "preferred_language": "ru", "career_goal": null, "skills": {"SK_GO": 1},
    "last_review_date": "2026-08-01"
  }]
}`)
	writeFile(t, dir, "events.json", `{
  "events": [{
    "event_id": "EV_001", "title": "Go Basics", "description": "Course",
    "type": "course", "format": "self_paced", "duration_hours": 2,
    "mandatory": false, "target_roles": ["Backend Engineer"], "target_grades": ["Junior"],
    "develops_skills": [{"skill_id": "SK_GO", "gain": 1, "max_level": 3}],
    "prerequisites": {}, "upcoming_sessions": []
  }]
}`)
	writeFile(t, dir, "activity_history.csv", "record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\n"+historyRows)
	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
