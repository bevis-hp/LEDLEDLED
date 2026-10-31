package effects

import (
	"math"
	"testing"

	"ledledled/internal/audio"
	"ledledled/internal/pixel"
)

// TestAllEffectsRender runs every effect with default params over a range
// of times and checks the output stays finite and in range.
func TestAllEffectsRender(t *testing.T) {
	for _, d := range All() {
		params, err := FillDefaults(d.Name, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range []int{1, 7, 200} {
			px := make([]pixel.Color, n)
			for _, tm := range []float64{0, 0.37, 12.5, 1e5} {
				ctx := &Context{Time: tm, Params: NewParams(d, params)}
				d.Render(ctx, px)
				for i, c := range px {
					for _, v := range []float32{c.R, c.G, c.B} {
						if math.IsNaN(float64(v)) || v < -1e-4 || v > 1+1e-4 {
							t.Fatalf("%s n=%d t=%v px[%d]=%v out of range", d.Name, n, tm, i, c)
						}
					}
				}
			}
		}
	}
}

func TestFillDefaultsReplacesBadTypes(t *testing.T) {
	p, err := FillDefaults("solid", map[string]any{"color": 12.0, "bogus": true})
	if err != nil {
		t.Fatal(err)
	}
	if p["color"] != "#ff0000" {
		t.Errorf("color = %v, want default", p["color"])
	}
	if _, ok := p["bogus"]; ok {
		t.Error("unknown param not dropped")
	}
	if _, err := FillDefaults("nope", nil); err == nil {
		t.Error("expected unknown effect error")
	}
}

func TestNoiseRange(t *testing.T) {
	lo, hi := 1.0, -1.0
	for i := range 10000 {
		v := noise2(float64(i)*0.137, float64(i)*0.071)
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if lo < -1.5 || hi > 1.5 || hi-lo < 0.8 {
		t.Errorf("noise range %v..%v looks wrong", lo, hi)
	}
}

func TestVUFollowsAudioLevel(t *testing.T) {
	d, _ := Get("vu")
	params, _ := FillDefaults("vu", map[string]any{"colorA": "#ffffff", "colorB": "#ffffff"})
	px := make([]pixel.Color, 11)
	ctx := &Context{Audio: &audio.Features{Level: 0.5}, Params: NewParams(d, params)}
	d.Render(ctx, px)
	if px[0].R != 1 || px[4].R != 1 || px[6].R != 0 || px[10].R != 0 {
		t.Errorf("vu at 0.5: %v", px)
	}
}
