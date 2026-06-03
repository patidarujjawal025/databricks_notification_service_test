package types

// WebhookPayload is the inbound Databricks system notification body.
type WebhookPayload struct {
	EventType string  `json:"event_type"`
	Run       RunInfo `json:"run"`
	Job       JobInfo `json:"job"`
}

// RunInfo holds the run-level fields from the webhook.
type RunInfo struct {
	RunID int64 `json:"run_id"`
}

// JobInfo holds the job-level fields from the webhook.
type JobInfo struct {
	JobID int64  `json:"job_id"`
	Name  string `json:"name"`
}

// NotificationResult is the response body returned to the caller after processing.
type NotificationResult struct {
	JobName       string `json:"job_name"`
	JobID         int64  `json:"job_id"`
	RunID         int64  `json:"run_id"`
	ErrorMessage  string `json:"error_message"`
	ErrorCategory string `json:"error_category"`
}

// AlertRequest is the payload sent to OpsGenie.
type AlertRequest struct {
	Message   string            `json:"message"`
	Alias     string            `json:"alias,omitempty"`
	Priority  string            `json:"priority"`
	Tags      []string          `json:"tags"`
	Details   map[string]string `json:"details"`
	Responders []Responder      `json:"responders"`
	Description string          `json:"description"`
}

// Responder identifies an OpsGenie user to page.
type Responder struct {
	Type     string `json:"type"`
	Username string `json:"username"`
}
