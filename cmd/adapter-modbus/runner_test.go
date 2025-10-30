package main

//import (
//	"encoding/json"
//	"os"
//	"path/filepath"
//	"testing"
//)
//
//// mock handler returning fixed values derived from a JSON map
//type mockHandler struct {
//	values map[string]float64
//}
//
//func (m *mockHandler) readFloat(addr uint16, scale float64, holding bool) (float64, error) {
//	// key as "addr:<n>"
//	key := "addr:" + itoa(int(addr))
//	if v, ok := m.values[key]; ok {
//		return v / scale, nil
//	}
//	// not found -> simulate read error by returning zero and no error;
//	// PublishOnce treats missing metrics gracefully when addr==0
//	return 0, nil
//}
//func (m *mockHandler) Close() error { return nil }
//
//// mock mqtt publisher collecting published JSON payloads
//type mockMQTT struct{ msgs [][]byte }
//
//func (m *mockMQTT) Publish(topic string, payload []byte, qos byte, retain bool) error {
//	m.msgs = append(m.msgs, payload)
//	return nil
//}
//
//func TestPublishOnce_WithMockHandler(t *testing.T) {
//	// load golden and script values
//	td := filepath.Join("testdata", "fixtures", "modbus")
//	_ = os.MkdirAll(td, 0o755)
//
//	// script values emulate raw registers BEFORE scaling (frequency 5000->50.00Hz, voltage 2300->230.0V, power 800W, energy 12345/100=123.45kWh)
//	script := map[string]float64{
//		"addr:8192": 5000,  // 0x2000
//		"addr:8193": 2300,  // 0x2001
//		"addr:8195": 800,   // 0x2003
//		"addr:8196": 12345, // 0x2004
//	}
//	h := &mockHandler{values: script}
//	mq := &mockMQTT{}
//
//	cfg := loadEnv()
//	// enforce deterministic device id and map
//	cfg.DeviceID = "cw100.inverter"
//	cfg.MapCfg.Frequency.Addr = 0x2000
//	cfg.MapCfg.Frequency.Scale = 100
//	cfg.MapCfg.Frequency.Holding = true
//	cfg.MapCfg.Voltage.Addr = 0x2001
//	cfg.MapCfg.Voltage.Scale = 10
//	cfg.MapCfg.Voltage.Holding = true
//	cfg.MapCfg.Power.Addr = 0x2003
//	cfg.MapCfg.Power.Scale = 1
//	cfg.MapCfg.Power.Holding = true
//	cfg.MapCfg.Energy.Addr = 0x2004
//	cfg.MapCfg.Energy.Scale = 100
//	cfg.MapCfg.Energy.Holding = true
//
//	now := int64(1700000000)
//	PublishOnce(h, cfg, now)
//
//	if len(mq.msgs) != 3 {
//		t.Fatalf("expected 3 state messages, got %d", len(mq.msgs))
//	}
//
//	// compare each payload against golden allowing numeric approx
//	golden := []string{
//		filepath.Join(td, "state_frequency.json"),
//		filepath.Join(td, "state_voltage.json"),
//		filepath.Join(td, "state_energy.json"),
//	}
//	gotObjs := make([]map[string]interface{}, 0, 3)
//	for _, b := range mq.msgs {
//		var m map[string]interface{}
//		if err := json.Unmarshal(b, &m); err != nil {
//			t.Fatalf("bad json: %v", err)
//		}
//		gotObjs = append(gotObjs, m)
//	}
//	// load goldens and check presence
//	wantObjs := make([]map[string]interface{}, 0, 3)
//	for _, g := range golden {
//		b, err := os.ReadFile(g)
//		if err != nil {
//			t.Fatalf("missing golden %s: %v", g, err)
//		}
//		var m map[string]interface{}
//		if err := json.Unmarshal(b, &m); err != nil {
//			t.Fatalf("bad golden json: %v", err)
//		}
//		wantObjs = append(wantObjs, m)
//	}
//	for _, want := range wantObjs {
//		found := false
//		for _, got := range gotObjs {
//			if approxEqualJSON(got, want, 1e-3) {
//				found = true
//				break
//			}
//		}
//		if !found {
//			t.Fatalf("expected to find payload like %v in published messages", stringMust(json.Marshal(want)))
//		}
//	}
//}
//
//// helpers
//func approxEqualJSON(got, want map[string]interface{}, eps float64) bool {
//	for k, v := range want {
//		gv, ok := got[k]
//		if !ok {
//			return false
//		}
//		switch w := v.(type) {
//		case float64:
//			gf, ok := gv.(float64)
//			if !ok {
//				return false
//			}
//			if abs(gf-w) > eps {
//				return false
//			}
//		default:
//			if stringMust(json.Marshal(gv)) != stringMust(json.Marshal(w)) {
//				return false
//			}
//		}
//	}
//	return true
//}
//
//func abs(x float64) float64 {
//	if x < 0 {
//		return -x
//	}
//	return x
//}
//func stringMust(b []byte, _ error) string { return string(b) }
//func itoa(i int) string                   { return fmtInt(i) }
//
//// tiny fmt-free int->string to avoid extra imports in this snippet
//func fmtInt(n int) string {
//	if n == 0 {
//		return "0"
//	}
//	sign := ""
//	if n < 0 {
//		sign = "-"
//		n = -n
//	}
//	var d []byte
//	for n > 0 {
//		d = append([]byte{byte('0' + n%10)}, d...)
//		n /= 10
//	}
//	return sign + string(d)
//}
