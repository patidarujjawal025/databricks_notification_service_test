package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/swiggy-private/gocommons/log"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/initialize"
)

func main() {
	sess := session.Must(session.NewSession())
	ctx := context.Background()

	handler, err := initialize.Init(ctx, sess)
	if err != nil {
		log.Errorw(ctx, "failed to initialise lambda handler", "err", err)
		return
	}

	lambda.Start(handler.Handler)
}