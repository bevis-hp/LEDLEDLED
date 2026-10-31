// Package sysinfo reports host health for unattended installations: CPU
// temperature, uptime, load, free disk and the Raspberry Pi's
// under-voltage/throttling flags. Anything unavailable (e.g. when running
// on a laptop) is simply left empty.
package sysinfo

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Info is a snapshot of host health.
type Info struct {
	Hostname      string    `json:"hostname"`
	Version       string    `json:"version"`
	AppUptime     float64   `json:"appUptime"`               // seconds
	SystemUptime  float64   `json:"systemUptime,omitempty"`  // seconds
	CPUTempC      float64   `json:"cpuTempC,omitempty"`      // 0 = unknown
	Load1         float64   `json:"load1,omitempty"`         // 1-minute load average
	DiskFreeBytes uint64    `json:"diskFreeBytes,omitempty"` // on the state directory
	Throttle      *Throttle `json:"throttle,omitempty"`      // Raspberry Pi only
}

// Throttle decodes the Pi firmware's get_throttled bits.
type Throttle struct {
	UnderVoltageNow  bool `json:"underVoltageNow"`
	ThrottledNow     bool `json:"throttledNow"`
	UnderVoltageEver bool `json:"underVoltageEver"` // since boot
	ThrottledEver    bool `json:"throttledEver"`
}

// Paths are variables so tests can point them at fixtures.
var (
	thermalPath  = "/sys/class/thermal/thermal_zone0/temp"
	uptimePath   = "/proc/uptime"
	loadPath     = "/proc/loadavg"
	throttlePath = "/sys/devices/platform/soc/soc:firmware/get_throttled"
)

// Reader gathers Info. Create one at startup so AppUptime is meaningful.
type Reader struct {
	start   time.Time
	version string
	dataDir string
}

// NewReader returns a reader; dataDir is checked for free space.
func NewReader(version, dataDir string) *Reader {
	return &Reader{start: time.Now(), version: version, dataDir: dataDir}
}

// Read takes a snapshot. It only reads a few tiny files, so it is cheap
// enough to call every second.
func (r *Reader) Read() Info {
	host, _ := os.Hostname()
	info := Info{Hostname: host, Version: r.version, AppUptime: time.Since(r.start).Seconds()}
	if v, ok := firstFloat(thermalPath); ok {
		info.CPUTempC = v / 1000 // millidegrees
	}
	if v, ok := firstFloat(uptimePath); ok {
		info.SystemUptime = v
	}
	if v, ok := firstFloat(loadPath); ok {
		info.Load1 = v
	}
	info.DiskFreeBytes = diskFree(r.dataDir)
	if b, err := os.ReadFile(throttlePath); err == nil {
		if bits, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(string(b)), "0x"), 16, 32); err == nil {
			info.Throttle = decodeThrottle(bits)
		}
	}
	return info
}

func decodeThrottle(bits uint64) *Throttle {
	return &Throttle{
		UnderVoltageNow:  bits&(1<<0) != 0,
		ThrottledNow:     bits&(1<<2) != 0,
		UnderVoltageEver: bits&(1<<16) != 0,
		ThrottledEver:    bits&(1<<18) != 0,
	}
}

// firstFloat parses the first whitespace-separated field of a file.
func firstFloat(path string) (float64, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0, false
	}
	v, err := strconv.ParseFloat(f[0], 64)
	return v, err == nil
}
