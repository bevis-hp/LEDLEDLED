package engine

import (
	"bytes"
	"net"
	"testing"
	"time"

	"ledledled/internal/show"
)

// testShow returns a show with one 4-pixel GRB strip on a controller that
// sends to a local UDP listener.
func testShow(t *testing.T) (*show.Show, *net.UDPConn) {
	t.Helper()
	rx, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rx.Close() })
	port := rx.LocalAddr().(*net.UDPAddr).Port
	s := &show.Show{
		Settings:    show.Settings{FPS: 40, Brightness: 1, Gamma: 1},
		Controllers: []show.Controller{{ID: "c", Name: "c", IP: "127.0.0.1", Port: port, Protocol: "sacn", StartUniverse: 3}},
		Strips:      []show.Strip{{ID: "s", Name: "s", ControllerID: "c", Pixels: 4, ColorOrder: "GRB", Brightness: 1}},
	}
	return s, rx
}

func readUniverse(t *testing.T, rx *net.UDPConn) (universe int, data []byte) {
	t.Helper()
	buf := make([]byte, 1024)
	_ = rx.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err := rx.ReadFromUDP(buf)
	if err != nil {
		t.Fatal(err)
	}
	return int(buf[113])<<8 | int(buf[114]), buf[126:n]
}

func TestFrameSendsEncodedPixels(t *testing.T) {
	s, rx := testShow(t)
	s.Groups = []show.Group{{ID: "g", Name: "g", Sections: []show.Section{{StripID: "s", Start: 1, End: 2}}}}
	s.Layers = []show.Layer{
		{Name: "bg", Effect: "solid", Params: map[string]any{"color": "#0000ff"}, Opacity: 1},
		{Name: "mid", GroupID: "g", Effect: "solid", Params: map[string]any{"color": "#ff0000"}, Opacity: 1},
	}
	e, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	e.frame(time.Now(), true)

	u, data := readUniverse(t, rx)
	want := []byte{0, 0, 255, 0, 255, 0, 0, 255, 0, 0, 0, 255} // GRB: blue, red, red, blue
	if u != 3 || !bytes.Equal(data, want) {
		t.Fatalf("universe %d data %v, want 3 %v", u, data, want)
	}
}

func TestReverseStripAndBlackout(t *testing.T) {
	s, rx := testShow(t)
	s.Strips[0].ColorOrder = "RGB"
	s.Strips[0].Reverse = true
	s.Layers = []show.Layer{{Name: "g", Effect: "gradient",
		Params: map[string]any{"colorA": "#ff0000", "colorB": "#0000ff", "speed": 0.0}, Opacity: 1}}
	e, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	e.frame(time.Now(), false)
	_, data := readUniverse(t, rx)
	// Logical pixel 0 is red; reversed, so it is the last pixel on the wire.
	if data[0] != 0 || data[2] != 255 || data[9] != 255 || data[11] != 0 {
		t.Fatalf("reversed gradient wire data = %v", data)
	}

	e.SetBlackout(true)
	e.frame(time.Now(), false)
	_, data = readUniverse(t, rx)
	if !bytes.Equal(data, make([]byte, 12)) {
		t.Fatalf("blackout data = %v", data)
	}
}

func TestApplyRejectsInvalidShowAndKeepsOld(t *testing.T) {
	s, _ := testShow(t)
	e, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	bad := s.Clone()
	bad.Layers = []show.Layer{{Name: "x", Effect: "does-not-exist"}}
	if err := e.Apply(bad); err == nil {
		t.Fatal("expected error")
	}
	if len(e.Show().Layers) != 0 {
		t.Error("invalid show was applied")
	}
}

func TestPreviewFrameFormat(t *testing.T) {
	s, _ := testShow(t)
	s.Layers = []show.Layer{{Name: "w", Effect: "solid", Params: map[string]any{"color": "#ffffff"}, Opacity: 1}}
	e, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	ch, cancel := e.Subscribe()
	defer cancel()
	e.frame(time.Now(), true)
	f := <-ch
	if f[0] != 1 || f[1] != 1 || f[3] != 4 || len(f) != 3+2+12 || f[5] != 255 {
		t.Fatalf("preview frame = %v", f)
	}
}

func TestApplyKeepsTempoButLoadRestoresIt(t *testing.T) {
	s, _ := testShow(t)
	s.Tempo.BPM = 100
	e, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	e.SetBPM(140)
	if err := e.Apply(s); err != nil { // live edit carrying the old 100 BPM
		t.Fatal(err)
	}
	if got := e.Show().Tempo.BPM; got != 140 {
		t.Errorf("after Apply bpm = %v, want 140 (unchanged)", got)
	}
	if err := e.Load(s); err != nil {
		t.Fatal(err)
	}
	if got := e.Show().Tempo.BPM; got != 100 {
		t.Errorf("after Load bpm = %v, want 100", got)
	}
}

func TestPowerLimitScalesWireData(t *testing.T) {
	s, rx := testShow(t)
	s.Strips[0].MilliampsPerChannel = 20
	s.Strips[0].IdleMilliamps = 0
	s.Strips[0].MaxAmps = 0.12 // full white would be 4 * 60 mA = 0.24 A
	s.Layers = []show.Layer{{Name: "w", Effect: "solid", Params: map[string]any{"color": "#ffffff"}, Opacity: 1}}
	e, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	e.frame(time.Now(), false)
	_, data := readUniverse(t, rx)
	for i, b := range data {
		if b > 128 || b < 120 {
			t.Fatalf("byte %d = %d, want ~127 (half power)", i, b)
		}
	}
	e.mu.Lock()
	rollPowerWindow(e.plan)
	e.mu.Unlock()
	st := e.Status().Strips[0]
	if !st.Limited || st.Amps > 0.12 || st.RequestedAmps < 0.239 || st.Watts > 0.6 {
		t.Errorf("status = %+v", st)
	}
}
