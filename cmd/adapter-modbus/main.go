package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	//"github.com/eclipse/paho.mqtt.golang"
	"github.com/goburrow/modbus"
	mqtt "github.com/tetragramaton/smh-go/internal/mqtt"
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

func main() {
	cfg := loadEnv()
	client, err := mqtt.New(mqtt.Config{BrokerURL: cfg.MQTTURL, ClientID: "adapter-modbus-" + time.Now().Format("150405")})
	if err != nil {
		log.Fatalf("mqtt: %v", err)
	}
	defer client.Close()

	// announce meta
	meta := Meta{
		DeviceID: cfg.DeviceID,
		Model:    cfg.Model,
		Area:     cfg.Area,
		Caps:     []string{"sensor.frequency", "sensor.voltage", "energy.meter"},
	}
	if err := client.Publish("smh/"+cfg.DeviceID+"/meta", must(json.Marshal(meta)), 1, false); err != nil {
		log.Printf("meta publish: %v", err)
	}

	handler, err := newHandler(cfg)
	if err != nil {
		log.Fatalf("modbus handler: %v", err)
	}
	defer handler.Close()

	ticker := time.NewTicker(time.Duration(cfg.IntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()

			if v, err := handler.readFloat(cfg.MapCfg.Frequency.Addr, cfg.MapCfg.Frequency.Scale, cfg.MapCfg.Frequency.Holding); err == nil {
				state := map[string]interface{}{"ts": now, "cap": "sensor.frequency", "value": round(v, 2), "unit": "Hz"}
				client.Publish("smh/"+cfg.DeviceID+"/state", must(json.Marshal(state)), 1, false)
			} else {
				log.Printf("read frequency: %v", err)
			}
			if v, err := handler.readFloat(cfg.MapCfg.Voltage.Addr, cfg.MapCfg.Voltage.Scale, cfg.MapCfg.Voltage.Holding); err == nil {
				state := map[string]interface{}{"ts": now, "cap": "sensor.voltage", "value": round(v, 1), "unit": "V"}
				client.Publish("smh/"+cfg.DeviceID+"/state", must(json.Marshal(state)), 1, false)
			} else {
				log.Printf("read voltage: %v", err)
			}
			// Power + Energy (optional)
			var powerW, energyKwh *float64
			if cfg.MapCfg.Power.Addr != 0 {
				if v, err := handler.readFloat(cfg.MapCfg.Power.Addr, cfg.MapCfg.Power.Scale, cfg.MapCfg.Power.Holding); err == nil {
					powerW = &v
				}
			}
			if cfg.MapCfg.Energy.Addr != 0 {
				if v, err := handler.readFloat(cfg.MapCfg.Energy.Addr, cfg.MapCfg.Energy.Scale, cfg.MapCfg.Energy.Holding); err == nil {
					energyKwh = &v
				}
			}
			if powerW != nil || energyKwh != nil {
				payload := map[string]interface{}{"ts": now, "cap": "energy.meter"}
				if powerW != nil {
					payload["power_w"] = round(*powerW, 1)
				}
				if energyKwh != nil {
					payload["energy_kwh"] = round(*energyKwh, 6)
				}
				client.Publish("smh/"+cfg.DeviceID+"/state", must(json.Marshal(payload)), 1, false)
			}
		}
	}
}

type handler interface {
	readFloat(addr uint16, scale float64, holding bool) (float64, error)
	Close() error
}

type rtuHandler struct{ client modbus.Client }
type tcpHandler struct{ client modbus.Client }

func newHandler(cfg envCfg) (handler, error) {
	if strings.ToLower(cfg.Mode) == "tcp" {
		h := &tcpHandler{client: modbus.TCPClient(fmt.Sprintf("%s", cfg.TCPAddr))}
		// warmup read with timeout by wrapping in handler
		return h, nil
	}
	// RTU
	// goburrow uses environment via serial.Config pushed into RTUClientHandler;
	rtu := modbus.NewRTUClientHandler(cfg.Port)
	rtu.BaudRate = cfg.Baud
	rtu.DataBits = cfg.DataBits
	rtu.Parity = cfg.Parity
	rtu.StopBits = cfg.StopBits
	rtu.SlaveId = byte(cfg.SlaveID)
	rtu.Timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	if err := rtu.Connect(); err != nil {
		return nil, err
	}
	return &rtuHandler{client: modbus.NewClient(rtu)}, nil
}

func (h *rtuHandler) readFloat(addr uint16, scale float64, holding bool) (float64, error) {
	var res []byte
	var err error
	if holding {
		res, err = h.client.ReadHoldingRegisters(addr, 1)
	} else {
		res, err = h.client.ReadInputRegisters(addr, 1)
	}
	if err != nil {
		return 0, err
	}
	// 16-bit register
	if len(res) < 2 {
		return 0, fmt.Errorf("short response")
	}
	raw := uint16(res[0])<<8 | uint16(res[1])
	return float64(int16(raw)) / scale, nil
}

func (h *rtuHandler) Close() error { return nil }

func (h *tcpHandler) readFloat(addr uint16, scale float64, holding bool) (float64, error) {
	var res []byte
	var err error
	if holding {
		res, err = h.client.ReadHoldingRegisters(addr, 1)
	} else {
		res, err = h.client.ReadInputRegisters(addr, 1)
	}
	if err != nil {
		return 0, err
	}
	if len(res) < 2 {
		return 0, fmt.Errorf("short response")
	}
	raw := uint16(res[0])<<8 | uint16(res[1])
	return float64(int16(raw)) / scale, nil
}
func (h *tcpHandler) Close() error { return nil }

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

func round(v float64, prec int) float64 {
	p := 1.0
	for i := 0; i < prec; i++ {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
}
