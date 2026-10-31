// Package timeline evaluates keyframe tracks. A track is a sorted list of
// keys (beat, value, easing); between two keys the value is interpolated
// using the easing of the earlier key. Before the first key the first
// value holds, after the last key the last value holds.
package timeline

import (
	"fmt"
	"math"
	"sort"

	"ledledled/internal/pixel"
)

// Kind is the value type a track animates.
type Kind int

const (
	Number Kind = iota
	Color
)

// Eases lists the supported easing names, for the UI.
var Eases = []string{"linear", "smooth", "in", "out", "step"}

// EaseFunc maps 0..1 progress to 0..1 eased progress.
type EaseFunc func(float64) float64

// Ease returns the named easing function (linear if unknown).
func Ease(name string) EaseFunc {
	switch name {
	case "step":
		return func(float64) float64 { return 0 } // hold until the next key
	case "smooth":
		return func(t float64) float64 { return t * t * (3 - 2*t) }
	case "in":
		return func(t float64) float64 { return t * t }
	case "out":
		return func(t float64) float64 { return 1 - (1-t)*(1-t) }
	default:
		return func(t float64) float64 { return t }
	}
}

// Key is one compiled keyframe.
type Key struct {
	Beat  float64
	Num   float64
	Color pixel.Color
	Ease  EaseFunc
}

// Track is a compiled, sorted keyframe track.
type Track struct {
	Kind Kind
	Keys []Key
}

// RawKey is a keyframe as stored in the show (value is a JSON number or a
// "#rrggbb" string).
type RawKey struct {
	Beat  float64
	Value any
	Ease  string
}

// Compile converts raw keys into a Track. Keys whose value doesn't match
// kind return an error.
func Compile(kind Kind, raw []RawKey) (Track, error) {
	t := Track{Kind: kind, Keys: make([]Key, 0, len(raw))}
	for i, r := range raw {
		k := Key{Beat: r.Beat, Ease: Ease(r.Ease)}
		switch kind {
		case Number:
			v, ok := r.Value.(float64)
			if !ok {
				return t, fmt.Errorf("key %d: want a number, got %v", i+1, r.Value)
			}
			k.Num = v
		case Color:
			s, _ := r.Value.(string)
			c, err := pixel.ParseHex(s)
			if err != nil {
				return t, fmt.Errorf("key %d: %w", i+1, err)
			}
			k.Color = c
		}
		t.Keys = append(t.Keys, k)
	}
	sort.SliceStable(t.Keys, func(i, j int) bool { return t.Keys[i].Beat < t.Keys[j].Beat })
	return t, nil
}

// At evaluates the track at beat. ok is false for an empty track.
func (t *Track) At(beat float64) (num float64, col pixel.Color, ok bool) {
	n := len(t.Keys)
	if n == 0 {
		return 0, pixel.Black, false
	}
	// First key strictly after beat.
	i := sort.Search(n, func(i int) bool { return t.Keys[i].Beat > beat })
	switch {
	case i == 0:
		k := t.Keys[0]
		return k.Num, k.Color, true
	case i == n:
		k := t.Keys[n-1]
		return k.Num, k.Color, true
	}
	a, b := t.Keys[i-1], t.Keys[i]
	u := 0.0
	if span := b.Beat - a.Beat; span > 0 {
		u = a.Ease(math.Max(0, math.Min(1, (beat-a.Beat)/span)))
	}
	return a.Num + (b.Num-a.Num)*u, a.Color.Lerp(b.Color, float32(u)), true
}
