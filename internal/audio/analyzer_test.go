package audio

import (
	"math"
	"math/cmplx"
	"testing"
	"time"
)

func TestFFTFindsSinePeak(t *testing.T) {
	x := make([]complex128, 64)
	for i := range x {
		x[i] = complex(math.Sin(2*math.Pi*5*float64(i)/64), 0)
	}
	fft(x)
	best := 0
	for k := 1; k < 32; k++ {
		if cmplx.Abs(x[k]) > cmplx.Abs(x[best]) {
			best = k
		}
	}
	if best != 5 {
		t.Errorf("peak bin = %d, want 5", best)
	}
}

// feed runs the analyzer over generated audio with synthetic timestamps.
func feed(a *analyzer, seconds float64, gen func([]float64)) Features {
	samples := make([]float64, blockSize)
	start := time.Unix(0, 0)
	var f Features
	blocks := int(seconds * sampleRate / blockSize)
	for b := range blocks {
		gen(samples)
		at := start.Add(time.Duration(float64(b) * blockSize / sampleRate * float64(time.Second)))
		f = a.process(samples, at)
	}
	return f
}

func TestToneLandsInMids(t *testing.T) {
	n := 0
	f := feed(newAnalyzer(), 2, func(s []float64) {
		for i := range s {
			s[i] = 0.5 * math.Sin(2*math.Pi*1000*float64(n)/sampleRate)
			n++
		}
	})
	if !f.Active || f.Mid < 0.8 || f.Level < 0.8 {
		t.Errorf("1 kHz tone: %+v", f)
	}
	if f.Bands[0] > 0.2 || f.Bands[NumBands-1] > 0.2 {
		t.Errorf("energy leaked to extreme bands: %v", f.Bands)
	}
}

func TestSilenceIsGated(t *testing.T) {
	f := feed(newAnalyzer(), 3, func(s []float64) {
		for i := range s {
			s[i] = 0.0005 // well under the gate
		}
	})
	if f.Active || f.Level > 0.01 || f.Beat > 0.01 {
		t.Errorf("silence: %+v", f)
	}
}

func TestSimTrackBeatsAndBPM(t *testing.T) {
	var g simGen
	f := feed(newAnalyzer(), 20, g.fill)
	// 20 s at 120 BPM = 40 beats; allow for warm-up and the odd miss.
	if f.BeatCount < 30 || f.BeatCount > 44 {
		t.Errorf("beats = %d, want ~40", f.BeatCount)
	}
	if math.Abs(f.BPM-120) > 3 {
		t.Errorf("bpm = %.1f, want ~120", f.BPM)
	}
}
