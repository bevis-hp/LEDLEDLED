// Package output encodes DMX universes as sACN (E1.31) or Art-Net packets and
// sends them over UDP. It is deliberately dependency-free: both protocols use
// small, fixed-layout packets, and owning the encoder makes it easy to control
// unicast destinations, sequence numbers and interface binding.
package output

import "fmt"

// Protocol identifies the wire protocol used for a destination.
type Protocol string

const (
	SACN   Protocol = "sacn"
	ArtNet Protocol = "artnet"
)

// Default UDP ports for each protocol.
const (
	SACNPort   = 5568
	ArtNetPort = 6454
)

// MaxSlots is the maximum number of DMX channels in a universe.
const MaxSlots = 512

// DefaultPort returns the standard UDP port for p.
func (p Protocol) DefaultPort() int {
	if p == ArtNet {
		return ArtNetPort
	}
	return SACNPort
}

// Valid reports whether p is a supported protocol.
func (p Protocol) Valid() bool { return p == SACN || p == ArtNet }

// ValidateUniverse checks a universe number is legal for the protocol.
// sACN universes are 1-63999. Art-Net uses a 15-bit port-address
// (Net<<8 | SubNet<<4 | Universe), i.e. 0-32767.
func (p Protocol) ValidateUniverse(u int) error {
	switch p {
	case SACN:
		if u < 1 || u > 63999 {
			return fmt.Errorf("sACN universe %d out of range 1-63999", u)
		}
	case ArtNet:
		if u < 0 || u > 32767 {
			return fmt.Errorf("Art-Net port-address %d out of range 0-32767", u)
		}
	default:
		return fmt.Errorf("unknown protocol %q", p)
	}
	return nil
}
