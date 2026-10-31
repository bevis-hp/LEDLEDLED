package effects

import (
	"math"

	"ledledled/internal/pixel"
)

// pos returns pixel i's normalised position 0..1 along a group of n pixels.
func pos(i, n int) float64 {
	if n <= 1 {
		return 0
	}
	return float64(i) / float64(n-1)
}

// frac returns the fractional part of x, always in 0..1.
func frac(x float64) float64 { return x - math.Floor(x) }

func init() {
	register(&Def{
		Name: "solid", Label: "Solid colour",
		Description: "Fills the group with a single colour.",
		Params: []ParamDef{
			{Name: "color", Label: "Colour", Type: Color, Default: "#ff0000"},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			c := ctx.Params.Color("color")
			for i := range px {
				px[i] = c
			}
		},
	})

	register(&Def{
		Name: "gradient", Label: "Gradient",
		Description: "Blends from colour A to colour B along the group. With a non-zero speed it becomes a seamless A-B-A loop that scrolls.",
		Params: []ParamDef{
			{Name: "colorA", Label: "Colour A", Type: Color, Default: "#ff0040"},
			{Name: "colorB", Label: "Colour B", Type: Color, Default: "#0040ff"},
			{Name: "palette", Label: "Palette (replaces colours)", Type: Palette, Default: ""},
			{Name: "speed", Label: "Scroll speed", Type: Number, Default: 0.0, Min: -2, Max: 2, Step: 0.01},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			a, b := ctx.Params.Color("colorA"), ctx.Params.Color("colorB")
			pal := ctx.Params.Palette("palette")
			speed := ctx.Params.Num("speed")
			for i := range px {
				t := pos(i, len(px))
				if speed != 0 {
					// Triangle wave so the loop has no hard seam.
					t = 1 - math.Abs(2*frac(t-ctx.Time*speed)-1)
				}
				if pal != nil {
					px[i] = pal.At(t)
				} else {
					px[i] = a.Lerp(b, float32(t))
				}
			}
		},
	})

	register(&Def{
		Name: "rainbow", Label: "Rainbow",
		Description: "Scrolling hue rainbow.",
		Params: []ParamDef{
			{Name: "speed", Label: "Speed (cycles/s)", Type: Number, Default: 0.2, Min: -3, Max: 3, Step: 0.01},
			{Name: "scale", Label: "Repeats across group", Type: Number, Default: 1.0, Min: 0, Max: 10, Step: 0.1},
			{Name: "saturation", Label: "Saturation", Type: Number, Default: 1.0, Min: 0, Max: 1, Step: 0.01},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			speed, scale := ctx.Params.Num("speed"), ctx.Params.Num("scale")
			sat := float32(ctx.Params.Num("saturation"))
			for i := range px {
				h := pos(i, len(px))*scale - ctx.Time*speed
				px[i] = pixel.HSV(float32(frac(h)), sat, 1)
			}
		},
	})

	register(&Def{
		Name: "noise", Label: "Noise",
		Description: "Smoothly drifting organic noise blended between two colours.",
		Params: []ParamDef{
			{Name: "colorA", Label: "Colour A", Type: Color, Default: "#000000"},
			{Name: "colorB", Label: "Colour B", Type: Color, Default: "#ff6a00"},
			{Name: "palette", Label: "Palette (replaces colours)", Type: Palette, Default: ""},
			{Name: "scale", Label: "Scale (blobs across group)", Type: Number, Default: 4.0, Min: 0.1, Max: 40, Step: 0.1},
			{Name: "speed", Label: "Speed", Type: Number, Default: 0.5, Min: 0, Max: 5, Step: 0.01},
			{Name: "detail", Label: "Detail (octaves)", Type: Number, Default: 2.0, Min: 1, Max: 5, Step: 1},
			{Name: "contrast", Label: "Contrast", Type: Number, Default: 1.0, Min: 0.1, Max: 4, Step: 0.05},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			a, b := ctx.Params.Color("colorA"), ctx.Params.Color("colorB")
			scale, speed := ctx.Params.Num("scale"), ctx.Params.Num("speed")
			oct := int(ctx.Params.Num("detail"))
			contrast := ctx.Params.Num("contrast")
			pal := ctx.Params.Palette("palette")
			for i := range px {
				n := fbm(pos(i, len(px))*scale, ctx.Time*speed, max(oct, 1))
				t := math.Max(0, math.Min(1, 0.5+0.5*n*contrast))
				if pal != nil {
					px[i] = pal.At(t)
				} else {
					px[i] = a.Lerp(b, float32(t))
				}
			}
		},
	})

	register(&Def{
		Name: "pulse", Label: "Pulse",
		Description: "Breathes the group between a background and a colour. Ripple offsets the phase along the group so the pulse travels.",
		Params: []ParamDef{
			{Name: "color", Label: "Colour", Type: Color, Default: "#ffffff"},
			{Name: "background", Label: "Background", Type: Color, Default: "#000000"},
			{Name: "rate", Label: "Rate (pulses/s)", Type: Number, Default: 0.5, Min: 0.01, Max: 10, Step: 0.01},
			{Name: "ripple", Label: "Ripple (waves across group)", Type: Number, Default: 0.0, Min: -5, Max: 5, Step: 0.05},
			{Name: "sharpness", Label: "Sharpness", Type: Number, Default: 1.0, Min: 0.2, Max: 10, Step: 0.1},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			c, bg := ctx.Params.Color("color"), ctx.Params.Color("background")
			rate, ripple := ctx.Params.Num("rate"), ctx.Params.Num("ripple")
			sharp := ctx.Params.Num("sharpness")
			for i := range px {
				phase := ctx.Time*rate - pos(i, len(px))*ripple
				v := 0.5 - 0.5*math.Cos(2*math.Pi*phase)
				px[i] = bg.Lerp(c, float32(math.Pow(v, sharp)))
			}
		},
	})

	register(&Def{
		Name: "chase", Label: "Chase",
		Description: "Pulses of light travelling along the group, with soft tails.",
		Params: []ParamDef{
			{Name: "color", Label: "Colour", Type: Color, Default: "#00ffff"},
			{Name: "background", Label: "Background", Type: Color, Default: "#000000"},
			{Name: "speed", Label: "Speed (group lengths/s)", Type: Number, Default: 0.5, Min: -5, Max: 5, Step: 0.01},
			{Name: "count", Label: "Pulse count", Type: Number, Default: 1.0, Min: 1, Max: 20, Step: 1},
			{Name: "width", Label: "Width", Type: Number, Default: 0.3, Min: 0.01, Max: 1, Step: 0.01},
			{Name: "tail", Label: "Tail softness", Type: Number, Default: 0.8, Min: 0, Max: 1, Step: 0.01},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			c, bg := ctx.Params.Color("color"), ctx.Params.Color("background")
			speed, count := ctx.Params.Num("speed"), math.Max(1, math.Round(ctx.Params.Num("count")))
			width, tail := ctx.Params.Num("width"), ctx.Params.Num("tail")
			n := len(px)
			for i := range px {
				// Distance behind the pulse head, 0..1 within each repeat.
				p := float64(i) / float64(max(n, 1)) * count
				d := frac(ctx.Time*speed*count - p)
				if speed < 0 {
					d = 1 - d
				}
				var v float64
				if d < width {
					x := d / width // 0 at head, 1 at end of tail
					v = 1 - x*tail
					v = math.Pow(v, 1+tail*2)
				}
				px[i] = bg.Lerp(c, float32(v))
			}
		},
	})

	register(&Def{
		Name: "sparkle", Label: "Sparkle",
		Description: "Random pixels flash and fade out.",
		Params: []ParamDef{
			{Name: "color", Label: "Colour", Type: Color, Default: "#ffffff"},
			{Name: "palette", Label: "Palette (random colour per flash)", Type: Palette, Default: ""},
			{Name: "background", Label: "Background", Type: Color, Default: "#000000"},
			{Name: "density", Label: "Density", Type: Number, Default: 0.1, Min: 0, Max: 1, Step: 0.01},
			{Name: "speed", Label: "Flashes/s per pixel", Type: Number, Default: 2.0, Min: 0.1, Max: 20, Step: 0.1},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			c, bg := ctx.Params.Color("color"), ctx.Params.Color("background")
			density, speed := ctx.Params.Num("density"), ctx.Params.Num("speed")
			pal := ctx.Params.Palette("palette")
			for i := range px {
				// Each pixel has its own phase so flashes don't sync up.
				t := ctx.Time*speed + hash01(i, 7919)
				slot := int(math.Floor(t))
				var v float64
				fc := c
				if hash01(i, slot) < density {
					v = math.Pow(1-frac(t), 3) // quick flash, exponential-ish fade
					if pal != nil {
						fc = pal.At(hash01(slot, i+101))
					}
				}
				px[i] = bg.Lerp(fc, float32(v))
			}
		},
	})

	register(&Def{
		Name: "palette", Label: "Palette scroll",
		Description: "Spreads a palette along the group and scrolls it. Wraps seamlessly; mirror plays it there-and-back.",
		Params: []ParamDef{
			{Name: "palette", Label: "Palette", Type: Palette, Default: "rainbow"},
			{Name: "speed", Label: "Speed (cycles/s)", Type: Number, Default: 0.1, Min: -3, Max: 3, Step: 0.01},
			{Name: "repeats", Label: "Repeats across group", Type: Number, Default: 1.0, Min: 0.05, Max: 10, Step: 0.05},
			{Name: "mirror", Label: "Mirror", Type: Bool, Default: false},
		},
		Render: func(ctx *Context, px []pixel.Color) {
			pal := ctx.Params.Palette("palette")
			if pal == nil {
				clear(px)
				return
			}
			speed, reps := ctx.Params.Num("speed"), ctx.Params.Num("repeats")
			mirror := ctx.Params.Bool("mirror")
			n := len(px)
			for i := range px {
				u := frac(float64(i)/float64(max(n, 1))*reps - ctx.Time*speed)
				if mirror {
					u = 1 - math.Abs(2*u-1)
				}
				px[i] = pal.At(u)
			}
		},
	})
}
