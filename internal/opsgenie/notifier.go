package opsgenie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/swiggy-private/gocommons/log"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/constants"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/types"
)

const httpTimeout = 10 * time.Second

// Notifier sends alerts to OpsGenie.
type Notifier interface {
	Send(ctx context.Context, jobName string, jobID, runID int64, errorMessage, category string) error
}

// Config holds OpsGenie credentials and routing info.
type Config struct {
	APIKey      string
	URL         string
	InfraOncall string
	CodeOncall  string
}

type notifier struct {
	cfg    Config
	client *http.Client
}

// New returns a Notifier backed by the given OpsGenie config.
func New(cfg Config) Notifier {
	return &notifier{
		cfg:    cfg,
		client: &http.Client{Timeout: httpTimeout},
	}
}

// Send constructs and dispatches an OpsGenie alert for a failed Databricks run.
func (n *notifier) Send(ctx context.Context, jobName string, jobID, runID int64, errorMessage, category string) error {
	if n.cfg.APIKey == "" {
		log.Infow(ctx, "opsgenie: skipped — API key not set")
		return nil
	}

	responder := n.cfg.CodeOncall
	if category == constants.ErrorCategoryInfra {
		responder = n.cfg.InfraOncall
	}

	payload := types.AlertRequest{
		Message:     fmt.Sprintf("Databricks job failure: %s (run_id=%d)", jobName, runID),
		Alias:       fmt.Sprintf("dbr-job-%d", jobID),
		Priority:    constants.OpsGeniePriority,
		Tags:        []string{category},
		Description: errorMessage,
		Responders: []types.Responder{
			{Type: "user", Username: responder},
		},
		Details: map[string]string{
			"error_category": category,
			"job_id":         fmt.Sprintf("%d", jobID),
			"job_name":       jobName,
			"run_id":         fmt.Sprintf("%d", runID),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("opsgenie: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("opsgenie: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "GenieKey "+n.cfg.APIKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("opsgenie: http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("opsgenie: non-2xx response: status=%d", resp.StatusCode)
	}

	log.Infow(ctx, "opsgenie: alert sent",
		"responder", responder,
		"status", resp.StatusCode,
		"run_id", runID,
	)
	return nil
}