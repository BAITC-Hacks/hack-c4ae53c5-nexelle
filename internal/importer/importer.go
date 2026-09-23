package importer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

var ErrDatasetInvalid = errors.New("dataset validation failed")

type ValidationIssue struct {
	File    string `json:"file"`
	Row     int    `json:"row,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (v ValidationIssue) String() string {
	location := v.File
	if v.Row != 0 {
		location += fmt.Sprintf(":%d", v.Row)
	}
	if v.Field != "" {
		location += "." + v.Field
	}
	return location + ": " + v.Message
}

type ValidationReport struct {
	Errors   []ValidationIssue `json:"errors"`
	Warnings []ValidationIssue `json:"warnings"`
}

func (r ValidationReport) IsValid() bool {
	return len(r.Errors) == 0
}

type skillsDocument struct {
	ProficiencyScale map[string]string    `json:"proficiency_scale"`
	Skills           []domain.Skill       `json:"skills"`
	RoleProfiles     []domain.RoleProfile `json:"role_profiles"`
}

type employeesDocument struct {
	Employees []domain.Employee `json:"employees"`
}

type eventsDocument struct {
	Events []domain.Event `json:"events"`
}

func LoadDir(ctx context.Context, dir string) (domain.Dataset, ValidationReport, error) {
	if err := ctx.Err(); err != nil {
		return domain.Dataset{}, ValidationReport{}, err
	}

	skillsFile, err := os.Open(filepath.Join(dir, "skills.json"))
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read skills.json: %w", err)
	}
	defer skillsFile.Close()
	eventsFile, err := os.Open(filepath.Join(dir, "events.json"))
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read events.json: %w", err)
	}
	defer eventsFile.Close()
	employeesFile, err := os.Open(filepath.Join(dir, "employees.json"))
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read employees.json: %w", err)
	}
	defer employeesFile.Close()
	historyFile, err := os.Open(filepath.Join(dir, "activity_history.csv"))
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read activity_history.csv: %w", err)
	}
	defer historyFile.Close()

	return LoadFiles(ctx, skillsFile, eventsFile, employeesFile, historyFile)
}

// LoadFiles validates a complete dataset supplied by the HTTP import endpoint or
// another source. The readers must contain skills.json, events.json,
// employees.json and activity_history.csv in that order.
func LoadFiles(ctx context.Context, skillsReader, eventsReader, employeesReader, historyReader io.Reader) (domain.Dataset, ValidationReport, error) {
	if err := ctx.Err(); err != nil {
		return domain.Dataset{}, ValidationReport{}, err
	}
	if skillsReader == nil || eventsReader == nil || employeesReader == nil || historyReader == nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("all four dataset files are required")
	}

	skills, err := readJSON[skillsDocument](skillsReader)
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read skills.json: %w", err)
	}
	events, err := readJSON[eventsDocument](eventsReader)
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read events.json: %w", err)
	}
	employees, err := readJSON[employeesDocument](employeesReader)
	if err != nil {
		return domain.Dataset{}, ValidationReport{}, fmt.Errorf("read employees.json: %w", err)
	}

	dataset := domain.NewDataset()
	report := ValidationReport{}
	dataset.ProficiencyScale = skills.ProficiencyScale

	loadSkills(&dataset, skills.Skills, &report)
	loadRoleProfiles(&dataset, skills.RoleProfiles, &report)
	loadEvents(&dataset, events.Events, &report)
	loadEmployees(&dataset, employees.Employees, &report)
	loadActivityHistory(ctx, &dataset, historyReader, &report)
	validateManagers(dataset, &report)

	if !report.IsValid() {
		return dataset, report, ErrDatasetInvalid
	}
	return dataset, report, nil
}

func readJSON[T any](reader io.Reader) (T, error) {
	var result T
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}

func loadSkills(dataset *domain.Dataset, skills []domain.Skill, report *ValidationReport) {
	for _, skill := range skills {
		if skill.SkillID == "" {
			report.add("skills.json", 0, "skill_id", "must not be empty")
			continue
		}
		if _, exists := dataset.Skills[skill.SkillID]; exists {
			report.add("skills.json", 0, "skill_id", "duplicate value "+skill.SkillID)
			continue
		}
		if skill.Type != "hard" && skill.Type != "soft" {
			report.add("skills.json", 0, "type", "must be hard or soft")
		}
		dataset.Skills[skill.SkillID] = skill
	}
}

func loadRoleProfiles(dataset *domain.Dataset, profiles []domain.RoleProfile, report *ValidationReport) {
	for _, profile := range profiles {
		if profile.Role == "" || !isGrade(profile.Grade) {
			report.add("skills.json", 0, "role_profiles", "role and grade must be valid")
			continue
		}
		if _, exists := dataset.RoleProfiles[profile.Key()]; exists {
			report.add("skills.json", 0, "role_profiles", "duplicate role and grade "+profile.Role+"/"+profile.Grade)
			continue
		}
		for skillID, level := range profile.RequiredSkills {
			validateSkillLevel(dataset, report, "skills.json", 0, "required_skills", skillID, level)
		}
		for _, skillID := range profile.CriticalSkills {
			if _, exists := profile.RequiredSkills[skillID]; !exists {
				report.add("skills.json", 0, "critical_skills", skillID+" is not required for this profile")
			}
		}
		dataset.RoleProfiles[profile.Key()] = profile
	}
}

func loadEvents(dataset *domain.Dataset, events []domain.Event, report *ValidationReport) {
	for _, event := range events {
		if event.EventID == "" {
			report.add("events.json", 0, "event_id", "must not be empty")
			continue
		}
		if _, exists := dataset.Events[event.EventID]; exists {
			report.add("events.json", 0, "event_id", "duplicate value "+event.EventID)
			continue
		}
		if event.DurationHours <= 0 {
			report.add("events.json", 0, "duration_hours", "must be greater than zero")
		}
		if event.Format != "online" && event.Format != "offline" && event.Format != "self_paced" {
			report.add("events.json", 0, "format", "must be online, offline or self_paced")
		}
		if event.Format == "self_paced" && len(event.UpcomingSessions) != 0 {
			report.add("events.json", 0, "upcoming_sessions", "must be empty for self_paced events")
		}
		if event.Format != "self_paced" && len(event.UpcomingSessions) == 0 {
			report.add("events.json", 0, "upcoming_sessions", "must contain at least one session for scheduled events")
		}
		for _, session := range event.UpcomingSessions {
			if _, err := parseDate(session); err != nil {
				report.add("events.json", 0, "upcoming_sessions", "invalid date "+session)
			}
		}
		for skillID, level := range event.Prerequisites {
			validateSkillLevel(dataset, report, "events.json", 0, "prerequisites", skillID, level)
		}
		for _, gain := range event.DevelopsSkills {
			if _, exists := dataset.Skills[gain.SkillID]; !exists {
				report.add("events.json", 0, "develops_skills", "unknown skill "+gain.SkillID)
			}
			if gain.Gain <= 0 || gain.MaxLevel < 1 || gain.MaxLevel > 5 {
				report.add("events.json", 0, "develops_skills", "gain must be positive and max_level must be between 1 and 5")
			}
		}
		dataset.Events[event.EventID] = event
	}
}

func loadEmployees(dataset *domain.Dataset, employees []domain.Employee, report *ValidationReport) {
	for _, employee := range employees {
		if employee.EmployeeID == "" {
			report.add("employees.json", 0, "employee_id", "must not be empty")
			continue
		}
		if _, exists := dataset.Employees[employee.EmployeeID]; exists {
			report.add("employees.json", 0, "employee_id", "duplicate value "+employee.EmployeeID)
			continue
		}
		if err := dataset.ValidateProfile(employee.Role, employee.Grade); err != nil {
			report.add("employees.json", 0, "role", err.Error())
		}
		if employee.CareerGoal != nil {
			if err := dataset.ValidateProfile(employee.CareerGoal.TargetRole, employee.CareerGoal.TargetGrade); err != nil {
				report.add("employees.json", 0, "career_goal", err.Error())
			}
		}
		if _, err := parseDate(employee.HireDate); err != nil {
			report.add("employees.json", 0, "hire_date", "must have format YYYY-MM-DD")
		}
		if _, err := parseDate(employee.LastReviewDate); err != nil {
			report.add("employees.json", 0, "last_review_date", "must have format YYYY-MM-DD")
		}
		if employee.TenureMonths < 0 {
			report.add("employees.json", 0, "tenure_months", "must not be negative")
		}
		if employee.WorkFormat != "office" && employee.WorkFormat != "hybrid" && employee.WorkFormat != "remote" {
			report.add("employees.json", 0, "work_format", "must be office, hybrid or remote")
		}
		if employee.PreferredLanguage != "kk" && employee.PreferredLanguage != "ru" && employee.PreferredLanguage != "en" {
			report.add("employees.json", 0, "preferred_language", "must be kk, ru or en")
		}
		for skillID, level := range employee.Skills {
			validateSkillLevel(dataset, report, "employees.json", 0, "skills", skillID, level)
		}
		dataset.Employees[employee.EmployeeID] = employee
	}
}

func validateManagers(dataset domain.Dataset, report *ValidationReport) {
	for _, employee := range dataset.Employees {
		if employee.ManagerID == nil || *employee.ManagerID == "" {
			continue
		}
		manager, exists := dataset.Employees[*employee.ManagerID]
		if !exists {
			report.add("employees.json", 0, "manager_id", "unknown employee "+*employee.ManagerID)
			continue
		}
		if manager.Grade != domain.GradeLead || manager.Department != employee.Department {
			report.add("employees.json", 0, "manager_id", "manager must be a Lead in the same department")
		}
	}
}

func loadActivityHistory(ctx context.Context, dataset *domain.Dataset, input io.Reader, report *ValidationReport) {
	reader := csv.NewReader(input)
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
	if err != nil {
		report.add("activity_history.csv", 0, "", "read header: "+err.Error())
		return
	}
	index := makeHeaderIndex(headers)
	missingRequiredColumn := false
	for _, name := range requiredHistoryColumns {
		if _, exists := index[name]; !exists {
			report.add("activity_history.csv", 1, name, "required column is missing")
			missingRequiredColumn = true
		}
	}
	if missingRequiredColumn {
		return
	}

	completed := make(map[string]struct{})
	for rowNumber := 2; ; rowNumber++ {
		if err := ctx.Err(); err != nil {
			report.add("activity_history.csv", rowNumber, "", "context cancelled")
			return
		}
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			report.add("activity_history.csv", rowNumber, "", "read row: "+err.Error())
			continue
		}
		record, ok := parseActivity(row, index, rowNumber, report)
		if !ok {
			continue
		}
		if _, exists := dataset.ActivityRecords[record.RecordID]; exists {
			report.add("activity_history.csv", rowNumber, "record_id", "duplicate value "+record.RecordID)
			continue
		}
		if _, exists := dataset.Employees[record.EmployeeID]; !exists {
			report.add("activity_history.csv", rowNumber, "employee_id", "unknown employee "+record.EmployeeID)
		}
		event, exists := dataset.Events[record.EventID]
		if !exists {
			report.add("activity_history.csv", rowNumber, "event_id", "unknown event "+record.EventID)
		} else {
			validateActivityForEvent(record, event, rowNumber, report)
			// Compliance is periodically assigned by HR in the supplied history. The
			// recurring club EV_036 is the only voluntary activity that may repeat.
			// New employee completion actions use a stricter rule in the service layer.
			if record.Status == domain.ActivityCompleted && !event.Mandatory && record.EventID != "EV_036" {
				key := record.EmployeeID + "\x00" + record.EventID
				if _, exists := completed[key]; exists {
					report.add("activity_history.csv", rowNumber, "event_id", "event cannot be completed twice by the same employee")
				}
				completed[key] = struct{}{}
			}
		}
		dataset.ActivityRecords[record.RecordID] = record
		dataset.ActivitiesByEmployee[record.EmployeeID] = append(dataset.ActivitiesByEmployee[record.EmployeeID], record)
	}
}

var requiredHistoryColumns = []string{
	"record_id", "employee_id", "event_id", "date", "due_date", "status",
	"completion_pct", "score", "feedback_rating", "assigned_by",
}

func makeHeaderIndex(headers []string) map[string]int {
	result := make(map[string]int, len(headers))
	for index, header := range headers {
		header = strings.TrimPrefix(strings.TrimSpace(header), "\ufeff")
		result[header] = index
	}
	return result
}

func parseActivity(row []string, index map[string]int, rowNumber int, report *ValidationReport) (domain.ActivityRecord, bool) {
	value := func(name string) string {
		position := index[name]
		if position >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[position])
	}

	record := domain.ActivityRecord{
		RecordID:   value("record_id"),
		EmployeeID: value("employee_id"),
		EventID:    value("event_id"),
		Status:     value("status"),
		AssignedBy: value("assigned_by"),
	}
	if record.RecordID == "" || record.EmployeeID == "" || record.EventID == "" {
		report.add("activity_history.csv", rowNumber, "record_id", "record_id, employee_id and event_id are required")
		return domain.ActivityRecord{}, false
	}

	date, err := parseDate(value("date"))
	if err != nil {
		report.add("activity_history.csv", rowNumber, "date", "must have format YYYY-MM-DD")
		return domain.ActivityRecord{}, false
	}
	record.Date = date
	if dueDate := value("due_date"); dueDate != "" {
		parsedDueDate, err := parseDate(dueDate)
		if err != nil {
			report.add("activity_history.csv", rowNumber, "due_date", "must have format YYYY-MM-DD")
			return domain.ActivityRecord{}, false
		}
		record.DueDate = &parsedDueDate
	}
	completion, err := parseInt(value("completion_pct"))
	if err != nil || completion < 0 || completion > 100 {
		report.add("activity_history.csv", rowNumber, "completion_pct", "must be an integer between 0 and 100")
		return domain.ActivityRecord{}, false
	}
	record.CompletionPct = completion

	score, ok := parseOptionalInt(value("score"), 0, 100, "score", rowNumber, report)
	if !ok {
		return domain.ActivityRecord{}, false
	}
	record.Score = score
	feedback, ok := parseOptionalInt(value("feedback_rating"), 1, 5, "feedback_rating", rowNumber, report)
	if !ok {
		return domain.ActivityRecord{}, false
	}
	record.FeedbackRating = feedback

	if !isActivityStatus(record.Status) {
		report.add("activity_history.csv", rowNumber, "status", "unknown status "+record.Status)
	}
	if record.AssignedBy != "self" && record.AssignedBy != "manager" && record.AssignedBy != "hr" {
		report.add("activity_history.csv", rowNumber, "assigned_by", "must be self, manager or hr")
	}
	return record, true
}

func validateActivityForEvent(record domain.ActivityRecord, event domain.Event, rowNumber int, report *ValidationReport) {
	if record.Status == domain.ActivityCompleted && record.CompletionPct != 100 {
		report.add("activity_history.csv", rowNumber, "completion_pct", "completed record must have 100")
	}
	if (record.Status == domain.ActivityNoShow || record.Status == domain.ActivityDeclined) && record.CompletionPct != 0 {
		report.add("activity_history.csv", rowNumber, "completion_pct", "no_show and declined records must have 0")
	}
	if record.Status == domain.ActivityNoShow && event.Format == "self_paced" {
		report.add("activity_history.csv", rowNumber, "status", "no_show is not valid for self_paced events")
	}
	if event.Mandatory && record.DueDate == nil {
		report.add("activity_history.csv", rowNumber, "due_date", "mandatory event must have due_date")
	}
	if !event.Mandatory && record.DueDate != nil {
		report.add("activity_history.csv", rowNumber, "due_date", "due_date is only valid for mandatory events")
	}
}

func parseOptionalInt(value string, minimum, maximum int, field string, rowNumber int, report *ValidationReport) (*int, bool) {
	if value == "" {
		return nil, true
	}
	parsed, err := parseInt(value)
	if err != nil || parsed < minimum || parsed > maximum {
		report.add("activity_history.csv", rowNumber, field, fmt.Sprintf("must be an integer between %d and %d", minimum, maximum))
		return nil, false
	}
	return &parsed, true
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func validateSkillLevel(dataset *domain.Dataset, report *ValidationReport, file string, row int, field, skillID string, level int) {
	if _, exists := dataset.Skills[skillID]; !exists {
		report.add(file, row, field, "unknown skill "+skillID)
	}
	if level < 0 || level > 5 {
		report.add(file, row, field, "level for "+skillID+" must be between 0 and 5")
	}
}

func parseDate(value string) (time.Time, error) {
	return time.Parse(time.DateOnly, value)
}

func isGrade(value string) bool {
	return value == domain.GradeJunior || value == domain.GradeMiddle || value == domain.GradeSenior || value == domain.GradeLead
}

func isActivityStatus(value string) bool {
	switch value {
	case domain.ActivityCompleted, domain.ActivityInProgress, domain.ActivityDropped, domain.ActivityNoShow, domain.ActivityDeclined, domain.ActivityOverdue:
		return true
	default:
		return false
	}
}

func (r *ValidationReport) add(file string, row int, field, message string) {
	r.Errors = append(r.Errors, ValidationIssue{File: file, Row: row, Field: field, Message: message})
}
