# SMH Go — Modbus RTU/TCP adapter (Go-only)

This repo contains a **Go-only** smart-home starter with a real Modbus adapter.

## Components
- `cmd/smh-core` — listens for `smh/<device>/meta` and publishes **Home Assistant Discovery** configs.
- `cmd/adapter-modbus` — reads real Modbus **RTU or TCP** registers and publishes states:
  - `sensor.frequency` (Hz), `sensor.voltage` (V)
  - `energy.meter` (`power_w`, `energy_kwh` — optional if mapped)

## Build
```bash
go mod tidy
go build ./cmd/smh-core
go build ./cmd/adapter-modbus
```

## Run with Docker (Mosquitto + core + adapter)
```bash
docker compose -f deploy/docker-compose.yml up --build
# Watch topics:
docker exec -it smh-mosquitto sh -c 'mosquitto_sub -h localhost -t "#" -v'
```

## Adapter configuration (env)
- General:
  - `MQTT_URL` (default `tcp://mqtt:1883`)
  - `DEVICE_ID` (default `cw100.inverter`), `MODEL`, `AREA`
  - `INTERVAL_SEC` (default `1`)
- Mode:
  - `MODBUS_MODE=rtu|tcp` (default `rtu`)
- RTU:
  - `MODBUS_PORT` (default `/dev/ttyUSB0`), `MODBUS_BAUD` (9600), `MODBUS_DATABITS` (8),
    `MODBUS_PARITY` (`N`), `MODBUS_STOPBITS` (1), `MODBUS_SLAVE_ID` (1),
    `MODBUS_TIMEOUT_MS` (500)
- TCP:
  - `MODBUS_TCP_ADDR` (default `127.0.0.1:502`)
- Register map (optional override):
  - `MODBUS_MAP_JSON` — JSON object, e.g.:
```
{"frequency":{"addr":8192,"scale":100,"holding":true},
 "voltage":{"addr":8193,"scale":10,"holding":true},
 "power":{"addr":8195,"scale":1,"holding":true},
 "energy":{"addr":8196,"scale":100,"holding":true}}
```

## Notes
- Default register map targets a **CW100-like** inverter (freq at 0x2000 scaled by 100, voltage at 0x2001 /10, etc.). Adjust for your device.
- For RS485 USB dongles that auto-handle DE/RE, you don't need GPIO control.
- If you use ESP32 as a Modbus TCP bridge, set `MODBUS_MODE=tcp` + `MODBUS_TCP_ADDR=<bridge IP:502>`.
