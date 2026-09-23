// Package explanation renders text without changing deterministic scores or evidence.
package explanation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

type localeKey struct{}

func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey{}, locale)
}

func Locale(ctx context.Context) string {
	if locale, _ := ctx.Value(localeKey{}).(string); locale == "kk" {
		return locale
	}
	return "ru"
}

// Provider receives only already calculated facts and returns one text per event.
type Provider interface {
	Generate(ctx context.Context, recommendations []domain.Recommendation, locale string) (map[string]string, error)
}

type Adapter struct {
	provider Provider
	timeout  time.Duration
	active   chan struct{}
}

func New(provider Provider) *Adapter {
	return &Adapter{provider: provider, timeout: 1400 * time.Millisecond, active: make(chan struct{}, 8)}
}

func (a *Adapter) Explain(ctx context.Context, recommendations []domain.Recommendation, locale string) []domain.Recommendation {
	result := append([]domain.Recommendation{}, recommendations...)
	for index := range result {
		result[index].Explanation = Template(result[index], locale)
	}
	if a.provider == nil || len(result) == 0 || ctx.Err() != nil {
		return result
	}
	// Bound workers even when a provider fails to observe cancellation.
	select {
	case a.active <- struct{}{}:
	default:
		return result
	}
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	texts := make(chan map[string]string, 1)
	// Copy nested slices so a provider cannot mutate scoring evidence.
	facts := append([]domain.Recommendation{}, result...)
	for index := range facts {
		facts[index].Evidence.Skills = append([]domain.SkillEvidence{}, result[index].Evidence.Skills...)
		facts[index].Evidence.FallbackReasons = append([]string{}, result[index].Evidence.FallbackReasons...)
	}
	go func() {
		defer func() { <-a.active }()
		response, err := a.provider.Generate(ctx, facts, locale)
		if err != nil {
			texts <- nil
			return
		}
		texts <- response
	}()
	select {
	case response := <-texts:
		for index := range result {
			if text := strings.TrimSpace(response[result[index].EventID]); text != "" && len(text) <= 4000 {
				result[index].Explanation = text
			}
		}
	case <-ctx.Done():
	}
	return result
}

func Template(item domain.Recommendation, locale string) string {
	gain := 0
	for _, skill := range item.Evidence.Skills {
		gain += skill.Gain
	}
	if locale == "kk" {
		if item.Evidence.Fallback {
			return "Мақсатты деңгей талаптары орындалды. Бұл іс-шара дағдыларды қолдауға және әрі қарай дамытуға арналған."
		}
		text := fmt.Sprintf("Іс-шара дағдылардағы алшақтықты %d ұпайға дейін азайтады.", gain)
		if item.Evidence.CriticalGap {
			text += " Маңызды дағдылардағы алшақтыққа басымдық беріледі."
		}
		if item.Evidence.History.NegativeCount > 0 {
			text += " Қатысу тарихындағы бас тартулар мен келмеулер ескерілді."
		}
		return text
	}
	if item.Evidence.Fallback {
		return "Требования целевого грейда выполнены. Мероприятие поддерживает навыки, развивает soft-skills или менторство."
	}
	text := fmt.Sprintf("Мероприятие сокращает разрыв в навыках на величину до %d.", gain)
	if item.Evidence.CriticalGap {
		text += " Критический разрыв имеет абсолютный приоритет."
	}
	if item.Evidence.History.NegativeCount > 0 {
		text += " Учтены пропуски и отказы в истории участия."
	}
	return text
}
