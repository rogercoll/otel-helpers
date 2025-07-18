package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/gofrs/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
)

const localConfig = `
transactions_max_spans: 123
`

type Agent struct {
	logger types.Logger

	agentType    string
	agentVersion string

	effectiveConfig string

	instanceId uuid.UUID

	agentDescription *protobufs.AgentDescription

	opampClient client.OpAMPClient

	remoteConfigStatus *protobufs.RemoteConfigStatus
}

func NewAgent(logger types.Logger, opamp_endpoint string, agentType string, agentVersion string) *Agent {
	agent := &Agent{
		effectiveConfig: localConfig,
		logger:          logger,
		agentType:       agentType,
		agentVersion:    agentVersion,
		remoteConfigStatus: &protobufs.RemoteConfigStatus{
			LastRemoteConfigHash: []byte("heeya"),
		},
	}

	agent.createAgentIdentity()
	agent.logger.Debugf(context.Background(), "Agent starting, id=%v, type=%s, version=%s.",
		agent.instanceId, agentType, agentVersion)

	if err := agent.connect(opamp_endpoint); err != nil {
		agent.logger.Errorf(context.Background(), "Cannot connect OpAMP client: %v", err)
		return nil
	}

	return agent
}

func (agent *Agent) connect(opampEndpoint string) error {
	agent.opampClient = client.NewHTTP(agent.logger)

	settings := types.StartSettings{
		Header: map[string][]string{
			"Authorization": {
				"sure, not a key my dear bot",
			},
		},
		OpAMPServerURL: opampEndpoint,
		InstanceUid:    types.InstanceUid(agent.instanceId),
		Callbacks: types.Callbacks{
			OnConnect: func(ctx context.Context) {
				agent.logger.Debugf(ctx, "Connected to the server.")
			},
			OnConnectFailed: func(ctx context.Context, err error) {
				agent.logger.Errorf(ctx, "Failed to connect to the server: %v", err)
			},
			OnError: func(ctx context.Context, err *protobufs.ServerErrorResponse) {
				agent.logger.Errorf(ctx, "Server returned an error response: %v", err.ErrorMessage)
			},
			SaveRemoteConfigStatus: func(_ context.Context, status *protobufs.RemoteConfigStatus) {
				agent.remoteConfigStatus = status
			},
			GetEffectiveConfig: func(ctx context.Context) (*protobufs.EffectiveConfig, error) {
				fmt.Println("get effective config")
				return &protobufs.EffectiveConfig{
					ConfigMap: &protobufs.AgentConfigMap{
						ConfigMap: map[string]*protobufs.AgentConfigFile{
							"": {Body: []byte(agent.effectiveConfig)},
						},
					},
				}, nil
			},
			OnMessage:                 agent.onMessage,
			OnOpampConnectionSettings: nil,
		},
		RemoteConfigStatus: agent.remoteConfigStatus,
		Capabilities: protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsOwnMetrics |
			protobufs.AgentCapabilities_AgentCapabilities_AcceptsOpAMPConnectionSettings,
	}

	err := agent.opampClient.SetAgentDescription(agent.agentDescription)
	if err != nil {
		return err
	}

	agent.logger.Debugf(context.Background(), "Starting OpAMP client...")

	err = agent.opampClient.Start(context.Background(), settings)
	if err != nil {
		return err
	}

	agent.logger.Debugf(context.Background(), "OpAMP Client started.")

	return nil
}

func (agent *Agent) disconnect(ctx context.Context) {
	agent.logger.Debugf(ctx, "Disconnecting from server...")
	agent.opampClient.Stop(ctx)
}

func (agent *Agent) createAgentIdentity() {
	// Generate instance id.
	uid, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	agent.instanceId = uid

	hostname, _ := os.Hostname()

	// Create Agent description.
	agent.agentDescription = &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			{
				Key: "service.name",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: agent.agentType},
				},
			},
			{
				Key: "service.version",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: agent.agentVersion},
				},
			},
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			{
				Key: "os.type",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{
						StringValue: runtime.GOOS,
					},
				},
			},
			{
				Key: "host.name",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{
						StringValue: hostname,
					},
				},
			},
		},
	}
}

func (agent *Agent) onMessage(ctx context.Context, msg *types.MessageData) {
	agent.logger.Debugf(context.Background(), "New message received")
	if msg.RemoteConfig != nil {
		_, err := agent.applyRemoteConfig(msg.RemoteConfig)
		if err != nil {
			agent.opampClient.SetRemoteConfigStatus(
				&protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         err.Error(),
				},
			)
		} else {
			agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
			})
		}
	}

	// if configChanged {
	// 	err := agent.opampClient.UpdateEffectiveConfig(ctx)
	// 	if err != nil {
	// 		agent.logger.Errorf(ctx, err.Error())
	// 	}
	// }
}

func (agent *Agent) applyRemoteConfig(config *protobufs.AgentRemoteConfig) (configChanged bool, err error) {
	if config == nil {
		return false, nil
	}

	newEffectiveConfig := string(config.GetConfig().ConfigMap["elastic"].Body)
	agent.logger.Debugf(context.Background(), "Received remote config from server, hash=%x config=%v", config.ConfigHash, newEffectiveConfig)

	configChanged = false
	if agent.effectiveConfig != newEffectiveConfig {
		agent.logger.Debugf(context.Background(), "Effective config changed. Need to report to server.")
		agent.effectiveConfig = newEffectiveConfig
		configChanged = true
	}

	return configChanged, nil
}

func (agent *Agent) Shutdown() {
	agent.logger.Debugf(context.Background(), "Agent shutting down...")
	if agent.opampClient != nil {
		_ = agent.opampClient.Stop(context.Background())
	}
}
