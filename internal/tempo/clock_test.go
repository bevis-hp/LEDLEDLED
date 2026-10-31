package tempo

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestBeatAdvances(t *testing.T) {
	c := New(120)
	if !near(c.Beat(1), 2) {
		t.Errorf("120bpm after 1s = %v beats, want 2", c.Beat(1))
	}
}

func TestSetBPMIsContinuous(t *testing.T) {
	c := New(120)
	before := c.Beat(3)
	c.SetBPM(3, 60)
	if !near(c.Beat(3), before) {
		t.Errorf("beat jumped from %v to %v", before, c.Beat(3))
	}
	if !near(c.Beat(4)-c.Beat(3), 1) {
		t.Errorf("60bpm should advance 1 beat per second")
	}
}

func TestTapSetsTempoAndPhase(t *testing.T) {
	c := New(60)
	for i := range 4 {
		c.Tap(10.1 + float64(i)*0.4) // 150 bpm
	}
	if !near(c.BPM(), 150) {
		t.Errorf("bpm = %v, want 150", c.BPM())
	}
	last := 10.1 + 3*0.4
	if b := c.Beat(last); !near(b, math.Round(b)) {
		t.Errorf("last tap not on a whole beat: %v", b)
	}
}

func TestTapResetsAfterPause(t *testing.T) {
	c := New(120)
	c.Tap(0)
	c.Tap(0.5) // 120
	c.Tap(10)  // long gap: new sequence, tempo unchanged
	if !near(c.BPM(), 120) {
		t.Errorf("bpm = %v after reset tap", c.BPM())
	}
}

func TestNudgeMovesTowardsBeat(t *testing.T) {
	c := New(60)
	c.Nudge(0.2, 0.5) // at beat 0.2, error -0.2 -> moves half way
	if !near(c.Beat(0.2), 0.1) {
		t.Errorf("nudged beat = %v, want 0.1", c.Beat(0.2))
	}
}
