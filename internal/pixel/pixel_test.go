package pixel

import (
	"bytes"
	"testing"
)

func TestParseHexRoundTrip(t *testing.T) {
	c, err := ParseHex("#ff8000")
	if err != nil {
		t.Fatal(err)
	}
	if c.Hex() != "#ff8000" {
		t.Errorf("round trip = %s", c.Hex())
	}
	if _, err := ParseHex("nope"); err == nil {
		t.Error("expected error for bad hex")
	}
}

func TestParseOrder(t *testing.T) {
	for _, s := range SupportedOrders {
		if _, err := ParseOrder(s); err != nil {
			t.Errorf("%s: %v", s, err)
		}
	}
	for _, bad := range []string{"RG", "RRB", "RGBX", "RGW", "RGBWW"} {
		if _, err := ParseOrder(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
}

func TestEncodeGRB(t *testing.T) {
	dst := make([]byte, 6)
	MustOrder("GRB").Encode(dst, []Color{{1, 0, 0}, {0, 0, 1}}, EncodeOptions{Brightness: 1})
	if !bytes.Equal(dst, []byte{0, 255, 0, 0, 0, 255}) {
		t.Errorf("GRB = %v", dst)
	}
}

func TestEncodeRGBWExtractsWhite(t *testing.T) {
	dst := make([]byte, 4)
	MustOrder("RGBW").Encode(dst, []Color{{1, 1, 0.5}}, EncodeOptions{Brightness: 1, ExtractWhite: true})
	// min component (0.5) moves into W.
	if !bytes.Equal(dst, []byte{128, 128, 0, 128}) {
		t.Errorf("RGBW = %v", dst)
	}
}

func TestEncodeBrightnessAndGamma(t *testing.T) {
	dst := make([]byte, 3)
	MustOrder("RGB").Encode(dst, []Color{{1, 1, 1}}, EncodeOptions{Brightness: 0.5, Gamma: NewGamma(2)})
	// 0.5^2 = 0.25 -> 64
	if dst[0] != 64 {
		t.Errorf("gamma 2 @ 0.5 = %d, want 64", dst[0])
	}
}
