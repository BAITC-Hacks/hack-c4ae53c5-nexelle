// Package recommendation выбирает объяснимые добровольные мероприятия для карьерного развития.
package recommendation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

// Input содержит все данные, необходимые для чистого и воспроизводимого расчёта рекомендаций.
// Пакет не читает файлов, не использует HTTP и не вызывает внешние модели.
type Input struct {
	Dataset    domain.Dataset
	Employee   domain.Employee
	Activities []domain.ActivityRecord
	Progress   domain.CareerProgress
	CutoffDate time.Time
	Limit      int
}

// Recommend возвращает до Limit доступных рекомендаций, отсортированных по многофакторному score.
func Recommend(input Input) ([]domain.Recommendation, error) {
	if input.CutoffDate.IsZero() {
		return nil, fmt.Errorf("cutoff date is required")
	}
	limit := input.Limit
	if limit <= 0 || limit > 3 {
		limit = 3
	}

	gapsBySkill := make(map[string]domain.SkillGap, len(input.Progress.SkillGaps))
	for _, gap := range input.Progress.SkillGaps {
		gapsBySkill[gap.SkillID] = gap
	}
	completed := completedEvents(input.Activities)
	candidates := make([]candidate, 0, len(input.Dataset.Events))
	for _, event := range input.Dataset.Events {
		if !isEligible(event, input.Employee, input.Progress, gapsBySkill, completed, input.CutoffDate) {
			continue
		}
		candidates = append(candidates, buildCandidate(event, input, gapsBySkill))
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].recommendation.Score.Total != candidates[j].recommendation.Score.Total {
			return candidates[i].recommendation.Score.Total > candidates[j].recommendation.Score.Total
		}
		if candidates[i].recommendation.Score.GapScore != candidates[j].recommendation.Score.GapScore {
			return candidates[i].recommendation.Score.GapScore > candidates[j].recommendation.Score.GapScore
		}
		return candidates[i].recommendation.EventID < candidates[j].recommendation.EventID
	})
	return selectDiverse(candidates, limit), nil
}

type candidate struct {
	recommendation domain.Recommendation
	skillIDs       map[string]struct{}
}

func isEligible(
	event domain.Event,
	employee domain.Employee,
	progress domain.CareerProgress,
	gapsBySkill map[string]domain.SkillGap,
	completed map[string]bool,
	cutoffDate time.Time,
) bool {
	if event.Mandatory || !matchesAudience(event, employee, progress.Target) {
		return false
	}
	if completed[event.EventID] && event.EventID != "EV_036" {
		return false
	}
	if !meetsPrerequisites(event, progress.EffectiveSkills) || !isAvailable(event, cutoffDate) {
		return false
	}
	for _, gain := range event.DevelopsSkills {
		if gap, exists := gapsBySkill[gain.SkillID]; exists && gap.Gap > 0 {
			return true
		}
	}
	return false
}

func buildCandidate(event domain.Event, input Input, gapsBySkill map[string]domain.SkillGap) candidate {
	skills, gapScore, skillIDs := skillEvidence(event, input.Dataset, gapsBySkill)
	history := historyForFormat(event.Format, input.Activities, input.Dataset.Events)
	nextSessionDate := nextSession(event, input.CutoffDate)
	goalMatchScore := 2.0
	if eventMatches(event, input.Progress.Target.Role, input.Progress.Target.Grade) {
		goalMatchScore = 5
	}
	feasibilityScore := feasibility(event, nextSessionDate, input.CutoffDate)
	historyScore := float64(history.CompletedCount*2 - history.NegativeCount*5)
	score := domain.ScoreBreakdown{
		GapScore:         gapScore,
		GoalMatchScore:   goalMatchScore,
		HistoryScore:     historyScore,
		FeasibilityScore: feasibilityScore,
	}
	score.Total = score.GapScore + score.GoalMatchScore + score.HistoryScore + score.FeasibilityScore

	recommendation := domain.Recommendation{
		EventID:         event.EventID,
		Title:           event.Title,
		Format:          event.Format,
		NextSessionDate: nextSessionDate,
		Score:           score,
		Evidence: domain.RecommendationEvidence{
			Skills:  skills,
			History: history,
		},
	}
	recommendation.Explanations = makeExplanations(recommendation, input.Progress.Target)
	recommendation.Explanation = recommendation.Explanations["ru"]
	return candidate{recommendation: recommendation, skillIDs: skillIDs}
}

