package effects

import (
	"math"

	"ledledled/internal/audio"
	"ledledled/internal/pixel"
)

// Audio-reactive effects. They read ctx.Audio, which is silence (all zero)
// when audio input is disabled.

func init() {
	register(&Def{
		Name: "vu", Label: "VU meter",
		Description: "A bar that grows with the chosen audio signal. Mirror grows it from the centre outwards.",
		Params: []ParamDef{
			{Name: "source", Label: "Signal", Type: Select, Default: "level", Options: []string{"level", "bass", "mid", "treble", "beat"}},
			{Name: "colorA", Label: "Low colour", Type: Color, Default: "#00ff40"},
			{Name: "colorB", Label: "High colour", Type: Color, Default: "#ff0000"},
			{Name: "background", Label: "Background", Type: Color, Default: "#000000"},
			{Name: "mirror", Label: "Mirror from centre", Type: Bool, Default: false},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			a, b := ctx.Params.Color("colorA"), ctx.Params.Color("colorB")
			bg := ctx.Params.Color("background")
			v := ctx.AudioFeatures().Get(ctx.Params.Str("source"))
			mirror := ctx.Params.Bool("mirror")
			n := len(px)
			for i := range px {
				u := pos(i, n)
				if mirror {
					u = math.Abs(2*u - 1)
				}
				// Soft leading edge, one pixel wide.
				edge := float64(n)
				if mirror {
					edge /= 2
				}
				lit := math.Max(0, math.Min(1, (v-u)*edge+1))
				px[i] = bg.Lerp(a.Lerp(b, float32(u)), float32(lit))
			}
		},
	})

	register(&Def{
		Name: "spectrum", Label: "Spectrum",
		Description: "Frequency bands spread along the group (bass at the start), each pixel's brightness following its band.",
		Params: []ParamDef{
			{Name: "colorA", Label: "Bass colour", Type: Color, Default: "#ff0040"},
			{Name: "colorB", Label: "Treble colour", Type: Color, Default: "#2060ff"},
			{Name: "floor", Label: "Minimum brightness", Type: Number, Default: 0.0, Min: 0, Max: 1, Step: 0.01},
			{Name: "mirror", Label: "Mirror from centre", Type: Bool, Default: false},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			a, b := ctx.Params.Color("colorA"), ctx.Params.Color("colorB")
			floor := ctx.Params.Num("floor")
			mirror := ctx.Params.Bool("mirror")
			bands := &ctx.AudioFeatures().Bands
			for i := range px {
				u := pos(i, len(px))
				if mirror {
					u = math.Abs(2*u - 1)
				}
				f := u * (audio.NumBands - 1)
				k := int(f)
				v := bands[k]
				if k+1 < audio.NumBands {
					v += (bands[k+1] - v) * (f - float64(k))
				}
				v = floor + (1-floor)*v
				px[i] = a.Lerp(b, float32(u)).Scale(float32(v))
			}
		},
	})
}
