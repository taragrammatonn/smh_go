package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	mq "github.com/eclipse/paho.mqtt.golang"
	"github.com/tetragramaton/smh-go/internal/ha"
	"github.com/tetragramaton/smh-go/internal/mqtt"
)

type Meta struct {
	DeviceID string   `json:"device_id"`
	Model    string   `json:"model,omitempty"`
	Area     string   `json:"area,omitempty"`
	Caps     []string `json:"caps"`
}

func main() {
	broker := getenv("MQTT_URL", "tcp://mqtt:1883")
	clientID := getenv("CLIENT_ID", "smh-core-"+time.Now().Format("150405"))
	mc, err := mqtt.New(mqtt.Config{BrokerURL: broker, ClientID: clientID})
	if err != nil {
		log.Fatalf("mqtt connect: %v", err)
	}
	defer mc.Close()

	err = mc.Subscribe("smh/+/meta", 1, func(_ mq.Client, m mq.Message) {
		var meta Meta
		if err := json.Unmarshal(m.Payload(), &meta); err != nil {
			log.Printf("bad meta: %v", err)
			return
		}
		publishDiscovery(mc, meta)
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	log.Println("smh-core up; waiting for meta...")
	select {}
}

func publishDiscovery(mc *mqtt.Client, meta Meta) {
	unique := sanitize(meta.DeviceID)
	device := &ha.Device{
		Identifiers:  []string{meta.DeviceID},
		Manufacturer: "SMH",
		Model:        meta.Model,
		Name:         meta.DeviceID,
	}

	for _, cap := range meta.Caps {
		switch cap {
		case "energy.meter":
			cfgP := &ha.SensorConfig{
				Name:        fmt.Sprintf("%s power", meta.DeviceID),
				UniqueID:    unique + "_power",
				StateTopic:  fmt.Sprintf("smh/%s/state", meta.DeviceID),
				ValueTpl:    "{{ value_json.power_w if value_json.cap == \"energy.meter\" }}",
				DeviceClass: "power",
				UnitOfMeas:  "W",
				Device:      device,
			}
			pubCfg(mc, ha.TopicSensorConfig("power_w", unique), cfgP)
			cfgE := &ha.SensorConfig{
				Name:        fmt.Sprintf("%s energy", meta.DeviceID),
				UniqueID:    unique + "_energy",
				StateTopic:  fmt.Sprintf("smh/%s/state", meta.DeviceID),
				ValueTpl:    "{{ value_json.energy_kwh if value_json.cap == \"energy.meter\" }}",
				DeviceClass: "energy",
				UnitOfMeas:  "kWh",
				Device:      device,
			}
			pubCfg(mc, ha.TopicSensorConfig("energy_kwh", unique), cfgE)

		case "sensor.frequency":
			cfg := &ha.SensorConfig{
				Name:       fmt.Sprintf("%s frequency", meta.DeviceID),
				UniqueID:   unique + "_freq",
				StateTopic: fmt.Sprintf("smh/%s/state", meta.DeviceID),
				ValueTpl:   "{{ value_json.value if value_json.cap == \"sensor.frequency\" }}",
				UnitOfMeas: "Hz",
				Device:     device,
			}
			pubCfg(mc, ha.TopicSensorConfig("frequency", unique), cfg)

		case "sensor.voltage":
			cfg := &ha.SensorConfig{
				Name:       fmt.Sprintf("%s voltage", meta.DeviceID),
				UniqueID:   unique + "_volt",
				StateTopic: fmt.Sprintf("smh/%s/state", meta.DeviceID),
				ValueTpl:   "{{ value_json.value if value_json.cap == \"sensor.voltage\" }}",
				UnitOfMeas: "V",
				Device:     device,
			}
			pubCfg(mc, ha.TopicSensorConfig("voltage", unique), cfg)
		}
	}
	log.Printf("HA discovery published for %s (%v)", meta.DeviceID, meta.Caps)
}

func pubCfg(mc *mqtt.Client, topic string, cfg *ha.SensorConfig) {
	b, err := cfg.Marshal()
	if err != nil {
		log.Printf("marshal cfg: %v", err)
		return
	}
	if err := mc.Publish(topic, b, 1, true); err != nil {
		log.Printf("publish cfg: %v", err)
	}
}

func sanitize(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	return strings.ToLower(re.ReplaceAllString(s, "_"))
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
