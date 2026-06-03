package classifier

import "strings"

// infraErrorPhrases is the canonical list of phrases that indicate infrastructure-level failures.
// Matching is case-insensitive.
var infraErrorPhrases = []string{
	"cluster launch failed",
	"cluster startup failed",
	"insufficientinstancecapacity",
	"allocationfailed",
	"driver is down",
	"executor lost",
	"worker lost",
	"internal server error",
	"service unavailable",
	"connection timed out",
	"network unreachable",
	"metastore unavailable",
	"unity catalog unavailable",
	"secret scope unavailable",
	"failed to scale cluster",
	"spot instance interruption",
}

// Classifier categorises a Databricks failure message as INFRA_ERROR or CODE_ERROR.
type Classifier interface {
	Classify(message string) string
}

type classifier struct{}

// New returns the default Classifier implementation.
func New() Classifier {
	return &classifier{}
}

// Classify returns constants.ErrorCategoryInfra when the message contains a known
// infrastructure phrase; otherwise constants.ErrorCategoryCode.
func (c *classifier) Classify(message string) string {
	lowered := strings.ToLower(message)
	for _, phrase := range infraErrorPhrases {
		if strings.Contains(lowered, phrase) {
			return "INFRA_ERROR"
		}
	}
	return "CODE_ERROR"
}