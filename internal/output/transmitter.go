package output

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/netip"
	"sync"
)

// Transmitter owns a single UDP socket and sends sACN/Art-Net packets to
// unicast destinations. It tracks a sequence number per (protocol,
// destination, universe) as both protocols expect.
//
// Binding to a specific local IP lets the Pi force traffic out of the
// interface that the pixel controllers are on (e.g. the USB Wi-Fi stick).
type Transmitter struct {
	mu     sync.Mutex
	conn   *net.UDPConn
	header SACNHeader
	seq    map[seqKey]byte
	buf    []byte
}

type seqKey struct {
	proto    Protocol
	dst      netip.AddrPort
	universe uint16
}

// Options configures a Transmitter.
type Options struct {
	BindIP     string   // local IP to send from; empty = any interface
	SourceName string   // sACN source name
	Priority   byte     // sACN priority (0 = default 100)
	CID        [16]byte // sACN component ID; zero = random
}

// NewTransmitter opens the UDP socket.
func NewTransmitter(o Options) (*Transmitter, error) {
	laddr := &net.UDPAddr{}
	if o.BindIP != "" {
		ip := net.ParseIP(o.BindIP)
		if ip == nil {
			return nil, fmt.Errorf("invalid bind IP %q", o.BindIP)
		}
		laddr.IP = ip
	}
	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, fmt.Errorf("open UDP socket: %w", err)
	}
	if o.Priority == 0 {
		o.Priority = 100
	}
	if o.CID == ([16]byte{}) {
		o.CID = NewCID()
	}
	return &Transmitter{
		conn:   conn,
		header: SACNHeader{CID: o.CID, SourceName: o.SourceName, Priority: o.Priority},
		seq:    make(map[seqKey]byte),
		buf:    make([]byte, 0, sacnHeaderLen+MaxSlots),
	}, nil
}

// Send transmits one universe of DMX data to dst.
func (t *Transmitter) Send(proto Protocol, dst netip.AddrPort, universe uint16, data []byte) error {
	return t.send(proto, dst, universe, data, false)
}

// Terminate tells sACN receivers the stream is ending so they release
// control immediately instead of waiting for the 2.5 s timeout. The spec
// asks for three packets. Art-Net has no equivalent, so it is a no-op there.
func (t *Transmitter) Terminate(proto Protocol, dst netip.AddrPort, universe uint16, data []byte) error {
	if proto != SACN {
		return nil
	}
	for range 3 {
		if err := t.send(proto, dst, universe, data, true); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transmitter) send(proto Protocol, dst netip.AddrPort, universe uint16, data []byte, terminated bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	k := seqKey{proto, dst, universe}
	seq := t.seq[k] + 1
	if proto == ArtNet && seq == 0 {
		seq = 1 // Art-Net reserves 0 for "sequencing disabled"
	}
	t.seq[k] = seq

	var pkt []byte
	switch proto {
	case SACN:
		pkt = BuildSACN(t.buf, &t.header, universe, seq, terminated, data)
	case ArtNet:
		pkt = BuildArtDmx(t.buf, universe, seq, data)
	default:
		return fmt.Errorf("unknown protocol %q", proto)
	}
	t.buf = pkt[:0]
	_, err := t.conn.WriteToUDPAddrPort(pkt, dst)
	return err
}

// Close releases the socket.
func (t *Transmitter) Close() error { return t.conn.Close() }

// NewCID returns a random RFC 4122 v4 UUID suitable for an sACN CID.
func NewCID() [16]byte {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return b
}
