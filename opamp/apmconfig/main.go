package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/open-telemetry/opamp-go/client/types"
)

var _ types.Logger = &Logger{}

type Logger struct {
	Logger *log.Logger
}

func (l *Logger) Debugf(_ context.Context, format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

func (l *Logger) Errorf(_ context.Context, format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func RandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func main() {
	var opampEndpoint string
	flag.StringVar(&opampEndpoint, "opamp_endpoint", "http://127.0.0.1:4320/v1/opamp", "OpAMP HTTP Endpoint")

	var agentService string
	flag.StringVar(&agentService, "service.name", "io.opentelemetry.collector", "Agent Type String")

	var agentEnv string
	flag.StringVar(&agentEnv, "service.version", "1.0.0", "Agent Version String")

	flag.Parse()

	stdLogger := log.Default()

	numAgents := 2
	agents := make([]*Agent, numAgents)
	for i := 0; i < numAgents; i++ {
		agents[i] = NewAgent(&Logger{stdLogger}, opampEndpoint, agentService+RandomString(i), agentEnv)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	for i := 0; i < numAgents; i++ {
		agents[i].Shutdown()
	}
}
