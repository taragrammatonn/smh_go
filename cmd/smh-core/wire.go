//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/tetragramaton/smh-go/internal/client/mqtt"
	mqttIface "github.com/tetragramaton/smh-go/internal/interface/mqtt"
)

type MainHandler struct {
	MQQTClient mqttIface.Client
}

func NewMainHandler(
	mqttClient mqttIface.Client,
) *MainHandler {
	return &MainHandler{
		MQQTClient: mqttClient,
	}
}

func InitMainHandler() (*MainHandler, error) {
	wire.Build(
		NewMainHandler,
		ProvideMqttClient,
	)
	return nil, nil // wire will generate the result
}

func ProvideMqttClient() (mqttIface.Client, error) {
	return mqtt.NewClient()
}
