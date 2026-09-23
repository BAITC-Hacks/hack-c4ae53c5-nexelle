package explanation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
)

// HTTPProvider uses the documented text-only gateway contract in docs/integration.md.
type HTTPProvider struct {
	URL   string
	Token string
}

func (p HTTPProvider) Generate(ctx context.Context, recommendations []domain.Recommendation, locale string) (map[string]string, error) {
	body, err := json.Marshal(struct {
		Locale          string                  `json:"locale"`
		Recommendations []domain.Recommendation `json:"recommendations"`
	}{locale, recommendations})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if p.Token != "" {
		request.Header.Set("Authorization", "Bearer "+p.Token)
	}
	client := &http.Client{Timeout: 1400 * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("explanation provider status %d", response.StatusCode)
	}
	var texts map[string]string
	if err := json.NewDecoder(io.LimitReader(response.Body, 32<<10)).Decode(&texts); err != nil {
		return nil, err
	}
	return texts, nil
}
