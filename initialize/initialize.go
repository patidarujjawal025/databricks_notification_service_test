package initialize

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/swiggy-private/gocommons/log"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/conf"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/constants"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/classifier"
	dbrClient "github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/databricks"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/opsgenie"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/lambdaHandler"
	"github.com/swiggy-private/sql-anti-patterns-checker/commons/secrets"
)

// Init wires up all dependencies and returns a ready-to-use lambda Handler.
func Init(ctx context.Context, sess *session.Session) (*lambdaHandler.Handler, error) {
	if err := initSecrets(ctx, sess); err != nil {
		log.Errorw(ctx, "failed to initialise secrets", "err", err)
		return nil, err
	}

	cfg, err := initConfig(ctx)
	if err != nil {
		return nil, err
	}

	dbr, err := dbrClient.New(dbrClient.Config{
		Host:  cfg.Databricks.AnalyticsHost,
		Token: cfg.Databricks.AnalyticsToken,
	})
	if err != nil {
		log.Errorw(ctx, "failed to create databricks client", "err", err)
		return nil, err
	}

	notif := opsgenie.New(opsgenie.Config{
		APIKey:      cfg.OpsGenie.APIKey,
		URL:         cfg.OpsGenie.URL,
		InfraOncall: cfg.OpsGenie.InfraOncall,
		CodeOncall:  cfg.OpsGenie.CodeOncall,
	})

	cls := classifier.New()

	return lambdaHandler.New(dbr, cls, notif), nil
}

// initSecrets fetches sensitive values from AWS Secrets Manager and injects
// them into the process environment so that Viper can expand them in config.yaml.
func initSecrets(ctx context.Context, sess *session.Session) error {
	region := os.Getenv(constants.AwsSecretRegion)

	secretClient := secrets.GetSecretsClient(sess, region)

	dbrTokenSecretName := os.Getenv(constants.DatabricksTokenSecretName)
	dbrToken, err := secretClient.GetSecret(ctx, dbrTokenSecretName)
	if err != nil {
		log.Errorw(ctx, "failed to fetch databricks token secret", "err", err)
		return err
	}
	os.Setenv(constants.DatabricksAnalyticsMumbaiToken, dbrToken)

	opsGenieSecretName := os.Getenv(constants.OpsGenieAPIKeySecretName)
	opsGenieKey, err := secretClient.GetSecret(ctx, opsGenieSecretName)
	if err != nil {
		log.Errorw(ctx, "failed to fetch opsgenie api key secret", "err", err)
		return err
	}
	os.Setenv(constants.OpsGenieAPIKey, opsGenieKey)

	return nil
}

func initConfig(ctx context.Context) (conf.Configuration, error) {
	if err := conf.Initialize("."); err != nil {
		log.Errorw(ctx, "failed to initialise config", "err", err)
		return conf.Configuration{}, err
	}
	return conf.GetConfig(), nil
}