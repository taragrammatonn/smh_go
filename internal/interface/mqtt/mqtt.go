package mqtt

import mqtt "github.com/eclipse/paho.mqtt.golang"

type Message struct {
	Topic   string `json:"topic"`
	Payload []byte `json:"payload"`
	QoS     byte   `json:"qos"`
	Retain  bool   `json:"retain"`
}

type Subscription struct {
	Topic    string              `json:"topic"`
	QoS      byte                `json:"qos"`
	Callback mqtt.MessageHandler `json:"-"`
}

type Client interface {
	API
	PublishEvent(message Message) error
	SubscribeToTopic(subscription Subscription) error
	Close(quiesce uint) error
}

type API interface {
	Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token
	Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token
	Disconnect(quiesce uint)
	IsConnectionOpen() bool
}
