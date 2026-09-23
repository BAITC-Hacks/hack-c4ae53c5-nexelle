package domain

import (
	"fmt"
	"time"
)

const (
	GradeJunior = "Junior"
	GradeMiddle = "Middle"
	GradeSenior = "Senior"
	GradeLead   = "Lead"

	ActivityCompleted  = "completed"
	ActivityInProgress = "in_progress"
	ActivityDropped    = "dropped"
	ActivityNoShow     = "no_show"
	ActivityDeclined   = "declined"
	ActivityOverdue    = "overdue"
)

type Skill struct {
	SkillID     string `json:"skill_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type RoleProfile struct {
	Role           string         `json:"role"`
	Grade          string         `json:"grade"`
	RequiredSkills map[string]int `json:"required_skills"`
	CriticalSkills []string       `json:"critical_skills"`
}

func (p RoleProfile) Key() string {
	return RoleGradeKey(p.Role, p.Grade)
}

func RoleGradeKey(role, grade string) string {
	return role + "\x00" + grade
}

type CareerGoal struct {
	TargetRole  string `json:"target_role"`
	TargetGrade string `json:"target_grade"`
}

type Employee struct {
	EmployeeID        string         `json:"employee_id"`
	FullName          string         `json:"full_name"`
	Department        string         `json:"department"`
	Role              string         `json:"role"`
	Grade             string         `json:"grade"`
	ManagerID         *string        `json:"manager_id"`
	HireDate          string         `json:"hire_date"`
	TenureMonths      int            `json:"tenure_months"`
	WorkFormat        string         `json:"work_format"`
	PreferredLanguage string         `json:"preferred_language"`
	CareerGoal        *CareerGoal    `json:"career_goal"`
	Skills            map[string]int `json:"skills"`
	LastReviewDate    string         `json:"last_review_date"`
}

func (e Employee) ProfileKey() string {
	return RoleGradeKey(e.Role, e.Grade)
}

type EventSkillGain struct {
	SkillID  string `json:"skill_id"`
	Gain     int    `json:"gain"`
	MaxLevel int    `json:"max_level"`
}

type Event struct {
	EventID          string           `json:"event_id"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	Type             string           `json:"type"`
	Format           string           `json:"format"`
	DurationHours    float64          `json:"duration_hours"`
	Mandatory        bool             `json:"mandatory"`
	TargetRoles      []string         `json:"target_roles"`
	TargetGrades     []string         `json:"target_grades"`
	DevelopsSkills   []EventSkillGain `json:"develops_skills"`
	Prerequisites    map[string]int   `json:"prerequisites"`
	UpcomingSessions []string         `json:"upcoming_sessions"`
}

type ActivityRecord struct {
	RecordID       string     `json:"record_id"`
	EmployeeID     string     `json:"employee_id"`
	EventID        string     `json:"event_id"`
	Date           time.Time  `json:"date"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	Status         string     `json:"status"`
	CompletionPct  int        `json:"completion_pct"`
	Score          *int       `json:"score,omitempty"`
	FeedbackRating *int       `json:"feedback_rating,omitempty"`
	AssignedBy     string     `json:"assigned_by"`
}

type Dataset struct {
	ProficiencyScale     map[string]string
	Skills               map[string]Skill
	RoleProfiles         map[string]RoleProfile
	Employees            map[string]Employee
	Events               map[string]Event
	ActivityRecords      map[string]ActivityRecord
	ActivitiesByEmployee map[string][]ActivityRecord
}

func NewDataset() Dataset {
	return Dataset{
		ProficiencyScale:     make(map[string]string),
		Skills:               make(map[string]Skill),
		RoleProfiles:         make(map[string]RoleProfile),
		Employees:            make(map[string]Employee),
		Events:               make(map[string]Event),
		ActivityRecords:      make(map[string]ActivityRecord),
		ActivitiesByEmployee: make(map[string][]ActivityRecord),
	}
}

func (d Dataset) Stats() DatasetStats {
	return DatasetStats{
		Skills:          len(d.Skills),
		RoleProfiles:    len(d.RoleProfiles),
		Employees:       len(d.Employees),
		Events:          len(d.Events),
		ActivityRecords: len(d.ActivityRecords),
	}
}

type DatasetStats struct {
	Skills          int `json:"skills"`
	RoleProfiles    int `json:"role_profiles"`
	Employees       int `json:"employees"`
	Events          int `json:"events"`
	ActivityRecords int `json:"activity_records"`
}

func (d Dataset) ValidateProfile(role, grade string) error {
	if _, ok := d.RoleProfiles[RoleGradeKey(role, grade)]; !ok {
		return fmt.Errorf("role profile %q/%q does not exist", role, grade)
	}
	return nil
}
