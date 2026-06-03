//go:build ignore

// local_main.go is a developer-only entrypoint for running the service locally.
//
// Steps to test end-to-end:
//
//  1. Fill in the constants below (databricksToken, opsGenieAPIKey).
//
//  2. Start the server:
//     go run local_main.go
//
//  3. In a separate terminal, open a public tunnel to port 8080:
//     ssh -o StrictHostKeyChecking=no -R 80:localhost:8080 localhost.run
//     Note the public URL printed, e.g. https://abc123.lhr.life
//
//  4. Configure the Databricks job notification webhook to:
//     https://<tunnel-url>/webhook
//     (Settings → Job → Edit → Notifications → Webhook → On Failure)
//
//  5. To manually test without a real Databricks trigger:
//     curl -X POST https://<tunnel-url>/webhook \
//       -H 'Content-Type: application/json' \
//       -d '{"event_type":"jobs.on_failure","run":{"run_id":<run_id>},"job":{"job_id":<job_id>,"name":"<job_name>"}}'
//
//  6. Health check:
//     curl https://<tunnel-url>/health
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/classifier"
	dbrClient "github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/databricks"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/internal/opsgenie"
	"github.com/swiggy-private/sql-anti-patterns-checker/Databricks_Notification_Service/lambdaHandler"
)

// ---- Local dev config — fill in before running ----
const (
	databricksHost  = "https://swiggy-analytics-mumbai.cloud.databricks.com/"
	databricksToken = "<dbr_token>"

	opsGenieAPIKey  = "<integration_api_key>"
	opsGenieURL     = "https://api.opsgenie.com/v2/alerts"
	infraOncall     = "ujjawal.patidar@swiggy.in"
	codeOncall      = "parshant.sharma@swiggy.in"

	serverPort = "8080"
)

// ---------------------------------------------------

func main() {
	dbr, err := dbrClient.New(dbrClient.Config{
		Host:  databricksHost,
		Token: databricksToken,
	})
	if err != nil {
		log.Fatalf("databricks client: %v", err)
	}

	notif := opsgenie.New(opsgenie.Config{
		APIKey:      opsGenieAPIKey,
		URL:         opsGenieURL,
		InfraOncall: infraOncall,
		CodeOncall:  codeOncall,
	})

	h := lambdaHandler.New(dbr, classifier.New(), notif)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}
		resp, _ := h.Handler(r.Context(), events.APIGatewayProxyRequest{Body: string(body)})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		fmt.Fprint(w, resp.Body)
	})

	addr := ":" + serverPort
	log.Printf("dbr-notifier listening on http://localhost%s", addr)
	log.Printf("  POST /webhook  — Databricks failure webhook")
	log.Printf("  GET  /health   — liveness check")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}