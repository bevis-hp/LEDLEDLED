// Package audio captures sound from an ALSA input and turns it into a few
// slowly-changing control signals for the lights: overall level, bass /
// mid / treble, a 16-band spectrum, a beat envelope and an estimated BPM.
package audio

// NumBands is the number of spectrum bands (log-spaced 40 Hz - 16 kHz).
const NumBands = 16

// Features are the analysed control signals, all normalised to 0..1
// (except BPM).
type Features struct {
	Active    bool              `json:"active"` // input running and above the noise gate recently
	Level     float64           `json:"level"`
	Bass      float64           `json:"bass"`
	Mid       float64           `json:"mid"`
	Treble    float64           `json:"treble"`
	Beat      float64           `json:"beat"` // 1 on a detected beat, decaying to 0
	Bands     [NumBands]float64 `json:"bands"`
	BPM       float64           `json:"bpm"`       // detected tempo, 0 = unknown
	BeatCount uint64            `json:"beatCount"` // increments on every detected beat
	// BeatAge is how long ago (seconds) the latest beat was detected, so the
	// engine can place it precisely on its own clock.
	BeatAge float64 `json:"-"`
}

// Sources lists the feature names usable by modulators and effects.
var Sources = []string{"level", "bass", "mid", "treble", "beat"}

// Get returns a feature by name (0 if unknown).
func (f *Features) Get(name string) float64 {
	switch name {
	case "level":
		return f.Level
	case "bass":
		return f.Bass
	case "mid":
		return f.Mid
	case "treble":
		return f.Treble
	case "beat":
		return f.Beat
	}
	return 0
}
