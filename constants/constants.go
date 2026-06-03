package constants

const (
	// AWS secret env var keys (read from .conf files)
	AwsSecretRegion                    = "AWS_SECRET_REGION"
	DatabricksTokenSecretName          = "DATABRICKS_ANALYTICS_MUMBAI_TOKEN_SECRET_NAME"
	OpsGenieAPIKeySecretName           = "OPSGENIE_API_KEY_SECRET_NAME"

	// Databricks env var populated after secret fetch
	DatabricksAnalyticsMumbaiToken = "DATABRICKS_ANALYTICS_MUMBAI_TOKEN"

	// OpsGenie env var populated after secret fetch
	OpsGenieAPIKey = "OPSGENIE_API_KEY"

	// Error categories
	ErrorCategoryInfra = "INFRA_ERROR"
	ErrorCategoryCode  = "CODE_ERROR"

	// Alert priority
	OpsGeniePriority = "P2"
)
