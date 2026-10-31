package show

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"ledledled/internal/fsutil"
)

// Load reads a show from path. A missing file yields Default().
func Load(path string) (*Show, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	var s Show
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	s.Normalize()
	return &s, nil
}

// Save writes the show atomically so a power cut mid-write can't leave a
// corrupt config on the Pi's SD card.
func Save(path string, s *Show) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(path, b)
}

// Clone returns a deep copy via JSON round-trip (shows are small).
func (s *Show) Clone() *Show {
	b, _ := json.Marshal(s)
	var c Show
	_ = json.Unmarshal(b, &c)
	return &c
}

// Default is the starting show: three AirPixels on the pixel network, each
// with a 200 pixel WS2812 (GRB) strip, and a rainbow over everything.
func Default() *Show {
	s := &Show{
		Settings: Settings{FPS: 40, Brightness: 0.5, Gamma: 2.2},
	}
	for i := range 3 {
		c := Controller{
			ID:            fmt.Sprintf("ap%d", i+1),
			Name:          fmt.Sprintf("AirPixel %d", i+1),
			IP:            fmt.Sprintf("10.43.0.%d", 11+i),
			Protocol:      "sacn",
			StartUniverse: 1 + i*4,
			MaxUniverses:  4,
		}
		s.Controllers = append(s.Controllers, c)
		s.Strips = append(s.Strips, Strip{
			ID:           fmt.Sprintf("strip%d", i+1),
			Name:         fmt.Sprintf("Strip %d", i+1),
			ControllerID: c.ID,
			Pixels:       200,
			ColorOrder:   "GRB",
			Brightness:   1,
			// WS2812B 5V defaults; set MaxAmps to your PSU/injection limit.
			Volts:               5,
			MilliampsPerChannel: 20,
			IdleMilliamps:       1,
		})
	}
	s.Layers = []Layer{{
		ID:      "base",
		Name:    "Rainbow",
		Effect:  "rainbow",
		Params:  map[string]any{"speed": 0.2, "scale": 1.0},
		Opacity: 1,
		Blend:   "normal",
	}}
	s.Normalize()
	return s
}
