package recaptcha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Verifier struct {
	apiKey       string
	expectedHost string
	httpClient   *http.Client
	minScore     float64
	projectID    string
	siteKey      string
}

type Config struct {
	APIKey     string
	AppBaseURL string
	MinScore   float64
	ProjectID  string
	SiteKey    string
}

func New(cfg Config) *Verifier {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.ProjectID) == "" || strings.TrimSpace(cfg.SiteKey) == "" {
		return nil
	}
	expectedHost := ""
	if parsed, err := url.Parse(cfg.AppBaseURL); err == nil {
		expectedHost = parsed.Hostname()
	}
	return &Verifier{
		apiKey:       strings.TrimSpace(cfg.APIKey),
		expectedHost: expectedHost,
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		minScore:     cfg.MinScore,
		projectID:    strings.TrimSpace(cfg.ProjectID),
		siteKey:      strings.TrimSpace(cfg.SiteKey),
	}
}

func (v *Verifier) Enabled() bool {
	return v != nil
}

func (v *Verifier) Verify(ctx context.Context, token, expectedAction string) error {
	if v == nil {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("missing token")
	}

	body, err := json.Marshal(assessmentRequest{
		Event: assessmentEvent{
			Token:          token,
			SiteKey:        v.siteKey,
			ExpectedAction: expectedAction,
		},
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("https://recaptchaenterprise.googleapis.com/v1/projects/%s/assessments?key=%s", url.PathEscape(v.projectID), url.QueryEscape(v.apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("recaptcha assessment failed with status %d", res.StatusCode)
	}

	var assessment assessmentResponse
	if err := json.NewDecoder(res.Body).Decode(&assessment); err != nil {
		return err
	}
	if !assessment.TokenProperties.Valid {
		return errors.New("invalid token")
	}
	if assessment.TokenProperties.Action != expectedAction {
		return errors.New("action mismatch")
	}
	if v.expectedHost != "" && assessment.TokenProperties.Hostname != "" && assessment.TokenProperties.Hostname != v.expectedHost {
		return errors.New("hostname mismatch")
	}
	if assessment.RiskAnalysis.Score < v.minScore {
		return errors.New("score too low")
	}
	return nil
}

type assessmentRequest struct {
	Event assessmentEvent `json:"event"`
}

type assessmentEvent struct {
	Token          string `json:"token"`
	SiteKey        string `json:"siteKey"`
	ExpectedAction string `json:"expectedAction"`
}

type assessmentResponse struct {
	RiskAnalysis struct {
		Score float64 `json:"score"`
	} `json:"riskAnalysis"`
	TokenProperties struct {
		Action   string `json:"action"`
		Hostname string `json:"hostname"`
		Valid    bool   `json:"valid"`
	} `json:"tokenProperties"`
}
