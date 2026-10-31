package show

// PaletteStop is a colour at a position 0..1.
type PaletteStop struct {
	Pos   float64 `json:"pos"`
	Color string  `json:"color"` // "#rrggbb"
}

// Palette is a named multi-stop gradient that palette-aware effects can
// use instead of their plain colour pickers.
type Palette struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Stops []PaletteStop `json:"stops"`
}

// BuiltinPalettes are always available and can't be edited (duplicate one
// to customise it). Their IDs are reserved.
var BuiltinPalettes = []Palette{
	{ID: "rainbow", Name: "Rainbow", Stops: []PaletteStop{
		{0, "#ff0000"}, {0.17, "#ffff00"}, {0.33, "#00ff00"}, {0.5, "#00ffff"},
		{0.67, "#0000ff"}, {0.83, "#ff00ff"}, {1, "#ff0000"}}},
	{ID: "fire", Name: "Fire", Stops: []PaletteStop{
		{0, "#000000"}, {0.35, "#b00000"}, {0.6, "#ff5a00"}, {0.85, "#ffc000"}, {1, "#fff4c0"}}},
	{ID: "lava", Name: "Lava", Stops: []PaletteStop{
		{0, "#000000"}, {0.3, "#5a0000"}, {0.55, "#ff2000"}, {0.75, "#ff8000"}, {1, "#ff0040"}}},
	{ID: "ocean", Name: "Ocean", Stops: []PaletteStop{
		{0, "#000020"}, {0.35, "#0030a0"}, {0.65, "#0090c0"}, {0.85, "#40e0d0"}, {1, "#e0ffff"}}},
	{ID: "forest", Name: "Forest", Stops: []PaletteStop{
		{0, "#002000"}, {0.4, "#106020"}, {0.7, "#50a020"}, {1, "#c0e040"}}},
	{ID: "sunset", Name: "Sunset", Stops: []PaletteStop{
		{0, "#200040"}, {0.3, "#a00060"}, {0.6, "#ff4020"}, {0.85, "#ffa020"}, {1, "#ffe080"}}},
	{ID: "party", Name: "Party", Stops: []PaletteStop{
		{0, "#5500ab"}, {0.25, "#b0007a"}, {0.5, "#ff4000"}, {0.75, "#ffd000"}, {1, "#5500ab"}}},
	{ID: "ice", Name: "Ice", Stops: []PaletteStop{
		{0, "#ffffff"}, {0.5, "#80c0ff"}, {1, "#0010a0"}}},
	{ID: "warm-white", Name: "Warm whites", Stops: []PaletteStop{
		{0, "#ff9a40"}, {0.5, "#ffc080"}, {1, "#fff0d8"}}},
}

// IsBuiltinPalette reports whether id belongs to a built-in palette.
func IsBuiltinPalette(id string) bool {
	for _, p := range BuiltinPalettes {
		if p.ID == id {
			return true
		}
	}
	return false
}

// AllPalettes returns the built-ins followed by the show's own palettes.
func (s *Show) AllPalettes() []Palette {
	return append(append([]Palette{}, BuiltinPalettes...), s.Palettes...)
}
