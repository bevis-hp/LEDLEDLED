package audio

import (
	"math"
	"time"
)

const (
	sampleRate = 44100
	blockSize  = 1024 // samples per analysis block (~23 ms)

	gateLevel  = 0.004 // RMS below this (~ -48 dBFS) counts as silence
	peakFloor  = 0.01  // AGC never normalises against less than this
	peakDecay  = 0.995 // AGC peak falloff per block (~3 s half-life)
	attack     = 0.6   // smoothing towards a rising value
	release    = 0.2   // smoothing towards a falling value
	beatDecay  = 0.85  // beat envelope falloff per block
	beatRatio  = 1.5   // bass energy vs recent average to count as a beat
	minBeatGap = 0.25  // seconds; caps detection at 240 BPM
)

// bandEdges are the log-spaced band boundaries in Hz.
var bandEdges = func() [NumBands + 1]float64 {
	var e [NumBands + 1]float64
	lo, hi := 40.0, 16000.0
	for i := range e {
		e[i] = lo * math.Pow(hi/lo, float64(i)/NumBands)
	}
	return e
}()

// analyzer turns blocks of samples into Features. It is owned by a single
// goroutine.
type analyzer struct {
	gain float64

	window [blockSize]float64
	buf    [blockSize]complex128
	mag    [blockSize / 2]float64

	// Automatic gain: running peaks per signal.
	pkLevel, pkBass, pkMid, pkTreble, pkBands float64

	bassHist   [43]float64 // ~1 s of bass energy for beat detection
	histPos    int
	lastBeat   time.Time
	quietSince time.Time

	tempo bpmEstimator
	out   Features
}

func newAnalyzer() *analyzer {
	a := &analyzer{gain: 1}
	for i := range a.window {
		a.window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(blockSize-1)) // Hann
	}
	return a
}

// process analyses one block of samples (-1..1) captured at time now.
func (a *analyzer) process(samples []float64, now time.Time) Features {
	// Time-domain RMS level.
	var sum float64
	for i, s := range samples {
		sum += s * s
		a.buf[i] = complex(s*a.window[i], 0)
	}
	rms := math.Sqrt(sum / float64(len(samples)))

	fft(a.buf[:])
	// Scale so a full-scale sine reads about 1 (Hann window halves amplitude).
	for k := range a.mag {
		a.mag[k] = 4 * cabs(a.buf[k]) / blockSize
	}

	bass := a.rangeRMS(30, 150)
	mid := a.rangeRMS(150, 2000)
	treble := a.rangeRMS(2000, 12000)
	var bands [NumBands]float64
	maxBand := 0.0
	for b := range bands {
		bands[b] = a.rangeRMS(bandEdges[b], bandEdges[b+1])
		maxBand = max(maxBand, bands[b])
	}

	quiet := rms < gateLevel
	if quiet {
		if a.quietSince.IsZero() {
			a.quietSince = now
		}
		rms, bass, mid, treble, maxBand = 0, 0, 0, 0, 0
		bands = [NumBands]float64{}
	} else {
		a.quietSince = time.Time{}
	}

	o := &a.out
	o.Active = !quiet || now.Sub(a.quietSince) < 2*time.Second
	o.Level = smooth(o.Level, a.agc(&a.pkLevel, rms))
	o.Bass = smooth(o.Bass, a.agc(&a.pkBass, bass))
	o.Mid = smooth(o.Mid, a.agc(&a.pkMid, mid))
	o.Treble = smooth(o.Treble, a.agc(&a.pkTreble, treble))
	// One shared peak keeps the spectrum's shape.
	a.pkBands = max(maxBand, a.pkBands*peakDecay, peakFloor)
	for b := range bands {
		o.Bands[b] = smooth(o.Bands[b], math.Min(1, a.gain*bands[b]/a.pkBands))
	}

	// Beat detection: bass energy well above its recent average.
	energy := bass * bass
	avg := 0.0
	for _, e := range a.bassHist {
		avg += e
	}
	avg /= float64(len(a.bassHist))
	a.bassHist[a.histPos] = energy
	a.histPos = (a.histPos + 1) % len(a.bassHist)

	o.Beat *= beatDecay
	if !quiet && energy > beatRatio*avg && bass > peakFloor && now.Sub(a.lastBeat).Seconds() > minBeatGap {
		a.lastBeat = now
		o.Beat = 1
		o.BeatCount++
		a.tempo.onset(now)
	}
	o.BPM = a.tempo.bpm
	return *o
}

// rangeRMS returns the RMS magnitude of FFT bins between lo and hi Hz.
func (a *analyzer) rangeRMS(lo, hi float64) float64 {
	binHz := float64(sampleRate) / blockSize
	k0 := max(1, int(lo/binHz))
	k1 := min(len(a.mag)-1, int(hi/binHz))
	if k1 < k0 {
		k1 = k0
	}
	var s float64
	for k := k0; k <= k1; k++ {
		s += a.mag[k] * a.mag[k]
	}
	return math.Sqrt(s / float64(k1-k0+1))
}

// agc normalises v against a slowly decaying peak and applies the gain.
func (a *analyzer) agc(peak *float64, v float64) float64 {
	*peak = max(v, *peak*peakDecay, peakFloor)
	return math.Min(1, a.gain*v / *peak)
}

func smooth(prev, v float64) float64 {
	if v > prev {
		return prev + (v-prev)*attack
	}
	return prev + (v-prev)*release
}

func cabs(c complex128) float64 { return math.Hypot(real(c), imag(c)) }
