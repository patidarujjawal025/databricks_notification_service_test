# Databricks Notification Service — Solutioning Document

## 1. Overview

The Databricks Notification Service is a serverless Go application that acts as an intelligent alert router for Databricks job failures. When a Databricks job fails, the service fetches the failure details, classifies the root cause as either an infrastructure issue or a code issue, routes the OpsGenie alert to the correct on-call team, and persists failure metrics into AWS OpenSearch for trend analysis.

---

## 2. Functional Components

### 2.1 Webhook Receiver (Lambda Handler)
- Entry point of the service, exposed via AWS API Gateway.
- Accepts `POST /webhook` with a Databricks system notification payload.
- Validates the request: checks for non-empty body, valid JSON, non-zero `run_id`, and filters only `jobs.on_failure` events — all other event types (start, success) are silently ignored.

### 2.2 Databricks Client
- Calls the Databricks Jobs API (`GET /api/2.1/jobs/runs/get`) using the `run_id` from the webhook payload.
- Extracts the consolidated error message from both the run-level state and individual task states.
- Authenticated via a token fetched from AWS Secrets Manager at cold-start.

### 2.3 Error Classifier
- Stateless component that inspects the error message against a curated list of known infrastructure failure phrases (e.g. `cluster launch failed`, `insufficientinstancecapacity`, `executor lost`, `metastore unavailable`, spot interruptions, etc.).
- Returns one of two categories:
  - `INFRA_ERROR` — cluster, capacity, network, or platform-level failures.
  - `CODE_ERROR` — application logic, data, or configuration failures.

### 2.4 OpsGenie Notifier
- Constructs and dispatches an OpsGenie alert with:
  - **Message**: human-readable summary with job name and run ID.
  - **Alias**: `dbr-job-<job_id>` — used by OpsGenie for alert deduplication (repeated failures update the same alert instead of opening new ones).
  - **Priority**: P2.
  - **Responders**: routed based on error category:
    - `INFRA_ERROR` → Compute / Infrastructure on-call.
    - `CODE_ERROR` → BL-wise (Business Line) on-call.
  - **Details**: `job_id`, `job_name`, `run_id`, `error_category` for quick triage.

### 2.5 OpenSearch Sink *(planned)*
- After classification, the failure document (job metadata + error message + category + timestamp) is indexed into AWS OpenSearch.
- Enables historical trend analysis: recurring infra failures, flaky jobs, BL-wise failure rates.
- Powers dashboards to proactively identify jobs that need hardening before they become incidents.

### 2.6 Secrets & Config
- Sensitive credentials (Databricks token, OpsGenie API key) are stored in AWS Secrets Manager and fetched once at Lambda cold-start via `initialize.Init`.
- Non-sensitive config (host URLs, on-call emails, port) is managed via Viper + `config.yaml`, with environment-specific overrides in `conf/integ.conf` and `conf/prod.conf`.

---

## 3. Production Workflow

```
Databricks Job Fails
        │
        ▼
Databricks Webhook (system notification)
        │  POST {event_type, run_id, job_id, job_name}
        ▼
AWS API Gateway  ──────────────────────────────────────────────┐
        │                                                       │
        ▼                                                  (auth / rate
AWS Lambda (this service)                                   limiting)
        │
        ├─ 1. Validate payload (event_type == jobs.on_failure, run_id present)
        │
        ├─ 2. Fetch run details  ──►  Databricks Jobs API
        │        └─ Extract error message from run state + task states
        │
        ├─ 3. Classify error
        │        ├─ INFRA_ERROR  (cluster/capacity/network phrases matched)
        │        └─ CODE_ERROR   (all other failures)
        │
        ├─ 4. Route OpsGenie alert
        │        ├─ INFRA_ERROR  ──►  Compute on-call  (ujjawal.patidar@swiggy.in)
        │        └─ CODE_ERROR   ──►  BL on-call        (parshant.sharma@swiggy.in)
        │
        ├─ 5. Persist to AWS OpenSearch  *(planned)*
        │        └─ Index: { job_id, job_name, run_id, error_category,
        │                    error_message, timestamp }
        │
        └─ 6. Return JSON response to API Gateway
                 { job_name, job_id, run_id, error_message, error_category }
```

---

## 4. Infrastructure Topology

