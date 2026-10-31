package audio

import (
	"math"
	"time"
)

// bpmEstimator guesses the tempo from beat onset times. Every interval
// between an onset and its next few onsets votes for a tempo, folded into
// one octave (80-160 BPM) so half/double-time detections agree. The most
// voted tempo wins once there is enough evidence.
type bpmEstimator struct {
	onsets []time.Time
	bpm    float64
}

const (
	bpmLo        = 80.0
	bpmHi        = 160.0
	minOnsets    = 8
	maxOnsets    = 32
	onsetWindow  = 12 * time.Second
	histRes      = 0.5 // BPM per histogram bin
	minConfident = 0.25
)

func (e *bpmEstimator) onset(t time.Time) {
	e.onsets = append(e.onsets, t)
	// Forget old onsets.
	cut := 0
	for cut < len(e.onsets) && (t.Sub(e.onsets[cut]) > onsetWindow || len(e.onsets)-cut > maxOnsets) {
		cut++
	}
	e.onsets = e.onsets[cut:]
	if len(e.onsets) < minOnsets {
		return
	}

	const bins = int((bpmHi - bpmLo) / histRes)
	var hist [bins]float64
	total := 0.0
	for i := range e.onsets {
		for j := i + 1; j < len(e.onsets) && j <= i+4; j++ {
			d := e.onsets[j].Sub(e.onsets[i]).Seconds()
			if d < 0.25 || d > 2 {
				continue
			}
			bpm := 60 / d
			for bpm < bpmLo {
				bpm *= 2
			}
			for bpm >= bpmHi {
				bpm /= 2
			}
			// Triangular kernel +-2 BPM tolerates timing jitter.
			for k := -4; k <= 4; k++ {
				b := int((bpm-bpmLo)/histRes) + k
				if b >= 0 && b < bins {
					w := 1 - math.Abs(float64(k))/5
					hist[b] += w
					total += w
				}
			}
		}
	}
	best, bestW := 0, 0.0
	for b, w := range hist {
		if w > bestW {
			best, bestW = b, w
		}
	}
	// A perfectly steady beat puts 1/5 of all weight in the peak bin (the
	// kernel weights sum to 5), so scale by 5 to get a 0..1 confidence.
	if total == 0 || bestW/total*5 < minConfident {
		return
	}
	est := bpmLo + (float64(best)+0.5)*histRes
	if e.bpm == 0 || math.Abs(est-e.bpm) > 6 {
		e.bpm = est
	} else {
		e.bpm += (est - e.bpm) * 0.2
	}
}
