package audio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os/exec"
	"strings"
	"time"
)

// SimDevice is a pseudo device that synthesises a 120 BPM beat, so the
// audio features can be developed and demoed without any hardware.
const SimDevice = "sim"

// blockFunc receives each block of samples (-1..1) and its capture time.
type blockFunc func(samples []float64, at time.Time)

// captureArecord streams mono 16-bit audio from an ALSA device using the
// `arecord` tool (alsa-utils). Using a subprocess avoids cgo, so the app
// stays a single static binary.
func captureArecord(ctx context.Context, device string, fn blockFunc) error {
	cmd := exec.CommandContext(ctx, "arecord", "-q",
		"-D", device,
		"-f", "S16_LE", "-c", "1", "-r", fmt.Sprint(sampleRate),
		"-t", "raw",
		"--buffer-time=50000", // 50 ms of buffering keeps latency low
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start arecord: %w (is alsa-utils installed?)", err)
	}

	r := bufio.NewReaderSize(out, blockSize*4)
	raw := make([]byte, blockSize*2)
	samples := make([]float64, blockSize)
	var readErr error
	for {
		if _, readErr = io.ReadFull(r, raw); readErr != nil {
			break
		}
		for i := range samples {
			samples[i] = float64(int16(binary.LittleEndian.Uint16(raw[i*2:]))) / 32768
		}
		fn(samples, time.Now())
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil // stopped on purpose
	}
	if msg := strings.TrimSpace(stderr.String()); msg != "" {
		return errors.New("arecord: " + lastLine(msg))
	}
	if waitErr != nil {
		return fmt.Errorf("arecord: %w", waitErr)
	}
	return readErr
}

// captureSim feeds a synthetic track in real time.
func captureSim(ctx context.Context, fn blockFunc) error {
	samples := make([]float64, blockSize)
	period := time.Second * blockSize / sampleRate
	tick := time.NewTicker(period)
	defer tick.Stop()
	var g simGen
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-tick.C:
			g.fill(samples)
			fn(samples, now)
		}
	}
}

// simGen synthesises a kick drum on every beat at 120 BPM, a hi-hat on the
// off-beats and a slowly wandering tone.
type simGen struct{ n int }

func (g *simGen) fill(samples []float64) {
	const bpm = 120.0
	for i := range samples {
		t := float64(g.n) / sampleRate
		beatPos := math.Mod(t*bpm/60, 1)
		bt := beatPos * 60 / bpm // seconds since the beat
		kick := math.Sin(2*math.Pi*(55+60*math.Exp(-bt*30))*bt) * math.Exp(-bt*12)
		off := math.Mod(beatPos+0.5, 1) * 60 / bpm
		hat := (rand.Float64()*2 - 1) * math.Exp(-off*40) * 0.25
		tone := math.Sin(2*math.Pi*(400+200*math.Sin(t*0.3))*t) * 0.08 * (1 + math.Sin(t*0.7))
		samples[i] = 0.7*kick + hat + tone
		g.n++
	}
}

// Devices lists ALSA capture devices (from `arecord -L`) plus the
// simulator. Errors just yield the simulator and "default".
func Devices() []string {
	devs := []string{SimDevice, "default"}
	out, err := exec.Command("arecord", "-L").Output()
	if err != nil {
		return devs
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue // descriptions are indented
		}
		if line == "null" || line == "default" {
			continue
		}
		devs = append(devs, line)
	}
	return devs
}

func lastLine(s string) string {
	lines := strings.Split(s, "\n")
	return lines[len(lines)-1]
}
