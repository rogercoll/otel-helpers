package nginxotel

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"gopkg.in/yaml.v3"
)

type ProcessorInstance struct {
	processorType component.ID
	config        string
}

type ProcessorsPipeline struct {
	signal  pipeline.Signal
	configs []ProcessorInstance
	yaml.Marshaler
}

// TODO: return collector's ready configuration
// yaml.MarshalYAML interface
func (p *ProcessorsPipeline) MarshalYAML() (interface{}, error) {
	return nil, errors.New("TODO")
}
