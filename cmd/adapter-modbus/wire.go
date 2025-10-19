//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/tetragramaton/smh-go/internal/client/modbus"
	"github.com/tetragramaton/smh-go/internal/client/mqtt"
	modbusClient "github.com/tetragramaton/smh-go/internal/interface/modbus"
	mqttIface "github.com/tetragramaton/smh-go/internal/interface/mqtt"
)

type MainHandler struct {
	MQQTClient   mqttIface.Client
	ModbusClient modbusClient.Client
}

func NewMainHandler(
	mqttClient mqttIface.Client,
	modbusClient modbusClient.Client,
) *MainHandler {
	return &MainHandler{
		MQQTClient:   mqttClient,
		ModbusClient: modbusClient,
	}
}

func InitMainHandler() (*MainHandler, error) {
	wire.Build(
		NewMainHandler,
		ProvideMqttClient,
		ProvideNewModbusClient,
	)
	return nil, nil // wire will generate the result
}

func ProvideMqttClient() (mqttIface.Client, error) {
	return mqtt.NewClient()
}

func ProvideNewModbusClient() (modbusClient.Client, error) {
	return modbus.NewHandler()
}
