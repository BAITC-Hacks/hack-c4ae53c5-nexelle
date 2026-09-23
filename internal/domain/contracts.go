package domain

// CareerTarget описывает профиль роли и грейда, относительно которого строится траектория сотрудника.
// Source принимает значения careerGoal, nextGrade или leadMaintenance.
type CareerTarget struct {
	Role   string `json:"role"`
	Grade  string `json:"grade"`
	Source string `json:"source"`
}

// SkillGap содержит объяснимый разрыв между эффективным уровнем навыка и требованием цели.
type SkillGap struct {
	SkillID       string `json:"skillId"`
	CurrentLevel  int    `json:"currentLevel"`
	RequiredLevel int    `json:"requiredLevel"`
	Gap           int    `json:"gap"`
	IsCritical    bool   `json:"isCritical"`
}

// CareerProgress объединяет результат расчёта навыков, цели и разрывов для одного сотрудника.
type CareerProgress struct {
	Target           CareerTarget   `json:"target"`
	EffectiveSkills  map[string]int `json:"effectiveSkills"`
	SkillGaps        []SkillGap     `json:"skillGaps"`
	ReadinessPercent float64        `json:"readinessPercent"`
}

// ScoreBreakdown показывает вклад каждого фактора в итоговый балл рекомендации.
type ScoreBreakdown struct {
	GapScore         float64 `json:"gapScore"`
	GoalMatchScore   float64 `json:"goalMatchScore"`
	HistoryScore     float64 `json:"historyScore"`
	FeasibilityScore float64 `json:"feasibilityScore"`
	Total            float64 `json:"total"`
}

// SkillEvidence объясняет, как мероприятие закрывает конкретный карьерный разрыв.
type SkillEvidence struct {
	SkillID       string `json:"skillId"`
	CurrentLevel  int    `json:"currentLevel"`
	RequiredLevel int    `json:"requiredLevel"`
	Gain          int    `json:"gain"`
	IsCritical    bool   `json:"isCritical"`
}

// HistorySignal отражает опыт сотрудника с форматом рекомендованного мероприятия.
type HistorySignal struct {
	Format         string `json:"format"`
	CompletedCount int    `json:"completedCount"`
	NegativeCount  int    `json:"negativeCount"`
}

// RecommendationEvidence содержит факты, на которых основана рекомендация и её объяснение.
type RecommendationEvidence struct {
	Skills  []SkillEvidence `json:"skills"`
	History HistorySignal   `json:"history"`
}

// Recommendation является контрактом между движком рекомендаций, API и пользовательским интерфейсом.
// Explanation формируется детерминированным шаблоном; LLM может только стилизовать этот текст.
type Recommendation struct {
	EventID         string                 `json:"eventId"`
	Title           string                 `json:"title"`
	Format          string                 `json:"format"`
	NextSessionDate string                 `json:"nextSessionDate,omitempty"`
	Score           ScoreBreakdown         `json:"score"`
	Evidence        RecommendationEvidence `json:"evidence"`
	Explanation     string                 `json:"explanation"`
}

// EmployeeProfileDTO - camelCase-контракт профиля для фронтенда.
// Исходная структура Employee сохраняет snake_case-теги, так как напрямую декодирует датасет.
type EmployeeProfileDTO struct {
	EmployeeID        string         `json:"employeeId"`
	FullName          string         `json:"fullName"`
	Department        string         `json:"department"`
	Role              string         `json:"role"`
	Grade             string         `json:"grade"`
	ManagerID         *string        `json:"managerId,omitempty"`
	HireDate          string         `json:"hireDate"`
	TenureMonths      int            `json:"tenureMonths"`
	WorkFormat        string         `json:"workFormat"`
	PreferredLanguage string         `json:"preferredLanguage"`
	CareerGoal        *CareerGoalDTO `json:"careerGoal,omitempty"`
	AssessedSkills    map[string]int `json:"assessedSkills"`
	LastReviewDate    string         `json:"lastReviewDate"`
}

// CareerGoalDTO описывает явно заданную карьерную цель в контракте API.
type CareerGoalDTO struct {
	TargetRole  string `json:"targetRole"`
	TargetGrade string `json:"targetGrade"`
}

// ActivityDTO представляет запись истории в camelCase-контракте API.
type ActivityDTO struct {
	RecordID       string `json:"recordId"`
	EmployeeID     string `json:"employeeId"`
	EventID        string `json:"eventId"`
	Date           string `json:"date"`
	DueDate        string `json:"dueDate,omitempty"`
	Status         string `json:"status"`
	CompletionPct  int    `json:"completionPct"`
	Score          *int   `json:"score,omitempty"`
	FeedbackRating *int   `json:"feedbackRating,omitempty"`
	AssignedBy     string `json:"assignedBy"`
}

// ProfileResponse - полный ответ профиля сотрудника для экрана employee и просмотра HR.
type ProfileResponse struct {
	Employee        EmployeeProfileDTO `json:"employee"`
	Activities      []ActivityDTO      `json:"activities"`
	Progress        CareerProgress     `json:"progress"`
	Recommendations []Recommendation   `json:"recommendations"`
}

// CompletionResponse возвращается после отметки мероприятия завершённым.
type CompletionResponse struct {
	Profile ProfileResponse `json:"profile"`
}

// SkillGapSummary показывает, сколько сотрудников имеют ненулевой разрыв по навыку до своей цели.
type SkillGapSummary struct {
	SkillID       string `json:"skillId"`
	EmployeeCount int    `json:"employeeCount"`
}

// EmployeeWithoutRecommendation объясняет, почему сотруднику нельзя показать доступный следующий шаг.
type EmployeeWithoutRecommendation struct {
	EmployeeID string `json:"employeeId"`
	FullName   string `json:"fullName"`
	Reason     string `json:"reason"`
}

// EventParticipation содержит агрегированную статистику участия без публичного ранжирования сотрудников.
type EventParticipation struct {
	EventID        string  `json:"eventId"`
	Registrations  int     `json:"registrations"`
	Completed      int     `json:"completed"`
	NoShow         int     `json:"noShow"`
	Dropped        int     `json:"dropped"`
	Declined       int     `json:"declined"`
	CompletionRate float64 `json:"completionRate"`
}

// HRAnalytics - агрегированный контракт HR-экрана.
type HRAnalytics struct {
	TopSkillGaps                   []SkillGapSummary               `json:"topSkillGaps"`
	EmployeesWithoutRecommendation []EmployeeWithoutRecommendation `json:"employeesWithoutRecommendation"`
	ParticipationByEvent           []EventParticipation            `json:"participationByEvent"`
}
