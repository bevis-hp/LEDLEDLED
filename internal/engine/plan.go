package engine

import (
	"errors"
	"fmt"
	"net/netip"

	"ledledled/internal/effects"
	"ledledled/internal/output"
	"ledledled/internal/pixel"
	"ledledled/internal/show"
	"ledledled/internal/timeline"
)

// plan is a show compiled into flat, allocation-free per-frame structures.
// It is rebuilt whenever the show changes and never mutated by the UI.
type plan struct {
	show        *show.Show
	strips      []stripPlan
	layers      []layerPlan
	tracks      []trackPlan
	controllers []ctrlPlan
	gamma       *pixel.Gamma
	masterK     float32 // this frame's master modulation factor
}

type stripPlan struct {
	id         string
	px         []pixel.Color // rendered colours, logical order (pixel 0 = start)
	scratch    []pixel.Color // reversal buffer for Reverse strips
	brightness float32
	reverse    bool
	white      bool // extract white on RGBW

	power   pixel.PowerModel
	maxMA   float64
	powered bool // attached to a controller, so it has encoded output
	// Power measurements: last frame, and peaks over the current and the
	// previous one-second window (the previous one is what we report).
	ma, scale         float64
	winMA, winReqMA   float64
	winLimited        bool
	peakMA, peakReqMA float64
	limited           bool
}

// trackPlan is a compiled timeline track bound to a layer in plan.layers.
type trackPlan struct {
	layer   int
	param   string
	opacity bool // animates the layer's opacity rather than an effect param
	track   timeline.Track
}

// pixelRef addresses one pixel of one strip.
type pixelRef struct{ strip, idx int32 }

type layerPlan struct {
	id        string
	tempoSync bool
	def       *effects.Def
	params    effects.Params
	opacity   float32 // stored opacity
	opNow     float32 // this frame's opacity (after keyframes/modulators)
	mods      []show.Modulator
	blend     blendFunc
	targets   []pixelRef    // group pixels in group order
	buf       []pixel.Color // effect output, len(targets)
}

type ctrlPlan struct {
	id            string
	proto         output.Protocol
	dst           netip.AddrPort
	startUniverse int
	universeSize  int
	universes     int
	channels      int
	strips        []show.StripPatch
	stream        []byte // full channel stream for this controller
	disabled      bool
	warnings      []string
}

// Prepare normalises a show, fills effect defaults and validates it. The
// show is modified in place.
func Prepare(s *show.Show) error {
	s.Normalize()
	var errs []error
	for i := range s.Layers {
		l := &s.Layers[i]
		p, err := effects.FillDefaults(l.Effect, l.Params)
		if err != nil {
			errs = append(errs, fmt.Errorf("layer %q: %w", l.Name, err))
			continue
		}
		l.Params = p
	}
	pruneTracks(s)
	pruneMods(s)
	errs = append(errs, s.Validate())
	return errors.Join(errs...)
}

// trackKind reports what a track on (layer, param) animates, or false if
// the parameter can't be animated (unknown, bool or select).
func trackKind(l *show.Layer, param string) (timeline.Kind, bool) {
	if param == "opacity" {
		return timeline.Number, true
	}
	def, ok := effects.Get(l.Effect)
	if !ok {
		return 0, false
	}
	pd, ok := def.Param(param)
	switch {
	case !ok:
		return 0, false
	case pd.Type == effects.Number:
		return timeline.Number, true
	case pd.Type == effects.Color:
		return timeline.Color, true
	}
	return 0, false
}

// pruneMods drops modulators whose parameter isn't numeric on the layer's
// current effect (e.g. after the effect was changed).
func pruneMods(s *show.Show) {
	for i := range s.Layers {
		l := &s.Layers[i]
		kept := l.Mods[:0]
		for _, m := range l.Mods {
			if kind, ok := trackKind(l, m.Param); ok && kind == timeline.Number {
				kept = append(kept, m)
			}
		}
		l.Mods = kept
	}
}

// pruneTracks drops timeline tracks and keys that no longer make sense,
// e.g. after a layer was deleted or its effect changed. Stale tracks are
// expected during normal editing, so this is silent rather than an error.
func pruneTracks(s *show.Show) {
	layers := map[string]*show.Layer{}
	for i := range s.Layers {
		layers[s.Layers[i].ID] = &s.Layers[i]
	}
	kept := s.Timeline.Tracks[:0]
	seen := map[[2]string]bool{}
	for _, tr := range s.Timeline.Tracks {
		l, ok := layers[tr.LayerID]
		if !ok || seen[[2]string{tr.LayerID, tr.Param}] {
			continue
		}
		kind, ok := trackKind(l, tr.Param)
		if !ok {
			continue
		}
		keys := tr.Keys[:0]
		for _, k := range tr.Keys {
			if _, err := timeline.Compile(kind, []timeline.RawKey{{Value: k.Value}}); err == nil {
				keys = append(keys, k)
			}
		}
		tr.Keys = keys
		seen[[2]string{tr.LayerID, tr.Param}] = true
		kept = append(kept, tr)
	}
	s.Timeline.Tracks = kept
}

