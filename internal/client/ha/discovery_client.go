package ha

import (
	"encoding/json"
	"fmt"
)

type Device struct {
	Identifiers  []string `json:"identifiers,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Model        string   `json:"model,omitempty"`
	Name         string   `json:"name,omitempty"`
}

type SensorConfig struct {
	Name         string                 `json:"name"`
	UniqueID     string                 `json:"unique_id"`
	StateTopic   string                 `json:"state_topic"`
	ValueTpl     string                 `json:"value_template,omitempty"`
	DeviceClass  string                 `json:"device_class,omitempty"`
	UnitOfMeas   string                 `json:"unit_of_measurement,omitempty"`
	Device       *Device                `json:"device,omitempty"`
	QoS          int                    `json:"qos,omitempty"`
	Availability []map[string]string    `json:"availability,omitempty"`
	Extra        map[string]interface{} `json:"-"`
}

func (c *SensorConfig) Marshal() ([]byte, error) {
	type alias SensorConfig
	a := alias(*c)
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	if c.Extra != nil {
		var base map[string]interface{}
		if err := json.Unmarshal(b, &base); err != nil {
			return nil, err
		}
		for k, v := range c.Extra {
			base[k] = v
		}
		return json.Marshal(base)
	}
	return b, nil
}

func TopicSensorConfig(cap, unique string) string {
	return fmt.Sprintf("homeassistant/sensor/%s/%s/config", unique, cap)
}
