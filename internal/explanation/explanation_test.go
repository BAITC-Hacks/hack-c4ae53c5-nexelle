package explanation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

type providerFunc func(context.Context, []domain.Recommendation, string) (map[string]string, error)

func (f providerFunc) Generate(ctx context.Context, r []domain.Recommendation, l string) (map[string]string, error) {
	return f(ctx, r, l)
}

func TestTimeoutReturnsTemplateBeforeDeadline(t *testing.T) {
	release := make(chan struct{})
	adapter := New(providerFunc(func(context.Context, []domain.Recommendation, string) (map[string]string, error) {
		<-release
		return map[string]string{"event": "late"}, nil
	}))
	defer close(release)
	start := time.Now()
	got := adapter.Explain(context.Background(), []domain.Recommendation{{EventID: "event"}}, "kk")
	if time.Since(start) >= 1500*time.Millisecond {
		t.Fatal("exceeded 1.5 seconds")
	}
	if got[0].Explanation != Template(got[0], "kk") {
		t.Fatal("timeout must return kk template")
	}
}

func TestProviderCanOnlyChangeText(t *testing.T) {
	item := domain.Recommendation{EventID: "event", Score: domain.ScoreBreakdown{Total: 12}, Evidence: domain.RecommendationEvidence{Skills: []domain.SkillEvidence{{SkillID: "skill", Gain: 1}}}}
	adapter := New(providerFunc(func(_ context.Context, r []domain.Recommendation, _ string) (map[string]string, error) {
		r[0].Evidence.Skills[0].Gain = 100
		return map[string]string{"event": "Текст"}, nil
	}))
	got := adapter.Explain(context.Background(), []domain.Recommendation{item}, "ru")
	if got[0].Explanation != "Текст" || !reflect.DeepEqual(got[0].Evidence, item.Evidence) || item.Evidence.Skills[0].Gain != 1 {
		t.Fatal("provider mutated evidence")
	}
	failed := New(providerFunc(func(context.Context, []domain.Recommendation, string) (map[string]string, error) {
		return nil, errors.New("offline")
	}))
	if failed.Explain(context.Background(), []domain.Recommendation{item}, "ru")[0].Explanation != Template(item, "ru") {
		t.Fatal("failure fallback")
	}
}

func TestHTTPProviderContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer test" {
			t.Error("invalid request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"event\":\"Пояснение\"}"))
	}))
	defer server.Close()
	got, err := (HTTPProvider{URL: server.URL, Token: "test"}).Generate(context.Background(), []domain.Recommendation{{EventID: "event"}}, "ru")
	if err != nil || got["event"] != "Пояснение" {
		t.Fatalf("%v %v", got, err)
	}
}
