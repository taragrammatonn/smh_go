package mqtt

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	c mqtt.Client
}

type Config struct {
	BrokerURL string // e.g., "tcp://mqtt:1883"
	ClientID  string
	Username  string
	Password  string
	TLS       bool
}

func New(cfg Config) (*Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(cfg.BrokerURL).SetClientID(cfg.ClientID)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}
	if cfg.TLS {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}
	opts.SetOrderMatters(false)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetPingTimeout(3 * time.Second)

	client := mqtt.NewClient(opts)
	t := client.Connect()
	if ok := t.WaitTimeout(10 * time.Second); !ok || t.Error() != nil {
		return nil, t.Error()
	}
	return &Client{c: client}, nil
}

func (c *Client) Publish(topic string, payload []byte, qos byte, retain bool) error {
	t := c.c.Publish(topic, qos, retain, payload)
	t.Wait()
	return t.Error()
}

func (c *Client) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) error {
	t := c.c.Subscribe(topic, qos, cb)
	t.Wait()
	return t.Error()
}

func (c *Client) Close() {
	if c.c != nil && c.c.IsConnectionOpen() {
		c.c.Disconnect(250)
	}
}
