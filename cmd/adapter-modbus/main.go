package main

import (
	"encoding/json"
	"fmt"
	"github.com/tetragramaton/smh-go/internal/interface/modbus"
	mqttIface "github.com/tetragramaton/smh-go/internal/interface/mqtt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Meta struct {
	DeviceID string   `json:"device_id"`
	Model    string   `json:"model,omitempty"`
	Area     string   `json:"area,omitempty"`
	Caps     []string `json:"caps"`
}

type RegMap struct {
	// Input/holding address and scaling for each metric
	Frequency struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"` // raw / scale
		Holding bool    `json:"holding"`
	} `json:"frequency"`
	Voltage struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	} `json:"voltage"`
	Power struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	} `json:"power"`
	Energy struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	} `json:"energy"`
}

type SensorState struct {
	Ts        int64    `json:"ts"`
	Cap       string   `json:"cap"`
	Unit      string   `json:"unit,omitempty"`
	Value     *float64 `json:"value,omitempty"`
	PowerW    *float64 `json:"power_w,omitempty"`
	EnergyKwh *float64 `json:"energy_kwh,omitempty"`
}

func main() {
	handler, err := InitMainHandler()
	if err != nil {
		log.Fatal(err)
	}
	handler.Handle()
}

func (h *MainHandler) Handle() {
	cfg := loadEnv()
	defer h.MQQTClient.Disconnect(250)

	// announce meta
	meta := Meta{
		DeviceID: cfg.DeviceID,
		Model:    cfg.Model,
		Area:     cfg.Area,
		Caps:     []string{"sensor.frequency", "sensor.voltage", "energy.meter"},
	}
	if err := h.publishEvent(cfg, meta, "/meta"); err != nil {
		log.Printf("meta publish: %v", err)
	}

	defer func(ModbusClient modbus.Client) {
		err := ModbusClient.Close()
		if err != nil {
			log.Printf("modbus client close: %v", err)
		}
	}(h.ModbusClient)

	ticker := time.NewTicker(time.Duration(cfg.IntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()
			const statePath = "/state"
			const errorLog = "publish event error: %v"

			// frequency
			freqParam := modbus.RegisterParam{
				Addr:    cfg.MapCfg.Frequency.Addr,
				Scale:   cfg.MapCfg.Frequency.Scale,
				Holding: cfg.MapCfg.Frequency.Holding,
			}
			if v, err := h.readFloat(freqParam); err == nil {
				state := SensorState{
					Ts:    now,
					Cap:   "sensor.frequency",
					Value: round(v, 2),
					Unit:  "Hz",
				}
				if err := h.publishEvent(cfg, state, statePath); err != nil {
					log.Printf(errorLog, err)
				}
			} else {
				log.Printf("read frequency: %v", err)
			}

			// voltage
			voltParam := modbus.RegisterParam{
				Addr:    cfg.MapCfg.Voltage.Addr,
				Scale:   cfg.MapCfg.Voltage.Scale,
				Holding: cfg.MapCfg.Voltage.Holding,
			}
			if v, err := h.readFloat(voltParam); err == nil {
				state := SensorState{
					Ts:    now,
					Cap:   "sensor.voltage",
					Value: round(v, 1),
					Unit:  "V",
				}
				if err := h.publishEvent(cfg, state, statePath); err != nil {
					log.Printf(errorLog, err)
				}
			} else {
				log.Printf("read voltage: %v", err)
			}

			// power + energy (optional)
			var pwPtr, ekPtr *float64
			ok := false

			if cfg.MapCfg.Power.Addr != 0 {
				p := modbus.RegisterParam{
					Addr:    cfg.MapCfg.Power.Addr,
					Scale:   cfg.MapCfg.Power.Scale,
					Holding: cfg.MapCfg.Power.Holding,
				}
				if v, err := h.readFloat(p); err == nil {
					pwPtr = round(v, 1)
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
					ekPtr = round(v, 6)
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
				if err := h.publishEvent(cfg, state, statePath); err != nil {
					log.Printf(errorLog, err)
				}
			}
		}
	}
}

func (h *MainHandler) publishEvent(cfg envCfg, payload any, path string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	return h.MQQTClient.PublishEvent(mqttIface.Message{
		Topic:   "smh/" + cfg.DeviceID + path,
		Payload: data,
		QoS:     1,
		Retain:  false,
	})
}

func (h *MainHandler) readFloat(p modbus.RegisterParam) (float64, error) {
	return h.ModbusClient.ReadFloat(p)
}

type envCfg struct {
	MQTTURL  string
	DeviceID string
	Model    string
	Area     string

	Mode string // "rtu" or "tcp"
	// RTU
	Port      string
	Baud      int
	DataBits  int
	Parity    string // "N","E","O"
	StopBits  int
	SlaveID   int
	TimeoutMs int

	// TCP
	TCPAddr string // "192.168.1.10:502"

	IntervalSec int
	MapCfg      RegMap
}

func loadEnv() envCfg {
	get := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	atoi := func(s string, def int) int {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
		return def
	}

	cfg := envCfg{
		MQTTURL:     get("MQTT_URL", "tcp://mqtt:1883"),
		DeviceID:    get("DEVICE_ID", "cw100.inverter"),
		Model:       get("MODEL", "CW100"),
		Area:        get("AREA", "lab"),
		Mode:        strings.ToLower(get("MODBUS_MODE", "rtu")),
		Port:        get("MODBUS_PORT", "/dev/ttyUSB0"),
		Baud:        atoi(get("MODBUS_BAUD", "9600"), 9600),
		DataBits:    atoi(get("MODBUS_DATABITS", "8"), 8),
		Parity:      strings.ToUpper(get("MODBUS_PARITY", "N")),
		StopBits:    atoi(get("MODBUS_STOPBITS", "1"), 1),
		SlaveID:     atoi(get("MODBUS_SLAVE_ID", "1"), 1),
		TimeoutMs:   atoi(get("MODBUS_TIMEOUT_MS", "500"), 500),
		TCPAddr:     get("MODBUS_TCP_ADDR", "127.0.0.1:502"),
		IntervalSec: atoi(get("INTERVAL_SEC", "1"), 1),
	}

	// register map (default CW100-like)
	cfg.MapCfg.Frequency = struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	}(struct {
		Addr    uint16
		Scale   float64
		Holding bool
	}{Addr: 0x2000, Scale: 100.0, Holding: true})
	cfg.MapCfg.Voltage = struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	}(struct {
		Addr    uint16
		Scale   float64
		Holding bool
	}{Addr: 0x2001, Scale: 10.0, Holding: true})
	cfg.MapCfg.Power = struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	}(struct {
		Addr    uint16
		Scale   float64
		Holding bool
	}{Addr: 0x2003, Scale: 1.0, Holding: true})
	cfg.MapCfg.Energy = struct {
		Addr    uint16  `json:"addr"`
		Scale   float64 `json:"scale"`
		Holding bool    `json:"holding"`
	}(struct {
		Addr    uint16
		Scale   float64
		Holding bool
	}{Addr: 0x2004, Scale: 100.0, Holding: true})

	// optional JSON override via MODBUS_MAP_JSON
	if js := os.Getenv("MODBUS_MAP_JSON"); js != "" {
		var m RegMap
		if err := json.Unmarshal([]byte(js), &m); err == nil {
			cfg.MapCfg = m
		} else {
			log.Printf("WARN: bad MODBUS_MAP_JSON: %v", err)
		}
	}

	return cfg
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

func round(v float64, prec int) *float64 {
	p := 1.0
	for i := 0; i < prec; i++ {
		p *= 10
	}
	r := float64(int(v*p+0.5)) / p
	return &r
}
