package pixel

import (
	"math"
	"testing"
)

func TestPowerEstimate(t *testing.T) {
	m := PowerModel{Volts: 5, MilliampsPerChannel: 20, IdleMilliamps: 1}
	// 10 RGB pixels at full white: 10 * (60 + 1) mA.
	data := make([]byte, 30)
	for i := range data {
		data[i] = 255
	}
	if got := m.Milliamps(data, 10); math.Abs(got-610) > 1e-9 {
		t.Errorf("full white = %v mA, want 610", got)
	}
}

func TestPowerLimitScalesUnderMax(t *testing.T) {
	m := PowerModel{Volts: 12, MilliampsPerChannel: 20, IdleMilliamps: 2}
	data := make([]byte, 300) // 100 pixels
	for i := range data {
		data[i] = 255
	}
	before, after, scale := m.Limit(data, 100, 3000)
	if before != 6200 || after > 3000 || after < 2950 || scale >= 1 {
		t.Errorf("before=%v after=%v scale=%v", before, after, scale)
	}

	// Under the limit: untouched.
	data2 := []byte{10, 10, 10}
	_, _, s2 := m.Limit(data2, 1, 3000)
	if s2 != 1 || data2[0] != 10 {
		t.Errorf("under-limit data modified")
	}
}
