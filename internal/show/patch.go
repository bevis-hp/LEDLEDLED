package show

import (
	"fmt"

	"ledledled/internal/pixel"
)

// StripPatch places one strip within its controller's channel stream.
type StripPatch struct {
	StripIndex    int // index into Show.Strips
	ChannelOffset int // first channel (0-based) in the controller stream
	Order         pixel.Order
}

// ControllerPatch describes how a controller's pixels map onto universes.
type ControllerPatch struct {
	ControllerIndex int
	Strips          []StripPatch
	Channels        int // total channels used by all strips
	UniverseSize    int // channels per universe actually sent
	Universes       int // number of universes needed
	Warnings        []string
}

// Patch computes the channel/universe layout for every controller. Strips
// on a controller are chained in Show.Strips order. Show must be valid.
func (s *Show) Patch() []ControllerPatch {
	out := make([]ControllerPatch, len(s.Controllers))
	for ci, c := range s.Controllers {
		cp := ControllerPatch{ControllerIndex: ci}
		hasWhite := false
		for si, st := range s.Strips {
			if st.ControllerID != c.ID {
				continue
			}
			order, err := pixel.ParseOrder(st.ColorOrder)
			if err != nil {
				continue // Validate reports this
			}
			hasWhite = hasWhite || order.HasWhite()
			cp.Strips = append(cp.Strips, StripPatch{StripIndex: si, ChannelOffset: cp.Channels, Order: order})
			cp.Channels += st.Pixels * order.Channels()
		}

		cp.UniverseSize = c.UniverseSize
		if cp.UniverseSize == 0 {
			// Keep whole pixels in each universe.
			cp.UniverseSize = 510
			if hasWhite {
				cp.UniverseSize = 512
			}
		}
		cp.Universes = (cp.Channels + cp.UniverseSize - 1) / cp.UniverseSize
		if c.MaxUniverses > 0 && cp.Universes > c.MaxUniverses {
			cp.Warnings = append(cp.Warnings, fmt.Sprintf(
				"needs %d universes but %s accepts %d; pixels past %d channels will not light",
				cp.Universes, c.Name, c.MaxUniverses, c.MaxUniverses*cp.UniverseSize))
		}
		out[ci] = cp
	}
	return out
}

// StripController returns the controller index a strip belongs to, or -1.
func (s *Show) StripController(stripIdx int) int {
	id := s.Strips[stripIdx].ControllerID
	for i, c := range s.Controllers {
		if c.ID == id {
			return i
		}
	}
	return -1
}
