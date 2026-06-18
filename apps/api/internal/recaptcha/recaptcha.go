package recaptcha

import (
	"bytes"
	"context"
	"encoding/json"
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

type VerificationError struct {
	Action         string
	ExpectedAction string
	ExpectedHost   string
	Hostname       string
	InvalidReason  string
	Reason         string
	Score          float64
	StatusCode     int
}

func (e *VerificationError) Error() string {
	return "recaptcha verification failed: " + e.Reason
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
		return &VerificationError{Reason: "missing_token", ExpectedAction: expectedAction, ExpectedHost: v.expectedHost}
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
		return &VerificationError{
			Reason:         "assessment_http_status",
			ExpectedAction: expectedAction,
			ExpectedHost:   v.expectedHost,
			StatusCode:     res.StatusCode,
		}
	}

	var assessment assessmentResponse
	if err := json.NewDecoder(res.Body).Decode(&assessment); err != nil {
		return err
	}
	if !assessment.TokenProperties.Valid {
		return &VerificationError{
			Action:         assessment.TokenProperties.Action,
			ExpectedAction: expectedAction,
			ExpectedHost:   v.expectedHost,
			Hostname:       assessment.TokenProperties.Hostname,
			InvalidReason:  assessment.TokenProperties.InvalidReason,
			Reason:         "invalid_token",
			Score:          assessment.RiskAnalysis.Score,
		}
	}
	if assessment.TokenProperties.Action != expectedAction {
		return &VerificationError{
			Action:         assessment.TokenProperties.Action,
			ExpectedAction: expectedAction,
			ExpectedHost:   v.expectedHost,
			Hostname:       assessment.TokenProperties.Hostname,
			Reason:         "action_mismatch",
			Score:          assessment.RiskAnalysis.Score,
		}
	}
	if v.expectedHost != "" && assessment.TokenProperties.Hostname != "" && assessment.TokenProperties.Hostname != v.expectedHost {
		return &VerificationError{
			Action:         assessment.TokenProperties.Action,
			ExpectedAction: expectedAction,
			ExpectedHost:   v.expectedHost,
			Hostname:       assessment.TokenProperties.Hostname,
			Reason:         "hostname_mismatch",
			Score:          assessment.RiskAnalysis.Score,
		}
	}
	if assessment.RiskAnalysis.Score < v.minScore {
		return &VerificationError{
			Action:         assessment.TokenProperties.Action,
			ExpectedAction: expectedAction,
			ExpectedHost:   v.expectedHost,
			Hostname:       assessment.TokenProperties.Hostname,
			Reason:         "score_too_low",
			Score:          assessment.RiskAnalysis.Score,
		}
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
		Action        string `json:"action"`
		Hostname      string `json:"hostname"`
		InvalidReason string `json:"invalidReason"`
		Valid         bool   `json:"valid"`
	} `json:"tokenProperties"`
}
