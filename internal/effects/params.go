package effects

import "ledledled/internal/pixel"

// Params gives effects typed access to a layer's parameter map. Missing or
// mistyped values fall back to the effect's declared defaults.
type Params struct {
	def    *Def
	values map[string]any
}

// NewParams binds a parameter map to an effect definition.
func NewParams(d *Def, values map[string]any) Params { return Params{def: d, values: values} }

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
