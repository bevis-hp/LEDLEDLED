package show

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPatchChainsStripsAndSplitsUniverses(t *testing.T) {
	s := &Show{
		Controllers: []Controller{{ID: "c", Name: "c", IP: "10.0.0.1", Protocol: "sacn", StartUniverse: 1}},
		Strips: []Strip{
			{ID: "a", ControllerID: "c", Pixels: 200, ColorOrder: "GRB"},
			{ID: "b", ControllerID: "c", Pixels: 10, ColorOrder: "GRB"},
		},
	}
	s.Normalize()
	p := s.Patch()[0]
	if p.Channels != 630 || p.UniverseSize != 510 || p.Universes != 2 {
		t.Fatalf("channels=%d usize=%d universes=%d", p.Channels, p.UniverseSize, p.Universes)
	}
	if p.Strips[1].ChannelOffset != 600 {
		t.Errorf("second strip offset = %d, want 600", p.Strips[1].ChannelOffset)
	}
	if len(p.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", p.Warnings)
	}
}

func TestPatchRGBWAndOverflowWarning(t *testing.T) {
	s := &Show{
		Controllers: []Controller{{ID: "c", Name: "c", IP: "10.0.0.1", Protocol: "sacn", StartUniverse: 1}},
		Strips:      []Strip{{ID: "a", ControllerID: "c", Pixels: 600, ColorOrder: "GRBW"}},
	}
	s.Normalize()
	p := s.Patch()[0]
	if p.UniverseSize != 512 || p.Universes != 5 || len(p.Warnings) != 1 {
		t.Fatalf("usize=%d universes=%d warnings=%v", p.UniverseSize, p.Universes, p.Warnings)
	}
}

func TestValidateCatchesBadRefs(t *testing.T) {
	s := Default()
	s.Groups = []Group{{ID: "g", Name: "g", Sections: []Section{{StripID: "strip1", Start: 0, End: 500}}}}
	s.Layers[0].GroupID = "missing"
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"outside strip", "unknown group"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}

func TestNormalizeEmptyListsAreNotNull(t *testing.T) {
	s := &Show{Groups: []Group{{Name: "g"}}}
	s.Normalize()
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), "null") {
		t.Errorf("normalised show contains null: %s", b)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "show.json")
	s := Default()
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Settings.CID != s.Settings.CID || len(got.Strips) != 3 {
		t.Errorf("round trip mismatch")
	}
}
