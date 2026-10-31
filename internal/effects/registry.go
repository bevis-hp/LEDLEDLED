// Package effects contains the animation sources that layers use to paint
// pixels. Every effect is a pure function of time and parameters: given the
// same inputs it produces the same output. That keeps rendering
// deterministic and makes a future keyframe/timeline system straightforward,
// since it only needs to supply parameter values for a given time.
package effects

import (
	"fmt"
	"slices"
	"sort"

	"ledledled/internal/pixel"
)

// ParamType tells the UI which control to draw.
type ParamType string

const (
	Number ParamType = "number"
	Color  ParamType = "color"
	Bool   ParamType = "bool"
	Select ParamType = "select" // one of Options
)

// ParamDef describes one effect parameter.
type ParamDef struct {
	Name    string    `json:"name"`
	Label   string    `json:"label"`
	Type    ParamType `json:"type"`
	Default any       `json:"default"`
	Min     float64   `json:"min,omitempty"`
	Max     float64   `json:"max,omitempty"`
	Step    float64   `json:"step,omitempty"`
	Options []string  `json:"options,omitempty"`
}

// Context is passed to an effect each frame.
type Context struct {
	// Time drives the animation. It is seconds, or beats when the layer
	// is tempo-synced, so "1 cycle per second" becomes "1 cycle per beat".
	Time   float64
	Beat   float64 // global beat position (always beats)
	Params Params
}

// RenderFunc fills px (the pixels of a layer's group, in group order).
type RenderFunc func(ctx *Context, px []pixel.Color)

// Def is a registered effect.
type Def struct {
	Name        string     `json:"name"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Params      []ParamDef `json:"params"`
	Render      RenderFunc `json:"-"`
}

var registry = map[string]*Def{}

func register(d *Def) {
	if _, dup := registry[d.Name]; dup {
		panic("duplicate effect " + d.Name)
	}
	registry[d.Name] = d
}

// Get returns an effect by name.
func Get(name string) (*Def, bool) {
	d, ok := registry[name]
	return d, ok
}

// All returns every effect sorted by label, for the UI.
func All() []*Def {
	out := make([]*Def, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// FillDefaults adds any missing parameters with their default values and
// drops parameters the effect doesn't know about.
func FillDefaults(name string, params map[string]any) (map[string]any, error) {
	d, ok := Get(name)
	if !ok {
		return params, fmt.Errorf("unknown effect %q", name)
	}
	out := make(map[string]any, len(d.Params))
	for _, p := range d.Params {
		v, ok := params[p.Name]
		if !ok || !TypeMatches(p, v) {
			v = p.Default
		}
		out[p.Name] = v
	}
	return out, nil
}

// Param returns the definition of a named parameter.
func (d *Def) Param(name string) (ParamDef, bool) {
	for _, p := range d.Params {
		if p.Name == name {
			return p, true
		}
	}
	return ParamDef{}, false
}

// TypeMatches reports whether v is a valid value for parameter p.
func TypeMatches(p ParamDef, v any) bool {
	switch p.Type {
	case Number:
		_, ok := v.(float64)
		return ok
	case Color:
		s, ok := v.(string)
		if !ok {
			return false
		}
		_, err := pixel.ParseHex(s)
		return err == nil
	case Bool:
		_, ok := v.(bool)
		return ok
	case Select:
		s, ok := v.(string)
		return ok && slices.Contains(p.Options, s)
	}
	return false
}