```
┌──────────────────────────────────────────────────────────────┐
│  Databricks Workspace                                        │
│  Job Notification Webhook  ──►  AWS API Gateway URL         │
└──────────────────────────────────────────────────────────────┘
                                         │
                                         ▼
                               ┌─────────────────┐
                               │  AWS API Gateway │
                               └────────┬────────┘
                                        │
                                        ▼
                               ┌─────────────────┐
                               │   AWS Lambda     │  (Go binary, 128 MB, 300s timeout)
                               │  dbr-notifier    │
                               └──┬──────────┬───┘
                                  │          │
                    ┌─────────────┘          └──────────────────┐
                    ▼                                            ▼
          ┌──────────────────┐                       ┌─────────────────────┐
          │  Databricks API  │                       │  AWS Secrets Manager │
          │  (run details)   │                       │  - DBR token         │
          └──────────────────┘                       │  - OpsGenie key      │
                                                     └─────────────────────┘
                    │
                    ▼
          ┌──────────────────┐
          │  OpsGenie        │
          │  Alert API       │
          └──────────────────┘

          ┌──────────────────┐
          │  AWS OpenSearch  │  ← failure documents indexed here  *(planned)*
          │  (metrics sink)  │
          └──────────────────┘
```

---

## 5. Error Classification Logic

| Matched phrase (case-insensitive) | Category |
|-----------------------------------|----------|
| `cluster launch failed` | INFRA_ERROR |
| `cluster startup failed` | INFRA_ERROR |
| `insufficientinstancecapacity` | INFRA_ERROR |
| `allocationfailed` | INFRA_ERROR |
| `driver is down` | INFRA_ERROR |
| `executor lost` / `worker lost` | INFRA_ERROR |
| `internal server error` | INFRA_ERROR |
| `service unavailable` | INFRA_ERROR |
| `connection timed out` | INFRA_ERROR |
| `network unreachable` | INFRA_ERROR |
| `metastore unavailable` | INFRA_ERROR |
| `unity catalog unavailable` | INFRA_ERROR |
| `secret scope unavailable` | INFRA_ERROR |
| `failed to scale cluster` | INFRA_ERROR |
| `spot instance interruption` | INFRA_ERROR |
| *(anything else)* | CODE_ERROR |

---

## 6. Alert Routing

| Error Category | OpsGenie Responder | Intent |
|----------------|--------------------|--------|
| `INFRA_ERROR` | Compute / Infra on-call | Platform team investigates cluster/capacity issues |
| `CODE_ERROR` | BL on-call | Owning team investigates job logic / data issues |

OpsGenie deduplication key: `dbr-job-<job_id>` — multiple failures of the same job within an alert window update the existing alert rather than flooding the on-call with duplicates.

---

## 7. Local Development

```bash
# 1. Fill in credentials in local_main.go (databricksToken, opsGenieAPIKey)

# 2. Start the server
go run local_main.go

# 3. Expose it publicly via tunnel (separate terminal)
ssh -o StrictHostKeyChecking=no -R 80:localhost:8080 localhost.run

# 4. Simulate a webhook
curl -X POST http://localhost:8080/webhook \
  -H 'Content-Type: application/json' \
  -d '{
    "event_type": "jobs.on_failure",
    "run": {"run_id": <run_id>},
    "job": {"job_id": <job_id>, "name": "<job_name>"}
  }'

# 5. Health check
curl http://localhost:8080/health
```

---

## 8. Deployment

- Packaged as a Docker image (see `Dockerfile`), built from `cmd/main.go`.
- Deployed as an AWS Lambda function via the CI/CD pipeline defined in `app.yaml`.
- Pipeline stages: CI (unit tests + coverage + Docker build) → UAT (staging) → Production.
- Secrets are injected at runtime from AWS Secrets Manager; no credentials are baked into the image.

---

## 9. Planned Enhancements

| Enhancement | Value |
|-------------|-------|
| AWS OpenSearch sink | Persist every failure document; enables dashboards, SLA tracking, and early detection of recurring failures |
| BL-wise on-call routing | Map `job_name` prefix or tag to the owning BL's OpsGenie schedule instead of a single code on-call |
| Classifier expansion | Add more phrase patterns as new failure modes are discovered in OpenSearch |
| Retry / dead-letter queue | If OpsGenie or OpenSearch is temporarily unavailable, re-queue the event via SQS DLQ instead of dropping it |