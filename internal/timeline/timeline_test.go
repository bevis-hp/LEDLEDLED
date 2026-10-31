package timeline

import (
	"math"
	"testing"
)

func TestNumberTrackInterpolates(t *testing.T) {
	tr, err := Compile(Number, []RawKey{
		{Beat: 4, Value: 1.0},
		{Beat: 0, Value: 0.0}, // out of order on purpose
		{Beat: 8, Value: 0.0, Ease: "step"},
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[float64]float64{-1: 0, 0: 0, 2: 0.5, 4: 1, 6: 0.5, 8: 0, 99: 0}
	for beat, want := range cases {
		if got, _, _ := tr.At(beat); math.Abs(got-want) > 1e-9 {
			t.Errorf("At(%v) = %v, want %v", beat, got, want)
		}
	}
}

func TestStepHoldsUntilNextKey(t *testing.T) {
	tr, _ := Compile(Number, []RawKey{{Beat: 0, Value: 1.0, Ease: "step"}, {Beat: 2, Value: 5.0}})
	if got, _, _ := tr.At(1.99); got != 1 {
		t.Errorf("step at 1.99 = %v, want 1", got)
	}
	if got, _, _ := tr.At(2); got != 5 {
		t.Errorf("step at 2 = %v, want 5", got)
	}
}

func TestColorTrack(t *testing.T) {
	tr, err := Compile(Color, []RawKey{{Beat: 0, Value: "#000000"}, {Beat: 1, Value: "#ff0000"}})
	if err != nil {
		t.Fatal(err)
	}
	_, c, _ := tr.At(0.5)
	if math.Abs(float64(c.R)-0.5) > 1e-6 || c.G != 0 {
		t.Errorf("mid colour = %+v", c)
	}
	if _, err := Compile(Color, []RawKey{{Beat: 0, Value: 3.0}}); err == nil {
		t.Error("expected type error")
	}
}
