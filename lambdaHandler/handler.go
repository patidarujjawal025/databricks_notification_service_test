package lambdaHandler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/swiggy-private/gocommons/log"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/classifier"
	dbrClient "github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/databricks"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/opsgenie"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/types"
)

// Handler wires together the Databricks client, error classifier, and OpsGenie notifier.
type Handler struct {
	dbrClient  dbrClient.Client
	classifier classifier.Classifier
	notifier   opsgenie.Notifier
}

// New constructs a Handler from its collaborators.
func New(dbr dbrClient.Client, cls classifier.Classifier, notif opsgenie.Notifier) *Handler {
	return &Handler{
		dbrClient:  dbr,
		classifier: cls,
		notifier:   notif,
	}
}

// Handler is the AWS Lambda entry point. It processes Databricks webhook payloads,
// classifies the failure, and dispatches an OpsGenie alert.
func (h *Handler) Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if req.Body == "" {
		log.Errorw(ctx, "empty request body")
		return errorResponse(http.StatusBadRequest, "missing request body"), nil
	}

	var payload types.WebhookPayload
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		log.Errorw(ctx, "failed to unmarshal webhook payload", "err", err)
		return errorResponse(http.StatusBadRequest, "invalid JSON: "+err.Error()), nil
	}

	if payload.EventType != "jobs.on_failure" {
		log.Infow(ctx, "ignoring non-failure event", "event_type", payload.EventType)
		return jsonResponse(http.StatusOK, map[string]string{"status": "ignored", "event_type": payload.EventType}), nil
	}

	runID := payload.Run.RunID
	jobID := payload.Job.JobID
	jobName := payload.Job.Name

	if runID == 0 {
		log.Errorw(ctx, "missing run_id in payload")
		return errorResponse(http.StatusBadRequest, "missing run_id"), nil
	}

	log.Infow(ctx, "processing webhook", "job", jobName, "job_id", jobID, "run_id", runID)

	runDetail, err := h.dbrClient.GetRunDetail(ctx, runID)
	if err != nil {
		log.Errorw(ctx, "failed to fetch run detail", "run_id", runID, "err", err)
		return errorResponse(http.StatusInternalServerError, err.Error()), nil
	}

	category := h.classifier.Classify(runDetail.ErrorMessage)
	log.Infow(ctx, "classified failure", "category", category, "message", runDetail.ErrorMessage)

	if err := h.notifier.Send(ctx, jobName, jobID, runID, runDetail.ErrorMessage, category); err != nil {
		log.Errorw(ctx, "failed to send opsgenie alert", "err", err)
		// Non-fatal: return the classification result even if alerting failed.
	}

	result := types.NotificationResult{
		JobName:       jobName,
		JobID:         jobID,
		RunID:         runID,
		ErrorMessage:  runDetail.ErrorMessage,
		ErrorCategory: category,
	}
	return jsonResponse(http.StatusOK, result), nil
}

func jsonResponse(statusCode int, body any) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func errorResponse(statusCode int, msg string) events.APIGatewayProxyResponse {
	return jsonResponse(statusCode, map[string]string{"error": msg})
}