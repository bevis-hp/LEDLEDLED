package engine

import "ledledled/internal/pixel"

// blendFunc combines a layer pixel (src) onto what's below (dst).
type blendFunc func(dst, src pixel.Color, opacity float32) pixel.Color

var blendFuncs = map[string]blendFunc{
	"normal": func(d, s pixel.Color, o float32) pixel.Color { return d.Lerp(s, o) },
	"add": func(d, s pixel.Color, o float32) pixel.Color {
		return pixel.Color{R: d.R + s.R*o, G: d.G + s.G*o, B: d.B + s.B*o}.Clamp()
	},
	"multiply": func(d, s pixel.Color, o float32) pixel.Color {
		return d.Lerp(pixel.Color{R: d.R * s.R, G: d.G * s.G, B: d.B * s.B}, o)
	},
	"screen": func(d, s pixel.Color, o float32) pixel.Color {
		scr := pixel.Color{R: 1 - (1-d.R)*(1-s.R), G: 1 - (1-d.G)*(1-s.G), B: 1 - (1-d.B)*(1-s.B)}
		return d.Lerp(scr, o)
	},
	"max": func(d, s pixel.Color, o float32) pixel.Color {
		return d.Lerp(pixel.Color{R: max(d.R, s.R), G: max(d.G, s.G), B: max(d.B, s.B)}, o)
	},
}
