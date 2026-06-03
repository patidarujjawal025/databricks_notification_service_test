package lambdaHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbrClient "github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/databricks"
)

// --- mock: Databricks client ---

type mockDbrClient struct{ mock.Mock }

func (m *mockDbrClient) GetRunDetail(ctx context.Context, runID int64) (*dbrClient.RunDetail, error) {
	args := m.Called(ctx, runID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dbrClient.RunDetail), args.Error(1)
}

// --- mock: Classifier ---

type mockClassifier struct{ mock.Mock }

func (m *mockClassifier) Classify(msg string) string {
	return m.Called(msg).String(0)
}

// --- mock: Notifier ---

type mockNotifier struct{ mock.Mock }

func (m *mockNotifier) Send(ctx context.Context, jobName string, jobID, runID int64, errMsg, category string) error {
	return m.Called(ctx, jobName, jobID, runID, errMsg, category).Error(0)
}

// --- tests ---

func TestHandler_HappyPath(t *testing.T) {
	payload := `{"event_type":"jobs.on_failure","run":{"run_id":42},"job":{"job_id":7,"name":"my-job"}}`

	dbr := new(mockDbrClient)
	dbr.On("GetRunDetail", mock.Anything, int64(42)).
		Return(&dbrClient.RunDetail{ErrorMessage: "executor lost"}, nil)

	cls := new(mockClassifier)
	cls.On("Classify", "executor lost").Return("INFRA_ERROR")

	notif := new(mockNotifier)
	notif.On("Send", mock.Anything, "my-job", int64(7), int64(42), "executor lost", "INFRA_ERROR").Return(nil)

	h := New(dbr, cls, notif)
	resp, err := h.Handler(context.Background(), events.APIGatewayProxyRequest{Body: payload})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	assert.NoError(t, json.Unmarshal([]byte(resp.Body), &result))
	assert.Equal(t, "my-job", result["job_name"])
	assert.Equal(t, "INFRA_ERROR", result["error_category"])

	dbr.AssertExpectations(t)
	cls.AssertExpectations(t)
	notif.AssertExpectations(t)
}

func TestHandler_NonFailureEventIgnored(t *testing.T) {
	h := New(nil, nil, nil)
	resp, err := h.Handler(context.Background(), events.APIGatewayProxyRequest{
		Body: `{"event_type":"jobs.on_start","run":{"run_id":1},"job":{"job_id":1,"name":"x"}}`,
	})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	assert.NoError(t, json.Unmarshal([]byte(resp.Body), &result))
	assert.Equal(t, "ignored", result["status"])
}

func TestHandler_EmptyBody(t *testing.T) {
	h := New(nil, nil, nil)
	resp, err := h.Handler(context.Background(), events.APIGatewayProxyRequest{Body: ""})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandler_MissingRunID(t *testing.T) {
	h := New(nil, nil, nil)
	resp, err := h.Handler(context.Background(), events.APIGatewayProxyRequest{
		Body: `{"event_type":"jobs.on_failure","run":{"run_id":0},"job":{"job_id":1,"name":"x"}}`,
	})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandler_InvalidJSON(t *testing.T) {
	h := New(nil, nil, nil)
	resp, err := h.Handler(context.Background(), events.APIGatewayProxyRequest{Body: "{bad json"})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}