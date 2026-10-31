// Package pixel holds colour types and the conversion from rendered colours
// to raw DMX channel bytes in a strip's wire colour order.
package pixel

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Color is a linear RGB colour with components in 0..1. Effects render in
// this space; white channels and gamma are only applied at encode time.
type Color struct{ R, G, B float32 }

var Black = Color{}

// Scale multiplies every component by k.
func (c Color) Scale(k float32) Color { return Color{c.R * k, c.G * k, c.B * k} }

// Lerp blends from c to d by t (0..1).
func (c Color) Lerp(d Color, t float32) Color {
	return Color{c.R + (d.R-c.R)*t, c.G + (d.G-c.G)*t, c.B + (d.B-c.B)*t}
}

// Clamp limits all components to 0..1.
func (c Color) Clamp() Color { return Color{clamp01(c.R), clamp01(c.G), clamp01(c.B)} }

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ParseHex parses "#rrggbb" or "rrggbb".
func ParseHex(s string) (Color, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return Black, fmt.Errorf("colour %q: want #rrggbb", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return Black, fmt.Errorf("colour %q: %w", s, err)
	}
	return Color{
		R: float32((v>>16)&0xff) / 255,
		G: float32((v>>8)&0xff) / 255,
		B: float32(v&0xff) / 255,
	}, nil
}

// Hex formats c as "#rrggbb".
func (c Color) Hex() string {
	c = c.Clamp()
	return fmt.Sprintf("#%02x%02x%02x", to8(c.R), to8(c.G), to8(c.B))
}

func to8(v float32) uint8 { return uint8(math.Round(float64(v) * 255)) }

// HSV builds a colour from hue (0..1, wraps), saturation and value (0..1).
func HSV(h, s, v float32) Color {
	h = h - float32(math.Floor(float64(h)))
	i := int(h * 6)
	f := h*6 - float32(i)
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	switch i % 6 {
	case 0:
		return Color{v, t, p}
	case 1:
		return Color{q, v, p}
	case 2:
		return Color{p, v, t}
	case 3:
		return Color{p, q, v}
	case 4:
		return Color{t, p, v}
	default:
		return Color{v, p, q}
	}
}
