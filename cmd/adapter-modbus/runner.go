package main

import (
	"github.com/tetragramaton/smh-go/internal/interface/modbus"
	"log"

	//"encoding/json"
	mqttIface "github.com/tetragramaton/smh-go/internal/interface/mqtt"
)

// mqttClient is a minimal interface for tests and real MQTT client.
type mqttClient interface {
	mqttIface.API
}

// PublishOnce reads mapped registers and publishes normalized states once.
// Used by tests; the main loop uses similar logic on a ticker.
func PublishOnce(h MainHandler, cfg envCfg, now int64) {
	// frequency
	freqParam := modbus.RegisterParam{
		Addr:    cfg.MapCfg.Frequency.Addr,
		Scale:   cfg.MapCfg.Frequency.Scale,
		Holding: cfg.MapCfg.Frequency.Holding,
	}
	const path = "/state"
	if v, err := h.readFloat(freqParam); err == nil {
		sensorState := SensorState{
			Ts:    now,
			Cap:   "sensor.frequency",
			Unit:  "Hz",
			Value: round(v, 2),
		}
		err := h.publishEvent(cfg, sensorState, path)
		if err != nil {
			log.Printf("Error publishing sensor state: %v", err)
		}
	}
	// voltage
	voltParam := modbus.RegisterParam{
		Addr:    cfg.MapCfg.Voltage.Addr,
		Scale:   cfg.MapCfg.Voltage.Scale,
		Holding: cfg.MapCfg.Voltage.Holding,
	}
	if v, err := h.readFloat(voltParam); err == nil {
		sensorState := SensorState{
			Ts:    now,
			Cap:   "sensor.voltage",
			Unit:  "V",
			Value: round(v, 1),
		}
		err := h.publishEvent(cfg, sensorState, path)
		if err != nil {
			log.Printf("Error publishing sensor state: %v", err)
		}
	}
	// energy.meter (optional fields if mapped)
	var pwPtr, ekPtr *float64
	ok := false

	if cfg.MapCfg.Power.Addr != 0 {
		p := modbus.RegisterParam{
			Addr:    cfg.MapCfg.Power.Addr,
			Scale:   cfg.MapCfg.Power.Scale,
			Holding: cfg.MapCfg.Power.Holding,
		}
		if v, err := h.readFloat(p); err == nil {
			pw := round(v, 1)
			pwPtr = pw
			ok = true
		} else {
			log.Printf("read power: %v", err)
		}
	}

	if cfg.MapCfg.Energy.Addr != 0 {
		p := modbus.RegisterParam{
			Addr:    cfg.MapCfg.Energy.Addr,
			Scale:   cfg.MapCfg.Energy.Scale,
			Holding: cfg.MapCfg.Energy.Holding,
		}
		if v, err := h.readFloat(p); err == nil {
			ek := round(v, 6)
			ekPtr = ek
			ok = true
		} else {
			log.Printf("read energy: %v", err)
		}
	}

	if ok {
		state := SensorState{
			Ts:        now,
			Cap:       "energy.meter",
			PowerW:    pwPtr,
			EnergyKwh: ekPtr,
		}
		if err := h.publishEvent(cfg, state, path); err != nil {
			log.Printf("publish state: %v", err)
		}
	}

}
