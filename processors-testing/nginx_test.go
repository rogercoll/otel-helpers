package nginxotel

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/cmd/ottlplayground"
	"github.com/open-telemetry/opentelemetry-collector-contrib/cmd/ottlplayground/executors"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
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

	logsMarshaler := plog.JSONMarshaler{}
	logsUnmarshaler := plog.JSONUnmarshaler{}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			inputLogs, err := golden.ReadLogs(tt.logsInput)
			require.NoError(t, err)

			jsonIntput, err := logsMarshaler.MarshalLogs(inputLogs)
			require.NoError(t, err)

			for _, config := range IngressNginxPipeline.configs {
				var executor ottlplayground.Executor
				switch config.processorType.Type() {
				case transformprocessor.NewFactory().Type():
					executor = executors.NewTransformProcessorExecutor()
				case filterprocessor.NewFactory().Type():
					executor = executors.NewFilterProcessorExecutor()
				default:
					t.Errorf("processor type %q not supported", config.processorType.String())
				}

				switch IngressNginxPipeline.signal {
				case pipeline.SignalLogs:
					jsonIntput, err = executor.ExecuteLogStatements(config.config, string(jsonIntput))
					require.NoError(t, err)

				default:
					t.Errorf("signal type %q not supported in Logs test case", IngressNginxPipeline.signal)

				}
			}

			actualLogs, err := logsUnmarshaler.UnmarshalLogs(jsonIntput)
			require.NoError(t, err)

			// err = golden.WriteLogs(t, tt.expectedLogsOutput, actualLogs)
			// require.NoError(t, err)

			expectedLogs, err := golden.ReadLogs(tt.expectedLogsOutput)
			require.NoError(t, err)

			assert.NoError(t, plogtest.CompareLogs(expectedLogs, actualLogs))
		})
	}
}
