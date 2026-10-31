package pixel

import (
	"fmt"
	"math"
	"strings"
)

// Order describes how a pixel's channels appear on the wire, e.g. "GRB" for
// WS2812 or "GRBW" for SK6812 RGBW. Each entry is a component index:
// 0=R, 1=G, 2=B, 3=W.
type Order struct {
	name string
	idx  []uint8
}

// SupportedOrders lists the colour orders offered in the UI.
var SupportedOrders = []string{
	"RGB", "RBG", "GRB", "GBR", "BRG", "BGR",
	"RGBW", "GRBW", "BRGW", "WRGB",
}

// ParseOrder parses a colour order string like "GRB" or "RGBW".
func ParseOrder(s string) (Order, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 3 && len(s) != 4 {
		return Order{}, fmt.Errorf("colour order %q: want 3 or 4 letters", s)
	}
	o := Order{name: s, idx: make([]uint8, len(s))}
	seen := map[rune]bool{}
	for i, r := range s {
		if seen[r] {
			return Order{}, fmt.Errorf("colour order %q: repeated %c", s, r)
		}
		seen[r] = true
		switch r {
		case 'R':
			o.idx[i] = 0
		case 'G':
			o.idx[i] = 1
		case 'B':
			o.idx[i] = 2
		case 'W':
			o.idx[i] = 3
		default:
			return Order{}, fmt.Errorf("colour order %q: bad letter %c", s, r)
		}
	}
	if len(s) == 4 && !seen['W'] || len(s) == 3 && seen['W'] {
		return Order{}, fmt.Errorf("colour order %q: needs R, G, B (+W for 4 channels)", s)
	}
	return o, nil
}

// MustOrder is ParseOrder that panics; for tests and constants.
func MustOrder(s string) Order {
	o, err := ParseOrder(s)
	if err != nil {
		panic(err)
	}
	return o
}

func (o Order) String() string { return o.name }

// Channels is the number of DMX channels per pixel (3 or 4).
func (o Order) Channels() int { return len(o.idx) }

// HasWhite reports whether the order includes a white channel.
func (o Order) HasWhite() bool { return len(o.idx) == 4 }

// EncodeOptions control the final colour-to-byte conversion.
type EncodeOptions struct {
	Brightness float32 // 0..1 overall scale (master x strip)
	Gamma      *Gamma  // nil = linear
	// ExtractWhite moves the common (grey) part of RGB into the W channel
	// on RGBW pixels. If false, W is left at 0.
	ExtractWhite bool
}

// Encode writes len(px)*o.Channels() bytes into dst.
func (o Order) Encode(dst []byte, px []Color, opt EncodeOptions) {
	ch := o.Channels()
	var comp [4]float32
	for i, c := range px {
		c = c.Scale(opt.Brightness).Clamp()
		comp[0], comp[1], comp[2], comp[3] = c.R, c.G, c.B, 0
		if ch == 4 && opt.ExtractWhite {
			w := min(c.R, c.G, c.B)
			comp[0], comp[1], comp[2], comp[3] = c.R-w, c.G-w, c.B-w, w
		}
		base := i * ch
		for j, k := range o.idx {
			dst[base+j] = opt.Gamma.Apply(comp[k])
		}
	}
}

// Gamma is a precomputed lookup table for perceptual brightness correction.
type Gamma struct {
	lut   [4096]uint8
}

// NewGamma builds a gamma table. g <= 1 means linear.
func NewGamma(g float64) *Gamma {
	if g <= 0 {
		g = 1
	}
	t := &Gamma{}
	for i := range t.lut {
		v := math.Pow(float64(i)/float64(len(t.lut)-1), g)
		t.lut[i] = uint8(math.Round(v * 255))
	}
	return t
}

// Apply converts a 0..1 component to a byte. A nil table is linear.
func (g *Gamma) Apply(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	if g == nil {
		return uint8(v*255 + 0.5)
	}
	return g.lut[int(v*float32(len(g.lut)-1)+0.5)]
}
