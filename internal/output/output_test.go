package output

import (
	"bytes"
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestBuildSACNLayout(t *testing.T) {
	h := &SACNHeader{SourceName: "test-src", Priority: 100}
	for i := range h.CID {
		h.CID[i] = byte(i)
	}
	data := []byte{10, 20, 30}
	pkt := BuildSACN(nil, h, 7, 42, false, data)

	if len(pkt) != 126+3 {
		t.Fatalf("len = %d, want 129", len(pkt))
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"preamble", binary.BigEndian.Uint16(pkt[0:]), uint16(0x0010)},
		{"acn id", string(pkt[4:13]), "ASC-E1.17"},
		{"root flags/len", binary.BigEndian.Uint16(pkt[16:]), uint16(0x7000 | (129 - 16))},
		{"root vector", binary.BigEndian.Uint32(pkt[18:]), uint32(4)},
		{"cid[15]", pkt[37], byte(15)},
		{"framing flags/len", binary.BigEndian.Uint16(pkt[38:]), uint16(0x7000 | (129 - 38))},
		{"framing vector", binary.BigEndian.Uint32(pkt[40:]), uint32(2)},
		{"source name", string(bytes.TrimRight(pkt[44:108], "\x00")), "test-src"},
		{"priority", pkt[108], byte(100)},
		{"sequence", pkt[111], byte(42)},
		{"options", pkt[112], byte(0)},
		{"universe", binary.BigEndian.Uint16(pkt[113:]), uint16(7)},
		{"dmp flags/len", binary.BigEndian.Uint16(pkt[115:]), uint16(0x7000 | (129 - 115))},
		{"dmp vector", pkt[117], byte(2)},
		{"addr type", pkt[118], byte(0xa1)},
		{"addr inc", binary.BigEndian.Uint16(pkt[121:]), uint16(1)},
		{"prop count", binary.BigEndian.Uint16(pkt[123:]), uint16(4)},
		{"start code", pkt[125], byte(0)},
		{"data", string(pkt[126:]), string(data)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}

	term := BuildSACN(nil, h, 7, 1, true, data)
	if term[112] != 0x40 {
		t.Errorf("terminated option = %#x, want 0x40", term[112])
	}
}

func TestBuildSACNFullUniverseLength(t *testing.T) {
	pkt := BuildSACN(nil, &SACNHeader{}, 1, 0, false, make([]byte, 512))
	if len(pkt) != 638 {
		t.Fatalf("full universe packet len = %d, want 638", len(pkt))
	}
}

func TestBuildArtDmxLayout(t *testing.T) {
	pa := ArtNetPortAddress(1, 2, 3) // 0x0123
	pkt := BuildArtDmx(nil, pa, 9, []byte{1, 2, 3})

	if len(pkt) != 18+4 { // odd length padded to even
		t.Fatalf("len = %d, want 22", len(pkt))
	}
	if string(pkt[:8]) != "Art-Net\x00" {
		t.Errorf("id = %q", pkt[:8])
	}
	if pkt[8] != 0x00 || pkt[9] != 0x50 {
		t.Errorf("opcode bytes = %#x %#x, want 0x00 0x50", pkt[8], pkt[9])
	}
	if binary.BigEndian.Uint16(pkt[10:]) != 14 {
		t.Errorf("protver wrong")
	}
	if pkt[12] != 9 || pkt[14] != 0x23 || pkt[15] != 0x01 {
		t.Errorf("seq/subuni/net = %d %#x %#x", pkt[12], pkt[14], pkt[15])
	}
	if binary.BigEndian.Uint16(pkt[16:]) != 4 {
		t.Errorf("length = %d, want 4", binary.BigEndian.Uint16(pkt[16:]))
	}
	if !bytes.Equal(pkt[18:], []byte{1, 2, 3, 0}) {
		t.Errorf("data = %v", pkt[18:])
	}
}

// TestTransmitterLoopback sends real UDP packets to localhost and checks
// they arrive intact with incrementing sequence numbers.
func TestTransmitterLoopback(t *testing.T) {
	rx, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer rx.Close()
	dst := netip.MustParseAddrPort(rx.LocalAddr().String())

	tx, err := NewTransmitter(Options{SourceName: "loop"})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()

	buf := make([]byte, 1024)
	for i := 1; i <= 2; i++ {
		if err := tx.Send(SACN, dst, 5, []byte{byte(i)}); err != nil {
			t.Fatal(err)
		}
		_ = rx.SetReadDeadline(time.Now().Add(time.Second))
		n, _, err := rx.ReadFromUDP(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != 127 || buf[111] != byte(i) || buf[126] != byte(i) {
			t.Errorf("packet %d: n=%d seq=%d data=%d", i, n, buf[111], buf[126])
		}
	}
}