func skillEvidence(event domain.Event, dataset domain.Dataset, gapsBySkill map[string]domain.SkillGap) ([]domain.SkillEvidence, float64, map[string]struct{}) {
	result := make([]domain.SkillEvidence, 0, len(event.DevelopsSkills))
	skillIDs := make(map[string]struct{})
	gapScore := 0.0
	for _, gain := range event.DevelopsSkills {
		gap, exists := gapsBySkill[gain.SkillID]
		if !exists || gap.Gap == 0 {
			continue
		}
		weight := 1.0
		if gap.IsCritical {
			weight = 2.5
		}
		gapScore += float64(min(gain.Gain, gap.Gap)) * weight
		skillName := gain.SkillID
		if skill, exists := dataset.Skills[gain.SkillID]; exists && skill.Name != "" {
			skillName = skill.Name
		}
		result = append(result, domain.SkillEvidence{
			SkillID:       gain.SkillID,
			SkillName:     skillName,
			CurrentLevel:  gap.CurrentLevel,
			RequiredLevel: gap.RequiredLevel,
			Gain:          gain.Gain,
			IsCritical:    gap.IsCritical,
		})
		skillIDs[gain.SkillID] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsCritical != result[j].IsCritical {
			return result[i].IsCritical
		}
		return result[i].SkillID < result[j].SkillID
	})
	return result, gapScore, skillIDs
}

func completedEvents(activities []domain.ActivityRecord) map[string]bool {
	result := make(map[string]bool)
	for _, activity := range activities {
		if activity.Status == domain.ActivityCompleted {
			result[activity.EventID] = true
		}
	}
	return result
}

func matchesAudience(event domain.Event, employee domain.Employee, target domain.CareerTarget) bool {
	return eventMatches(event, employee.Role, employee.Grade) || eventMatches(event, target.Role, target.Grade)
}

func eventMatches(event domain.Event, role, grade string) bool {
	return contains(event.TargetRoles, role) && contains(event.TargetGrades, grade)
}

func meetsPrerequisites(event domain.Event, skills map[string]int) bool {
	for skillID, requiredLevel := range event.Prerequisites {
		if skills[skillID] < requiredLevel {
			return false
		}
	}
	return true
}

func isAvailable(event domain.Event, cutoffDate time.Time) bool {
	return event.Format == "self_paced" || nextSession(event, cutoffDate) != ""
}

func nextSession(event domain.Event, cutoffDate time.Time) string {
	var earliest time.Time
	for _, rawDate := range event.UpcomingSessions {
		date, err := time.Parse(time.DateOnly, rawDate)
		if err != nil || !date.After(cutoffDate) {
			continue
		}
		if earliest.IsZero() || date.Before(earliest) {
			earliest = date
		}
	}
	if earliest.IsZero() {
		return ""
	}
	return earliest.Format(time.DateOnly)
}

func historyForFormat(format string, activities []domain.ActivityRecord, events map[string]domain.Event) domain.HistorySignal {
	result := domain.HistorySignal{Format: format}
	for _, activity := range activities {
		event, exists := events[activity.EventID]
		if !exists || event.Format != format {
			continue
		}
		switch activity.Status {
		case domain.ActivityCompleted:
			result.CompletedCount++
		case domain.ActivityNoShow, domain.ActivityDropped, domain.ActivityDeclined:
			result.NegativeCount++
		}
	}
	return result
}

func feasibility(event domain.Event, nextSessionDate string, cutoffDate time.Time) float64 {
	if event.Format == "self_paced" {
		return 3
	}
	nextSession, err := time.Parse(time.DateOnly, nextSessionDate)
	if err != nil {
		return 0
	}
	days := int(nextSession.Sub(cutoffDate).Hours() / 24)
	switch {
	case days <= 7:
		return 3
	case days <= 30:
		return 2
	default:
		return 1
	}
}