// compile builds a plan from a prepared, valid show.
func compile(s *show.Show) (*plan, error) {
	p := &plan{show: s, gamma: pixel.NewGamma(s.Settings.Gamma), masterK: 1}

	stripIdx := map[string]int{}
	for i, st := range s.Strips {
		stripIdx[st.ID] = i
		p.strips = append(p.strips, stripPlan{
			id:         st.ID,
			power:      pixel.PowerModel{Volts: st.Volts, MilliampsPerChannel: st.MilliampsPerChannel, IdleMilliamps: st.IdleMilliamps},
			maxMA:      st.MaxAmps * 1000,
			scale:      1,
			px:         make([]pixel.Color, st.Pixels),
			scratch:    make([]pixel.Color, st.Pixels),
			brightness: float32(st.Brightness),
			reverse:    st.Reverse,
			white:      st.WhiteMode != "off",
		})
	}

	groups := map[string]show.Group{}
	for _, g := range s.Groups {
		groups[g.ID] = g
	}

	for _, l := range s.Layers {
		if l.Muted {
			continue
		}
		def, ok := effects.Get(l.Effect)
		if !ok {
			return nil, fmt.Errorf("layer %q: unknown effect %q", l.Name, l.Effect)
		}
		var targets []pixelRef
		if l.GroupID == "" {
			// No group: every strip end to end, in show order.
			for si, st := range s.Strips {
				for i := range st.Pixels {
					targets = append(targets, pixelRef{int32(si), int32(i)})
				}
			}
		} else {
			for _, sec := range groups[l.GroupID].Sections {
				si := int32(stripIdx[sec.StripID])
				for k := range sec.Len() {
					i := sec.Start + k
					if sec.Reverse {
						i = sec.End - k
					}
					targets = append(targets, pixelRef{si, int32(i)})
				}
			}
		}
		p.layers = append(p.layers, layerPlan{
			id:        l.ID,
			tempoSync: l.TempoSync,
			mods:      l.Mods,
			def:       def,
			params:    effects.NewParams(def, l.Params),
			opacity:   float32(l.Opacity),
			blend:     blendFuncs[l.Blend],
			targets:   targets,
			buf:       make([]pixel.Color, len(targets)),
		})
	}

	layerIdx := map[string]int{}
	for i, lp := range p.layers {
		layerIdx[lp.id] = i
	}
	for i := range s.Layers {
		l := &s.Layers[i]
		li, ok := layerIdx[l.ID]
		if !ok {
			continue // muted
		}
		for _, tr := range s.Timeline.Tracks {
			if tr.LayerID != l.ID || len(tr.Keys) == 0 {
				continue
			}
			kind, _ := trackKind(l, tr.Param)
			raw := make([]timeline.RawKey, len(tr.Keys))
			for k, key := range tr.Keys {
				raw[k] = timeline.RawKey{Beat: key.Beat, Value: key.Value, Ease: key.Ease}
			}
			ct, err := timeline.Compile(kind, raw)
			if err != nil {
				return nil, fmt.Errorf("track %s/%s: %w", l.Name, tr.Param, err)
			}
			p.tracks = append(p.tracks, trackPlan{layer: li, param: tr.Param, opacity: tr.Param == "opacity", track: ct})
		}
	}

	for ci, cp := range s.Patch() {
		c := s.Controllers[ci]
		addr, err := netip.ParseAddr(c.IP)
		if err != nil {
			return nil, fmt.Errorf("controller %q: %w", c.Name, err)
		}
		port := c.Port
		if port == 0 {
			port = c.Protocol.DefaultPort()
		}
		universes := cp.Universes
		if c.MaxUniverses > 0 {
			universes = min(universes, c.MaxUniverses)
		}
		p.controllers = append(p.controllers, ctrlPlan{
			id:            c.ID,
			proto:         c.Protocol,
			dst:           netip.AddrPortFrom(addr, uint16(port)),
			startUniverse: c.StartUniverse,
			universeSize:  cp.UniverseSize,
			universes:     universes,
			channels:      cp.Channels,
			strips:        cp.Strips,
			stream:        make([]byte, cp.Channels),
			disabled:      c.Disabled,
			warnings:      cp.Warnings,
		})
	}
	return p, nil
}

// universeData returns the slice of the stream sent as universe u (0-based).
func (c *ctrlPlan) universeData(u int) []byte {
	lo := u * c.universeSize
	hi := min(lo+c.universeSize, len(c.stream))
	return c.stream[lo:hi]
}
