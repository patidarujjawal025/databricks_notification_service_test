package databricks

import (
	"context"
	"fmt"
	"strings"

	dbrsdk "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// RunDetail holds the error information extracted from a Databricks job run.
type RunDetail struct {
	ErrorMessage string
}

// Client fetches Databricks job run details.
type Client interface {
	GetRunDetail(ctx context.Context, runID int64) (*RunDetail, error)
}

type dbrClient struct {
	workspace *dbrsdk.WorkspaceClient
}

// Config holds the Databricks workspace credentials.
type Config struct {
	Host  string
	Token string
}

// New creates a Databricks client from the given config.
func New(cfg Config) (Client, error) {
	wc, err := dbrsdk.NewWorkspaceClient(&dbrsdk.Config{
		Host:  cfg.Host,
		Token: cfg.Token,
	})
	if err != nil {
		return nil, fmt.Errorf("databricks: failed to create workspace client: %w", err)
	}
	return &dbrClient{workspace: wc}, nil
}

// GetRunDetail fetches the run and extracts the consolidated error message from
// the run-level state and all task states (mirrors the Python extract_error_message).
func (c *dbrClient) GetRunDetail(ctx context.Context, runID int64) (*RunDetail, error) {
	run, err := c.workspace.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return nil, fmt.Errorf("databricks: get_run(%d): %w", runID, err)
	}

	var parts []string
	if run.State != nil && run.State.StateMessage != "" {
		parts = append(parts, run.State.StateMessage)
	}
	for _, task := range run.Tasks {
		if task.State != nil && task.State.StateMessage != "" {
			parts = append(parts, task.State.StateMessage)
		}
	}

	return &RunDetail{ErrorMessage: strings.Join(parts, " | ")}, nil
}