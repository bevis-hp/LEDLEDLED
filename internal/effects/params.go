package effects

import "ledledled/internal/pixel"

// Params gives effects typed access to a layer's parameter values.
//
// Lookup order: per-frame overrides (set by the engine from keyframes and
// audio modulators) -> the layer's stored value -> the effect's default.
// Overrides live in maps that are cleared and refilled every frame, so
// animating a parameter never allocates.
type Params struct {
	def      *Def
	values   map[string]any
	nums     map[string]float64
	colors   map[string]pixel.Color
	palettes map[string]*pixel.Palette // by ID, shared by all layers
}

// NewParams binds a parameter map to an effect definition.
func NewParams(d *Def, values map[string]any) Params {
	return Params{def: d, values: values, nums: map[string]float64{}, colors: map[string]pixel.Color{}}
}

// WithPalettes returns params that resolve palette IDs through pals.
func (p Params) WithPalettes(pals map[string]*pixel.Palette) Params {
	p.palettes = pals
	return p
}

// Palette returns the palette chosen for a palette parameter, or nil when
// none is selected (or it no longer exists): effects then fall back to
// their own colour parameters.
func (p Params) Palette(name string) *pixel.Palette {
	id, _ := p.lookup(name).(string)
	if id == "" {
		return nil
	}
	return p.palettes[id]
}

// Def returns the effect definition the params belong to.
func (p Params) Def() *Def { return p.def }

// ClearOverrides drops all per-frame overrides.
func (p Params) ClearOverrides() {
	clear(p.nums)
	clear(p.colors)
}

// OverrideNum sets a numeric parameter for this frame.
func (p Params) OverrideNum(name string, v float64) { p.nums[name] = v }

// OverrideColor sets a colour parameter for this frame.
func (p Params) OverrideColor(name string, c pixel.Color) { p.colors[name] = c }

func (p Params) lookup(name string) any {
	if v, ok := p.values[name]; ok {
		return v
	}
	for _, pd := range p.def.Params {
		if pd.Name == name {
			return pd.Default
		}
	}
	return nil
}

// Num returns a numeric parameter.
func (p Params) Num(name string) float64 {
	if v, ok := p.nums[name]; ok {
		return v
	}
	switch v := p.lookup(name).(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

// Color returns a colour parameter.
func (p Params) Color(name string) pixel.Color {
	if c, ok := p.colors[name]; ok {
		return c
	}
	if s, ok := p.lookup(name).(string); ok {
		if c, err := pixel.ParseHex(s); err == nil {
			return c
		}
	}
	return pixel.Black
}

// Bool returns a boolean parameter.
func (p Params) Bool(name string) bool {
	b, _ := p.lookup(name).(bool)
	return b
}

// Str returns a select parameter.
func (p Params) Str(name string) string {
	s, _ := p.lookup(name).(string)
	return s
}
