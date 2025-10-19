package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	mqttIface "github.com/tetragramaton/smh-go/internal/interface/mqtt"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttClient struct {
	mqttIface.API
	context.Context
}

type Config struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string
	TLS       bool
}

func LoadConfigFromEnv() (Config, error) {
	var cfg Config

	cfg.BrokerURL = os.Getenv("MQTT_URL")
	if cfg.BrokerURL == "" {
		return cfg, errors.New("missing MQTT_URL")
	}
	cfg.ClientID = os.Getenv("MQTT_CLIENT_ID")
	if cfg.ClientID == "" {
		return cfg, errors.New("missing MQTT_CLIENT_ID")
	}
	cfg.Username = os.Getenv("MQTT_USERNAME")
	cfg.Password = os.Getenv("MQTT_PASSWORD")

	if v := os.Getenv("MQTT_TLS"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid MQTT_TLS %q: %w", v, err)
		}
		cfg.TLS = b
	}

	return cfg, nil
}

func NewClient() (mqttIface.Client, error) {
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetKeepAlive(30 * time.Second).
		SetConnectTimeout(5 * time.Second).
		SetPingTimeout(3 * time.Second).
		SetOrderMatters(false)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}
	if cfg.TLS {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}

	client := mqtt.NewClient(opts)
	t := client.Connect()
	if ok := t.WaitTimeout(10 * time.Second); !ok || t.Error() != nil {
		return nil, t.Error()
	}
	return &mqttClient{
		API:     client,
		Context: ctx,
	}, nil
}

func (c mqttClient) PublishEvent(message mqttIface.Message) error {
	t := c.API.Publish(message.Topic, message.QoS, message.Retain, message.Payload)
	t.Wait()
	return t.Error()
}

func (c mqttClient) SubscribeToTopic(sub mqttIface.Subscription) error {
	t := c.API.Subscribe(sub.Topic, sub.QoS, sub.Callback)
	t.Wait()
	return t.Error()
}

func (c mqttClient) Close(quiesce uint) error {
	if c.IsConnectionOpen() {
		c.Disconnect(quiesce)
	}
	return nil
}
