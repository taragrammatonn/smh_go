package modbus

type RegisterParam struct {
	Addr    uint16  `json:"addr"`
	Scale   float64 `json:"scale"`
	Holding bool    `json:"holding"`
}

type Client interface {
	API
	ReadFloat(param RegisterParam) (float64, error)
	Close() error
}

type API interface {
	ReadHoldingRegisters(address, quantity uint16) (results []byte, err error)
	ReadInputRegisters(address, quantity uint16) (results []byte, err error)
}
