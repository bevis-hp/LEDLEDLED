package output

import "encoding/binary"

// Art-Net ArtDmx packet layout (Art-Net 4 spec, compatible with II/III).
const (
	artHeaderLen  = 18
	artOpDmx      = 0x5000
	artProtVer    = 14
	artSeqOff     = 12
	artPhysOff    = 13
	artSubUniOff  = 14
	artNetOff     = 15
	artLengthOff  = 16
	artMinDataLen = 2
)

var artNetID = [8]byte{'A', 'r', 't', '-', 'N', 'e', 't', 0}

// BuildArtDmx writes an ArtDmx packet into buf and returns the slice to send.
// portAddress is the 15-bit Art-Net port-address (Net<<8 | SubNet<<4 | Uni).
// seq should cycle 1-255; 0 tells receivers to ignore ordering.
func BuildArtDmx(buf []byte, portAddress uint16, seq byte, data []byte) []byte {
	if len(data) > MaxSlots {
		data = data[:MaxSlots]
	}
	// The spec requires an even data length of at least 2.
	dataLen := len(data)
	if dataLen < artMinDataLen {
		dataLen = artMinDataLen
	}
	if dataLen%2 == 1 {
		dataLen++
	}
	n := artHeaderLen + dataLen
	if cap(buf) < n {
		buf = make([]byte, n)
	}
	buf = buf[:n]

	copy(buf[0:], artNetID[:])
	binary.LittleEndian.PutUint16(buf[8:], artOpDmx) // OpCode is little-endian
	binary.BigEndian.PutUint16(buf[10:], artProtVer)
	buf[artSeqOff] = seq
	buf[artPhysOff] = 0
	buf[artSubUniOff] = byte(portAddress & 0xff)
	buf[artNetOff] = byte((portAddress >> 8) & 0x7f)
	binary.BigEndian.PutUint16(buf[artLengthOff:], uint16(dataLen))

	copy(buf[artHeaderLen:], data)
	clear(buf[artHeaderLen+len(data):]) // zero any padding
	return buf
}

// ArtNetPortAddress combines Net (0-127), SubNet (0-15) and Universe (0-15).
func ArtNetPortAddress(net, subnet, universe int) uint16 {
	return uint16((net&0x7f)<<8 | (subnet&0x0f)<<4 | (universe & 0x0f))
}
