package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFromFixtures(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	thermalPath = write("temp", "52341\n")
	uptimePath = write("uptime", "12345.67 40000.00\n")
	loadPath = write("loadavg", "0.42 0.30 0.25 1/123 4567\n")
	throttlePath = write("throttled", "0x50005\n") // under-voltage + throttled, now and since boot

	info := NewReader("v1", dir).Read()
	if info.CPUTempC < 52.3 || info.CPUTempC > 52.4 {
		t.Errorf("temp = %v", info.CPUTempC)
	}
	if info.SystemUptime != 12345.67 || info.Load1 != 0.42 || info.Version != "v1" {
		t.Errorf("info = %+v", info)
	}
	th := info.Throttle
	if th == nil || !th.UnderVoltageNow || !th.ThrottledNow || !th.UnderVoltageEver || !th.ThrottledEver {
		t.Errorf("throttle = %+v", th)
	}
	if info.DiskFreeBytes == 0 {
		t.Error("disk free not reported")
	}

	// Missing files just leave fields empty.
	thermalPath, throttlePath = filepath.Join(dir, "nope"), filepath.Join(dir, "nope")
	info = NewReader("v1", dir).Read()
	if info.CPUTempC != 0 || info.Throttle != nil {
		t.Errorf("missing files: %+v", info)
	}
}
