package pixel

import (
	"math"
	"sort"
)

// Stop is one colour at a position (0..1) in a palette.
type Stop struct {
	Pos   float64
	Color Color
}

// Palette is a multi-stop gradient, precomputed into a lookup table so
// effects can sample it per pixel cheaply.
type Palette struct {
	lut [256]Color
}

// NewPalette builds a palette from stops (any order). Colours are blended
// linearly between stops and held flat before the first / after the last.
func NewPalette(stops []Stop) *Palette {
	p := &Palette{}
	if len(stops) == 0 {
		return p
	}
	s := append([]Stop(nil), stops...)
	sort.SliceStable(s, func(i, j int) bool { return s[i].Pos < s[j].Pos })
	k := 0
	for i := range p.lut {
		u := float64(i) / float64(len(p.lut)-1)
		for k < len(s)-1 && s[k+1].Pos <= u {
			k++
		}
		switch {
		case u <= s[0].Pos:
			p.lut[i] = s[0].Color
		case k == len(s)-1:
			p.lut[i] = s[k].Color
		default:
			a, b := s[k], s[k+1]
			t := 0.0
			if span := b.Pos - a.Pos; span > 0 {
				t = (u - a.Pos) / span
			}
			p.lut[i] = a.Color.Lerp(b.Color, float32(t))
		}
	}
	return p
}

// At samples the palette at u, clamped to 0..1.
func (p *Palette) At(u float64) Color {
	if math.IsNaN(u) {
		u = 0
	}
	u = math.Max(0, math.Min(1, u))
	return p.lut[int(u*float64(len(p.lut)-1)+0.5)]
}
