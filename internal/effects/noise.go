package effects

import "math"

// noise2 is 2D gradient (Perlin-style) noise returning roughly -1..1.
// Effects sample it along (position, time) so patterns drift smoothly.
func noise2(x, y float64) float64 {
	xi, yi := math.Floor(x), math.Floor(y)
	xf, yf := x-xi, y-yi
	ix, iy := int(xi)&255, int(yi)&255

	u, v := fade(xf), fade(yf)
	aa := perm[perm[ix]+iy]
	ab := perm[perm[ix]+iy+1]
	ba := perm[perm[ix+1]+iy]
	bb := perm[perm[ix+1]+iy+1]

	x1 := lerp(grad(aa, xf, yf), grad(ba, xf-1, yf), u)
	x2 := lerp(grad(ab, xf, yf-1), grad(bb, xf-1, yf-1), u)
	return lerp(x1, x2, v) * 1.4 // scale to roughly fill -1..1
}

// fbm layers several octaves of noise for a more organic look.
func fbm(x, y float64, octaves int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for range octaves {
		sum += noise2(x, y) * amp
		norm += amp
		x, y, amp = x*2, y*2, amp*0.5
	}
	return sum / norm
}

func fade(t float64) float64       { return t * t * t * (t*(t*6-15) + 10) }
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func grad(h int, x, y float64) float64 {
	switch h & 7 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x
	case 5:
		return -x
	case 6:
		return y
	default:
		return -y
	}
}

// hash01 is a cheap deterministic hash of two integers to 0..1, used for
// per-pixel randomness that doesn't need stored state.
func hash01(a, b int) float64 {
	h := uint32(a)*374761393 + uint32(b)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h) / float64(math.MaxUint32)
}

// perm is Ken Perlin's reference permutation, doubled to avoid wrapping.
var perm = func() [512]int {
	p := [256]int{151, 160, 137, 91, 90, 15, 131, 13, 201, 95, 96, 53, 194, 233, 7, 225,
		140, 36, 103, 30, 69, 142, 8, 99, 37, 240, 21, 10, 23, 190, 6, 148, 247, 120, 234,
		75, 0, 26, 197, 62, 94, 252, 219, 203, 117, 35, 11, 32, 57, 177, 33, 88, 237, 149,
		56, 87, 174, 20, 125, 136, 171, 168, 68, 175, 74, 165, 71, 134, 139, 48, 27, 166,
		77, 146, 158, 231, 83, 111, 229, 122, 60, 211, 133, 230, 220, 105, 92, 41, 55, 46,
		245, 40, 244, 102, 143, 54, 65, 25, 63, 161, 1, 216, 80, 73, 209, 76, 132, 187, 208,
		89, 18, 169, 200, 196, 135, 130, 116, 188, 159, 86, 164, 100, 109, 198, 173, 186, 3,
		64, 52, 217, 226, 250, 124, 123, 5, 202, 38, 147, 118, 126, 255, 82, 85, 212, 207,
		206, 59, 227, 47, 16, 58, 17, 182, 189, 28, 42, 223, 183, 170, 213, 119, 248, 152, 2,
		44, 154, 163, 70, 221, 153, 101, 155, 167, 43, 172, 9, 129, 22, 39, 253, 19, 98, 108,
		110, 79, 113, 224, 232, 178, 185, 112, 104, 218, 246, 97, 228, 251, 34, 242, 193,
		238, 210, 144, 12, 191, 179, 162, 241, 81, 51, 145, 235, 249, 14, 239, 107, 49, 192,
		214, 31, 181, 199, 106, 157, 184, 84, 204, 176, 115, 121, 50, 45, 127, 4, 150, 254,
		138, 236, 205, 93, 222, 114, 67, 29, 24, 72, 243, 141, 128, 195, 78, 66, 215, 61,
		156, 180}
	var out [512]int
	for i := range out {
		out[i] = p[i&255]
	}
	return out
}()
