// Package show defines the persisted show configuration: pixel controllers
// (e.g. AirPixels), the strips attached to them, groups of strip sections,
// and the effect layers that paint those groups.
//
// Everything is plain JSON-serialisable data so the web UI can edit it
// directly. The engine turns a Show into a render plan.
package show

import (
	"crypto/rand"
	"encoding/hex"
	"ledledled/internal/output"
)

// Show is the complete, persisted state of the installation.
type Show struct {
	Settings    Settings     `json:"settings"`
	Tempo       Tempo        `json:"tempo"`
	Controllers []Controller `json:"controllers"`
	Strips      []Strip      `json:"strips"`
	Groups      []Group      `json:"groups"`
	Layers      []Layer      `json:"layers"`
	Timeline    Timeline     `json:"timeline"`
}

// Timeline is a single looping keyframe timeline measured in beats, so it
// follows tap tempo / detected BPM. Tracks override layer parameters.
type Timeline struct {
	LengthBeats float64 `json:"lengthBeats"` // loop length, default 16 (4 bars of 4/4)
	OneShot     bool    `json:"oneShot"`     // stop at the end instead of looping
	Tracks      []Track `json:"tracks"`
}

// Track animates one parameter of one layer.
type Track struct {
	LayerID string     `json:"layerId"`
	Param   string     `json:"param"` // effect param name, or "opacity"
	Keys    []Keyframe `json:"keys"`
}

// Keyframe is a value at a beat. Ease controls the interpolation from this
// key to the next: linear, smooth, in, out or step.
type Keyframe struct {
	Beat  float64 `json:"beat"`
	Value any     `json:"value"` // number, or "#rrggbb" for colours
	Ease  string  `json:"ease"`
}

// Tempo is the beat clock's saved state. The live BPM (tap tempo, audio
// detection) is runtime state in the engine; it is written back here when
// the show is saved and restored when a show is loaded.
type Tempo struct {
	BPM float64 `json:"bpm"`
}

// Settings are global output settings.
type Settings struct {
	FPS        int     `json:"fps"`        // render/output rate, default 40
	Brightness float64 `json:"brightness"` // master brightness 0..1
	Gamma      float64 `json:"gamma"`      // 1 = linear, ~2.2 looks natural on LEDs
	BindIP     string  `json:"bindIp"`     // local IP to send from ("" = any)
	SourceName string  `json:"sourceName"` // sACN source name
	Priority   int     `json:"priority"`   // sACN priority 0-200
	CID        string  `json:"cid"`        // sACN component ID (hex), generated once
}

// Controller is a network pixel driver such as an AirPixel Mini II. It
// receives consecutive universes starting at StartUniverse and outputs them
// as one continuous pixel stream.
type Controller struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	IP            string          `json:"ip"`
	Port          int             `json:"port,omitempty"` // 0 = protocol default
	Protocol      output.Protocol `json:"protocol"`
	StartUniverse int             `json:"startUniverse"`
	// UniverseSize is the number of channels sent per universe. It must
	// match the controller's own "universe size" setting. 0 = auto: 510 for
	// RGB strips (170 px) and 512 for RGBW strips (128 px).
	UniverseSize int  `json:"universeSize"`
	MaxUniverses int  `json:"maxUniverses"` // AirPixel Mini II = 4
	Disabled     bool `json:"disabled"`
}

// Strip is a run of pixels on a controller. Strips on the same controller
// are chained in the order they appear in Show.Strips.
type Strip struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ControllerID string  `json:"controllerId"`
	Pixels       int     `json:"pixels"`
	ColorOrder   string  `json:"colorOrder"` // e.g. GRB (WS2812), GRBW (SK6812)
	Brightness   float64 `json:"brightness"` // per-strip trim 0..1
	Reverse      bool    `json:"reverse"`    // physical pixel 0 is at the far end
	WhiteMode    string  `json:"whiteMode"`  // RGBW only: "auto" or "off"

	// Power model and limit. See pixel.PowerModel for typical values.
	Volts               float64 `json:"volts"`               // 5 or 12 (used for watts)
	MilliampsPerChannel float64 `json:"milliampsPerChannel"` // one channel at full, per pixel
	IdleMilliamps       float64 `json:"idleMilliamps"`       // quiescent draw per pixel
	MaxAmps             float64 `json:"maxAmps"`             // 0 = no limit
}

// Section is an inclusive pixel range on one strip. Reverse flips the
// direction effects travel through it.
type Section struct {
	StripID string `json:"stripId"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Reverse bool   `json:"reverse"`
}

// Len is the number of pixels in the section.
func (s Section) Len() int { return s.End - s.Start + 1 }

// Group is an ordered list of sections that effects treat as one continuous
// line of pixels, even when they span several strips.
type Group struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Sections []Section `json:"sections"`
}

// Layer applies an effect to a group. Layers are composited in order, the
// first layer being the bottom.
type Layer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	GroupID string `json:"groupId"` // "" = every pixel of every strip
	Effect  string `json:"effect"`
	// Params are effect parameters keyed by name. Numbers are float64,
	// colours are "#rrggbb" strings. Keeping them as a generic map means a
	// future keyframe system can animate any parameter by name.
	Params  map[string]any `json:"params"`
	Opacity float64        `json:"opacity"` // 0..1
	Blend   string         `json:"blend"`   // see BlendModes
	Muted   bool           `json:"muted"`
	// TempoSync feeds the effect beats instead of seconds, so speeds and
	// rates become "per beat" and follow tap tempo / detected BPM.
	TempoSync bool `json:"tempoSync"`
}

// BlendModes are the supported layer blend modes.
var BlendModes = []string{"normal", "add", "multiply", "screen", "max"}

// NewID returns a short random identifier.
func NewID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