func selectDiverse(candidates []candidate, limit int) []domain.Recommendation {
	result := make([]domain.Recommendation, 0, limit)
	selected := make(map[string]bool)
	usedFormats := make(map[string]bool)
	usedSkills := make(map[string]bool)
	for _, candidate := range candidates {
		if len(result) == limit {
			return result
		}
		if usedFormats[candidate.recommendation.Format] || !hasNewSkill(candidate.skillIDs, usedSkills) {
			continue
		}
		appendCandidate(&result, candidate, selected, usedFormats, usedSkills)
	}
	for _, candidate := range candidates {
		if len(result) == limit {
			break
		}
		if selected[candidate.recommendation.EventID] {
			continue
		}
		appendCandidate(&result, candidate, selected, usedFormats, usedSkills)
	}
	return result
}

func appendCandidate(result *[]domain.Recommendation, candidate candidate, selected map[string]bool, usedFormats, usedSkills map[string]bool) {
	*result = append(*result, candidate.recommendation)
	selected[candidate.recommendation.EventID] = true
	usedFormats[candidate.recommendation.Format] = true
	for skillID := range candidate.skillIDs {
		usedSkills[skillID] = true
	}
}

func hasNewSkill(skillIDs map[string]struct{}, usedSkills map[string]bool) bool {
	for skillID := range skillIDs {
		if !usedSkills[skillID] {
			return true
		}
	}
	return false
}

func makeExplanations(recommendation domain.Recommendation, target domain.CareerTarget) map[string]string {
	ruFacts := make([]string, 0, 4)
	kkFacts := make([]string, 0, 4)
	if len(recommendation.Evidence.Skills) > 0 {
		skill := recommendation.Evidence.Skills[0]
		skillName := skill.SkillName
		if skillName == "" {
			skillName = skill.SkillID
		}
		ruFacts = append(ruFacts, fmt.Sprintf("%s: %d из %d для %s", skillName, skill.CurrentLevel, skill.RequiredLevel, target.Grade))
		kkFacts = append(kkFacts, fmt.Sprintf("%s: %s үшін %d-ден %d", skillName, target.Grade, skill.RequiredLevel, skill.CurrentLevel))
		if skill.IsCritical {
			ruFacts = append(ruFacts, "навык критичен для цели")
			kkFacts = append(kkFacts, "мақсат үшін маңызды дағды")
		}
	}
	if recommendation.Evidence.History.CompletedCount > 0 {
		ruFacts = append(ruFacts, "этот формат вы раньше завершали")
		kkFacts = append(kkFacts, "бұл форматты бұрын сәтті аяқтағансыз")
	}
	if recommendation.NextSessionDate != "" {
		ruFacts = append(ruFacts, "сессия доступна "+recommendation.NextSessionDate)
		kkFacts = append(kkFacts, "сессия қолжетімді "+recommendation.NextSessionDate)
	} else if recommendation.Format == "self_paced" {
		ruFacts = append(ruFacts, "формат доступен сразу")
		kkFacts = append(kkFacts, "формат бірден қолжетімді")
	}
	for len(ruFacts) < 3 {
		ruFacts = append(ruFacts, "мероприятие соответствует карьерной цели")
		kkFacts = append(kkFacts, "іс-шара мансаптық мақсатқа сәйкес")
	}
	return map[string]string{
		"ru": strings.Join(ruFacts[:3], "; ") + ".",
		"kk": strings.Join(kkFacts[:3], "; ") + ".",
	}
}

func makeExplanation(recommendation domain.Recommendation, target domain.CareerTarget) string {
	facts := make([]string, 0, 4)
	if len(recommendation.Evidence.Skills) > 0 {
		skill := recommendation.Evidence.Skills[0]
		facts = append(facts, fmt.Sprintf("%s: %d из %d для %s", skill.SkillID, skill.CurrentLevel, skill.RequiredLevel, target.Grade))
		if skill.IsCritical {
			facts = append(facts, "навык критичен для цели")
		}
	}
	if recommendation.Evidence.History.CompletedCount > 0 {
		facts = append(facts, "этот формат вы ранее завершали")
	}
	if recommendation.Evidence.History.NegativeCount > 0 {
		facts = append(facts, "учтены предыдущие пропуски этого формата")
	}
	if recommendation.NextSessionDate != "" {
		facts = append(facts, "следующая сессия доступна "+recommendation.NextSessionDate)
	} else if recommendation.Format == "self_paced" {
		facts = append(facts, "self-paced формат доступен сразу")
	}
	for len(facts) < 3 {
		facts = append(facts, "мероприятие соответствует карьерной цели")
	}
	return strings.Join(facts[:3], "; ") + "."
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
