package nginx

import (
	"testing"

	"github.com/rogercoll/processorstest"
	"github.com/stretchr/testify/assert"
)

func TestLogsPipeline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		logsInput          string
		expectedLogsOutput string
	}{
		{
			name:               "Access log ipv4",
			logsInput:          "testdata/filelog-logs.yaml",
			expectedLogsOutput: "testdata/filelog-logs-expected.yaml",
		},
		{
			name:               "Error log from config.go",
			logsInput:          "testdata/filelog-error-logs.yaml",
			expectedLogsOutput: "testdata/filelog-error-logs-expected.yaml",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := processorstest.FromCollectorConfig("./config.yaml", "logs", tt.logsInput, tt.expectedLogsOutput)
			assert.NoError(t, err)
		})
	}
}
