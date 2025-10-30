package main

import (
	"encoding/json"
	"fmt"
	mq "github.com/eclipse/paho.mqtt.golang"
	"github.com/tetragramaton/smh-go/internal/client/ha"
	"github.com/tetragramaton/smh-go/internal/interface/mqtt"
	"log"
	"regexp"
	"strings"
)

type Meta struct {
	DeviceID string   `json:"device_id"`
	Model    string   `json:"model,omitempty"`
	Area     string   `json:"area,omitempty"`
	Caps     []string `json:"caps"`
}

func main() {
	handler, err := InitMainHandler()
	if err != nil {
		log.Fatal(err)
	}
	handler.Handle()
}

func (h *MainHandler) Handle() {
	//broker := getenv("MQTT_URL", "tcp://mqtt:1883")
	//clientID := getenv("CLIENT_ID", "smh-core-"+time.Now().Format("150405"))
	//mc, err := mqtt.New(mqtt.Config{BrokerURL: broker, ClientID: clientID})
	//if err != nil {
	//	log.Fatalf("mqtt connect: %v", err)
	//}
	//defer mc.Close()
	subscription := mqtt.Subscription{
		Topic: "smh/+/meta",
		QoS:   1,
		Callback: func(_ mq.Client, m mq.Message) {
			var meta Meta
			if err := json.Unmarshal(m.Payload(), &meta); err != nil {
				log.Printf("bad meta: %v", err)
				return
			}
			publishDiscovery(h, meta)
		},
	}
	err := h.MQQTClient.SubscribeToTopic(subscription)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	log.Println("smh-core up; waiting for meta...")
	select {}
}

func publishDiscovery(mc *MainHandler, meta Meta) {
	unique := sanitize(meta.DeviceID)
	device := &ha.Device{
		Identifiers:  []string{meta.DeviceID},
		Manufacturer: "SMH",
		Model:        meta.Model,
		Name:         meta.DeviceID,
	}

	for _, c := range meta.Caps {
		const stateFormat = "smh/%s/state"
		switch c {
		case "energy.meter":
			cfgP := &ha.SensorConfig{
				Name:        fmt.Sprintf("%s power", meta.DeviceID),
				UniqueID:    unique + "_power",
				StateTopic:  fmt.Sprintf(stateFormat, meta.DeviceID),
				ValueTpl:    "{{ value_json.power_w if value_json.c == \"energy.meter\" }}",
				DeviceClass: "power",
				UnitOfMeas:  "W",
				Device:      device,
			}
			pubCfg(mc, ha.TopicSensorConfig("power_w", unique), cfgP)
			cfgE := &ha.SensorConfig{
				Name:        fmt.Sprintf("%s energy", meta.DeviceID),
				UniqueID:    unique + "_energy",
				StateTopic:  fmt.Sprintf(stateFormat, meta.DeviceID),
				ValueTpl:    "{{ value_json.energy_kwh if value_json.c == \"energy.meter\" }}",
				DeviceClass: "energy",
				UnitOfMeas:  "kWh",
				Device:      device,
			}
			pubCfg(mc, ha.TopicSensorConfig("energy_kwh", unique), cfgE)

		case "sensor.frequency":
			cfg := &ha.SensorConfig{
				Name:       fmt.Sprintf("%s frequency", meta.DeviceID),
				UniqueID:   unique + "_freq",
				StateTopic: fmt.Sprintf(stateFormat, meta.DeviceID),
				ValueTpl:   "{{ value_json.value if value_json.c == \"sensor.frequency\" }}",
				UnitOfMeas: "Hz",
				Device:     device,
			}
			pubCfg(mc, ha.TopicSensorConfig("frequency", unique), cfg)

		case "sensor.voltage":
			cfg := &ha.SensorConfig{
				Name:       fmt.Sprintf("%s voltage", meta.DeviceID),
				UniqueID:   unique + "_volt",
				StateTopic: fmt.Sprintf(stateFormat, meta.DeviceID),
				ValueTpl:   "{{ value_json.value if value_json.c == \"sensor.voltage\" }}",
				UnitOfMeas: "V",
				Device:     device,
			}
			pubCfg(mc, ha.TopicSensorConfig("voltage", unique), cfg)
		}
	}
	log.Printf("HA discovery published for %s (%v)", meta.DeviceID, meta.Caps)
}

func pubCfg(mc *MainHandler, topic string, cfg *ha.SensorConfig) {
	b, err := cfg.Marshal()
	if err != nil {
		log.Printf("marshal cfg: %v", err)
		return
	}
	if err := mc.MQQTClient.PublishEvent(mqtt.Message{
		Topic:   topic,
		Payload: b,
		QoS:     1,
		Retain:  true,
	}); err != nil {
		log.Printf("publish cfg: %v", err)
	}
}

func sanitize(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	return strings.ToLower(re.ReplaceAllString(s, "_"))
}
