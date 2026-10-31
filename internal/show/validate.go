package show

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"ledledled/internal/output"
	"ledledled/internal/pixel"
)

// Normalize fills in missing IDs and defaults and clamps out-of-range
// values. It never fails; call Validate afterwards for real errors.
func (s *Show) Normalize() {
	st := &s.Settings
	if st.FPS <= 0 {
		st.FPS = 40
	}
	st.FPS = min(st.FPS, 120)
	st.Brightness = clamp(st.Brightness, 0, 1)
	if st.Gamma <= 0 {
		st.Gamma = 2.2
	}
	if st.SourceName == "" {
		st.SourceName = "LEDLEDLED"
	}
	if st.Priority <= 0 || st.Priority > 200 {
		st.Priority = 100
	}
	if s.Tempo.BPM <= 0 {
		s.Tempo.BPM = 120
	}
	s.Tempo.BPM = clamp(s.Tempo.BPM, 20, 300)
	if _, err := ParseCID(st.CID); err != nil {
		cid := output.NewCID()
		st.CID = hex.EncodeToString(cid[:])
	}

	// Empty lists must serialise as [] rather than null for the UI.
	s.Controllers = nonNil(s.Controllers)
	s.Strips = nonNil(s.Strips)
	s.Groups = nonNil(s.Groups)
	s.Layers = nonNil(s.Layers)

	for i := range s.Controllers {
		c := &s.Controllers[i]
		if c.ID == "" {
			c.ID = NewID()
		}
		if !c.Protocol.Valid() {
			c.Protocol = output.SACN
		}
		if c.Protocol == output.SACN && c.StartUniverse < 1 {
			c.StartUniverse = 1
		}
		if c.MaxUniverses <= 0 {
			c.MaxUniverses = 4
		}
		if c.UniverseSize < 0 || c.UniverseSize > output.MaxSlots {
			c.UniverseSize = 0
		}
	}
	for i := range s.Strips {
		st := &s.Strips[i]
		if st.ID == "" {
			st.ID = NewID()
			if st.Brightness == 0 {
				st.Brightness = 1
			}
			if st.IdleMilliamps == 0 {
				st.IdleMilliamps = 1
			}
		}
		if st.Volts <= 0 {
			st.Volts = 5
		}
		if st.MilliampsPerChannel <= 0 {
			st.MilliampsPerChannel = 20
		}
		st.IdleMilliamps = max(st.IdleMilliamps, 0)
		st.MaxAmps = max(st.MaxAmps, 0)
		if st.ColorOrder == "" {
			st.ColorOrder = "GRB"
		}
		if st.WhiteMode != "off" {
			st.WhiteMode = "auto"
		}
		st.Brightness = clamp(st.Brightness, 0, 1)
		st.Pixels = max(st.Pixels, 0)
	}
	for i := range s.Groups {
		if s.Groups[i].ID == "" {
			s.Groups[i].ID = NewID()
		}
		s.Groups[i].Sections = nonNil(s.Groups[i].Sections)
	}
	for i := range s.Layers {
		l := &s.Layers[i]
		if l.ID == "" {
			l.ID = NewID()
			if l.Opacity == 0 {
				l.Opacity = 1
			}
		}
		if l.Params == nil {
			l.Params = map[string]any{}
		}
		if !slices.Contains(BlendModes, l.Blend) {
			l.Blend = "normal"
		}
		l.Opacity = clamp(l.Opacity, 0, 1)
	}
}

// Validate checks references and ranges. Effect names and parameters are
// validated by the engine, which owns the effect registry.
func (s *Show) Validate() error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	ctrls := map[string]bool{}
	for _, c := range s.Controllers {
		if ctrls[c.ID] {
			add("controller %q: duplicate id", c.ID)
		}
		ctrls[c.ID] = true
		if _, err := netip.ParseAddr(c.IP); err != nil {
			add("controller %q: invalid IP %q", c.Name, c.IP)
		}
		if err := c.Protocol.ValidateUniverse(c.StartUniverse); err != nil {
			add("controller %q: %v", c.Name, err)
		}
		if c.Port < 0 || c.Port > 65535 {
			add("controller %q: invalid port %d", c.Name, c.Port)
		}
	}

	strips := map[string]Strip{}
	for _, st := range s.Strips {
		if _, dup := strips[st.ID]; dup {
			add("strip %q: duplicate id", st.ID)
		}
		strips[st.ID] = st
		if st.ControllerID != "" && !ctrls[st.ControllerID] {
			add("strip %q: unknown controller %q", st.Name, st.ControllerID)
		}
		if _, err := pixel.ParseOrder(st.ColorOrder); err != nil {
			add("strip %q: %v", st.Name, err)
		}
	}

	groups := map[string]bool{}
	for _, g := range s.Groups {
		if groups[g.ID] {
			add("group %q: duplicate id", g.ID)
		}
		groups[g.ID] = true
		for i, sec := range g.Sections {
			st, ok := strips[sec.StripID]
			if !ok {
				add("group %q section %d: unknown strip %q", g.Name, i+1, sec.StripID)
				continue
			}
			if sec.Start < 0 || sec.End < sec.Start || sec.End >= st.Pixels {
				add("group %q section %d: range %d-%d outside strip %q (0-%d)",
					g.Name, i+1, sec.Start, sec.End, st.Name, st.Pixels-1)
			}
		}
	}

	layers := map[string]bool{}
	for _, l := range s.Layers {
		if layers[l.ID] {
			add("layer %q: duplicate id", l.ID)
		}
		layers[l.ID] = true
		if l.GroupID != "" && !groups[l.GroupID] {
			add("layer %q: unknown group %q", l.Name, l.GroupID)
		}
	}
	return errors.Join(errs...)
}

// ParseCID decodes a 32-hex-digit sACN CID.
func ParseCID(s string) ([16]byte, error) {
	var cid [16]byte
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 16 {
		return cid, fmt.Errorf("invalid CID %q", s)
	}
	copy(cid[:], b)
	return cid, nil
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
