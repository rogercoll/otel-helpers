package processorstest

import (
	"fmt"
	"os"

	"github.com/open-telemetry/opentelemetry-collector-contrib/cmd/ottlplayground"
	"github.com/open-telemetry/opentelemetry-collector-contrib/cmd/ottlplayground/executors"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
	"gopkg.in/yaml.v3"
)

func assertLogsPipeline(config *confmap.Conf, pipeID pipeline.ID, inputData string, expectedData string) error {
	inputLogs, err := golden.ReadLogs(inputData)
	if err != nil {
		return err
	}

	logsMarshaler := plog.JSONMarshaler{}
	logsUnmarshaler := plog.JSONUnmarshaler{}
	jsonIntput, err := logsMarshaler.MarshalLogs(inputLogs)
	if err != nil {
		return err
	}

	processors := config.Get("service::pipelines::" + pipeID.String() + "::processors")
	processorsList := (*&processors).([]any)

	for _, rawProcessor := range processorsList {
		id := (*&rawProcessor).(string)
		var processor component.ID
		err := processor.UnmarshalText([]byte(id))
		if err != nil {
			return err
		}
		var executor ottlplayground.Executor
		switch processor.Type() {
		case transformprocessor.NewFactory().Type():
			executor = executors.NewTransformProcessorExecutor()
		case filterprocessor.NewFactory().Type():
			executor = executors.NewFilterProcessorExecutor()
		default:
			return fmt.Errorf("processor type %q not supported", processor.Type().String())
		}

		switch pipeID.Signal() {
		case pipeline.SignalLogs:
			processorRawConfig, err := yaml.Marshal(config.Get("processors::" + processor.String()))
			if err != nil {
				return err
			}

			jsonIntput, err = executor.ExecuteLogStatements(string(processorRawConfig), string(jsonIntput))
			if err != nil {
				return err
			}

		default:
			fmt.Errorf("signal type %q not supported in Logs test case", IngressNginxPipeline.signal)

		}
	}

	actualLogs, err := logsUnmarshaler.UnmarshalLogs(jsonIntput)
	if err != nil {
		return err
	}

	expectedLogs, err := golden.ReadLogs(expectedData)
	if err != nil {
		return err
	}

	return plogtest.CompareLogs(expectedLogs, actualLogs)
}

func FromCollectorConfig(configPath string, pipeID string, inputData string, expectedData string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("unable to read the file: %w", err)
	}

	collectorConfig, err := confmap.NewRetrievedFromYAML(content)
	if err != nil {
		return err
	}
	collectorConf, err := collectorConfig.AsConf()
	if err != nil {
		return err
	}
	switch pipeline.MustNewID(pipeID).Signal() {
	case pipeline.SignalLogs:
		return assertLogsPipeline(collectorConf, pipeline.MustNewID(pipeID), inputData, expectedData)
	}
	return nil
}
