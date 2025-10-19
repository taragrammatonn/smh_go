package main

import (
	"encoding/json"
	"time"
)

// MQTTPublisher is a minimal interface for tests and real MQTT client.
type MQTTPublisher interface {
	Publish(topic string, payload []byte, qos byte, retain bool) error
}

// PublishOnce reads mapped registers and publishes normalized states once.
// Used by tests; main loop uses similar logic on a ticker.
func PublishOnce(h handler, client MQTTPublisher, cfg envCfg, now int64) {
	// frequency
	if v, err := h.readFloat(cfg.MapCfg.Frequency.Addr, cfg.MapCfg.Frequency.Scale, cfg.MapCfg.Frequency.Holding); err == nil {
		state := map[string]interface{}{"ts": now, "cap": "sensor.frequency", "value": round(v, 2), "unit": "Hz"}
		client.Publish("smh/"+cfg.DeviceID+"/state", must(json.Marshal(state)), 1, false)
	}
	// voltage
	if v, err := h.readFloat(cfg.MapCfg.Voltage.Addr, cfg.MapCfg.Voltage.Scale, cfg.MapCfg.Voltage.Holding); err == nil {
		state := map[string]interface{}{"ts": now, "cap": "sensor.voltage", "value": round(v, 1), "unit": "V"}
		client.Publish("smh/"+cfg.DeviceID+"/state", must(json.Marshal(state)), 1, false)
	}
	// energy.meter (optional fields if mapped)
	payload := map[string]interface{}{"ts": now, "cap": "energy.meter"}
	ok := false
	if cfg.MapCfg.Power.Addr != 0 {
		if v, err := h.readFloat(cfg.MapCfg.Power.Addr, cfg.MapCfg.Power.Scale, cfg.MapCfg.Power.Holding); err == nil {
			payload["power_w"] = round(v, 1)
			ok = true
		}
	}
	if cfg.MapCfg.Energy.Addr != 0 {
		if v, err := h.readFloat(cfg.MapCfg.Energy.Addr, cfg.MapCfg.Energy.Scale, cfg.MapCfg.Energy.Holding); err == nil {
			payload["energy_kwh"] = round(v, 6)
			ok = true
		}
	}
	if ok {
		client.Publish("smh/"+cfg.DeviceID+"/state", must(json.Marshal(payload)), 1, false)
	}
}

// helper for tests to fabricate "now"
func nowUnix() int64 { return time.Now().Unix() }
