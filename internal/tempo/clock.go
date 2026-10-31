// Package tempo implements the musical beat clock: a BPM plus a phase
// anchor, driven by tap tempo or by detected audio beats.
//
// Time is passed in as seconds (the engine's clock) so the clock is easy to
// test and never reads wall time itself. It is not safe for concurrent use;
// the engine guards it with its own mutex.
package tempo

import "math"

const (
	MinBPM = 20
	MaxBPM = 300

	tapReset = 2.0 // seconds without a tap before a new tap sequence starts
	maxTaps  = 8
)

// Clock maps seconds to beats.
type Clock struct {
	bpm        float64
	anchorT    float64 // time at which the beat count equals anchorBeat
	anchorBeat float64
	taps       []float64
}

// New returns a clock at bpm starting at beat 0 at time 0.
func New(bpm float64) *Clock {
	return &Clock{bpm: clampBPM(bpm)}
}

// BPM returns the current tempo.
func (c *Clock) BPM() float64 { return c.bpm }

// Beat returns the (fractional) number of beats elapsed at time t.
func (c *Clock) Beat(t float64) float64 {
	return c.anchorBeat + (t-c.anchorT)*c.bpm/60
}

// SetBPM changes the tempo without a jump in the beat position at time t.
func (c *Clock) SetBPM(t, bpm float64) {
	c.anchorBeat = c.Beat(t)
	c.anchorT = t
	c.bpm = clampBPM(bpm)
}

// Tap registers a tap at time t. One tap re-syncs the downbeat; two or more
// taps within tapReset seconds of each other also set the tempo from their
// average interval.
func (c *Clock) Tap(t float64) {
	if n := len(c.taps); n > 0 && t-c.taps[n-1] > tapReset {
		c.taps = c.taps[:0]
	}
	c.taps = append(c.taps, t)
	if len(c.taps) > maxTaps {
		c.taps = c.taps[1:]
	}
	if n := len(c.taps); n >= 2 {
		interval := (c.taps[n-1] - c.taps[0]) / float64(n-1)
		if interval > 0 {
			c.SetBPM(t, 60/interval)
		}
	}
	// Make this tap land exactly on a whole beat.
	c.anchorBeat = math.Round(c.Beat(t))
	c.anchorT = t
}

// Nudge pulls the phase towards a beat that was detected at time t, by
// strength (0..1) of the error. Used to lock onto audio beats gently.
func (c *Clock) Nudge(t, strength float64) {
	b := c.Beat(t)
	err := math.Round(b) - b // -0.5..0.5 beats
	c.anchorBeat = b + err*strength
	c.anchorT = t
}

func clampBPM(b float64) float64 {
	if b <= 0 || math.IsNaN(b) {
		return 120
	}
	return math.Max(MinBPM, math.Min(MaxBPM, b))
}
