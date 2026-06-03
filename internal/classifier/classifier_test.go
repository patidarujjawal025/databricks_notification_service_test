package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	c := New()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{"infra: cluster launch failed", "Cluster launch failed due to quota", "INFRA_ERROR"},
		{"infra: case insensitive", "INSUFFICIENTINSTANCECAPACITY in zone us-east", "INFRA_ERROR"},
		{"infra: executor lost", "executor lost, reason: worker lost", "INFRA_ERROR"},
		{"infra: metastore", "Metastore unavailable: timeout", "INFRA_ERROR"},
		{"code: nil pointer", "NullPointerException at line 42", "CODE_ERROR"},
		{"code: syntax", "SyntaxError: unexpected token", "CODE_ERROR"},
		{"code: empty message", "", "CODE_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, c.Classify(tt.message))
		})
	}
}