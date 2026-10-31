package engine

import (
	"fmt"
	"math"
)

// transport is the timeline's play state. While playing, the playhead is
// locked to the beat clock, so it speeds up and slows down with the tempo.
type transport struct {
	playing   bool
	pos       float64 // playhead (beats) while stopped/paused
	startBeat float64 // clock beat at which the playhead was 0, while playing
}

// position returns the playhead for clock beat `beat`.
func (tr *transport) position(beat, length float64, oneShot bool) float64 {
	if !tr.playing {
		return math.Min(tr.pos, length)
	}
	p := beat - tr.startBeat
	if oneShot {
		if p >= length {
			tr.playing, tr.pos = false, length
			return length
		}
		return math.Max(p, 0)
	}
	p = math.Mod(p, length)
	if p < 0 {
		p += length
	}
	return p
}

// Transport controls the timeline: "play", "pause", "stop" or "seek".
func (e *Engine) Transport(action string, beat float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	tl := e.plan.show.Timeline
	clockBeat := e.clock.Beat(e.now())
	cur := e.tr.position(clockBeat, tl.LengthBeats, tl.OneShot)
	switch action {
	case "play":
		if e.tr.playing {
			return nil
		}
		if tl.OneShot && cur >= tl.LengthBeats {
			cur = 0
		}
		e.tr.playing, e.tr.startBeat = true, clockBeat-cur
	case "pause":
		e.tr.playing, e.tr.pos = false, cur
	case "stop":
		e.tr.playing, e.tr.pos = false, 0
	case "seek":
		beat = math.Max(0, math.Min(beat, tl.LengthBeats))
		e.tr.pos, e.tr.startBeat = beat, clockBeat-beat
	default:
		return fmt.Errorf("unknown transport action %q", action)
	}
	return nil
}
