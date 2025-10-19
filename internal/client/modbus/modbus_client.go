package modbus

import (
	"context"
	"errors"
	"fmt"
	"github.com/goburrow/modbus"
	modbusIface "github.com/tetragramaton/smh-go/internal/interface/modbus"
	"os"
	"strconv"
	"strings"
	"time"
)

type EnvCfg struct {
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

type handler struct {
	modbusIface.API
	context.Context
	closeFn func() error
}

func NewHandler() (modbusIface.Client, error) {
	cfg, err := LoadEnvCfg()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	if cfg.Mode == "tcp" {
		th := modbus.NewTCPClientHandler(cfg.TCPAddr)
		th.Timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
		if err := th.Connect(); err != nil {
			return nil, err
		}
		return &handler{
			API:     modbus.NewClient(th),
			Context: ctx,
			closeFn: th.Close,
		}, nil
	}

	rh := modbus.NewRTUClientHandler(cfg.Port)
	rh.BaudRate = cfg.Baud
	rh.DataBits = cfg.DataBits
	rh.Parity = cfg.Parity
	rh.StopBits = cfg.StopBits
	rh.SlaveId = byte(cfg.SlaveID)
	rh.Timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	if err := rh.Connect(); err != nil {
		return nil, err
	}

	return &handler{
		API:     modbus.NewClient(rh),
		Context: ctx,
		closeFn: rh.Close,
	}, nil
}

func (h *handler) ReadFloat(param modbusIface.RegisterParam) (float64, error) {
	var res []byte
	var err error
	if param.Holding {
		res, err = h.API.ReadHoldingRegisters(param.Addr, 1)
	} else {
		res, err = h.API.ReadInputRegisters(param.Addr, 1)
	}
	if err != nil {
		return 0, err
	}
	// 16-bit register
	if len(res) < 2 {
		return 0, fmt.Errorf("short response")
	}
	raw := uint16(res[0])<<8 | uint16(res[1])
	return float64(int16(raw)) / param.Scale, nil
}

func (h *handler) Close() error { return nil }

func LoadEnvCfg() (EnvCfg, error) {
	var c EnvCfg

	c.Mode = strings.ToLower(os.Getenv("MODBUS_MODE"))
	if c.Mode != "rtu" && c.Mode != "tcp" {
		return c, errors.New("MODBUS_MODE must be 'rtu' or 'tcp'")
	}

	if c.Mode == "tcp" {
		c.TCPAddr = os.Getenv("MODBUS_TCP_ADDR")
		if c.TCPAddr == "" {
			return c, errors.New("missing MODBUS_TCP_ADDR for tcp mode")
		}
	} else {
		c.Port = os.Getenv("MODBUS_PORT")
		if c.Port == "" {
			return c, errors.New("missing MODBUS_PORT for rtu mode")
		}
		c.Baud, _ = strconv.Atoi(getEnvDefault("MODBUS_BAUD", "9600"))
		c.DataBits, _ = strconv.Atoi(getEnvDefault("MODBUS_DATABITS", "8"))
		c.Parity = getEnvDefault("MODBUS_PARITY", "N")
		c.StopBits, _ = strconv.Atoi(getEnvDefault("MODBUS_STOPBITS", "1"))
		c.SlaveID, _ = strconv.Atoi(getEnvDefault("MODBUS_SLAVE_ID", "1"))
		c.TimeoutMs, _ = strconv.Atoi(getEnvDefault("MODBUS_TIMEOUT_MS", "500"))
	}

	c.IntervalSec, _ = strconv.Atoi(getEnvDefault("INTERVAL_SEC", "1"))
	c.DeviceID = getEnvDefault("DEVICE_ID", "unknown")
	c.Model = getEnvDefault("MODEL", "unknown")
	c.Area = getEnvDefault("AREA", "lab")

	return c, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
