package output

import "encoding/binary"

// sACN / E1.31 data packet layout (ANSI E1.31-2016). Offsets are into the
// UDP payload. The packet is 126 bytes of headers followed by the DMX slots.
const (
	sacnHeaderLen       = 126
	sacnRootFlagsOff    = 16
	sacnCIDOff          = 22
	sacnFramingFlagsOff = 38
	sacnSourceNameOff   = 44
	sacnSourceNameLen   = 64
	sacnPriorityOff     = 108
	sacnSequenceOff     = 111
	sacnOptionsOff      = 112
	sacnUniverseOff     = 113
	sacnDMPFlagsOff     = 115
	sacnPropCountOff    = 123
	sacnStartCodeOff    = 125

	// Options bit flags.
	sacnOptStreamTerminated = 0x40
)

var acnPacketIdentifier = [12]byte{'A', 'S', 'C', '-', 'E', '1', '.', '1', '7', 0, 0, 0}

// SACNHeader holds the per-source fields of an E1.31 data packet.
type SACNHeader struct {
	CID        [16]byte // stable per-source UUID
	SourceName string   // human-readable, max 63 bytes
	Priority   byte     // 0-200, default 100
}

// BuildSACN writes an E1.31 data packet into buf and returns the slice that
// should be sent. buf is grown if needed so callers can reuse it per frame.
func BuildSACN(buf []byte, h *SACNHeader, universe uint16, seq byte, terminated bool, data []byte) []byte {
	if len(data) > MaxSlots {
		data = data[:MaxSlots]
	}
	n := sacnHeaderLen + len(data)
	if cap(buf) < n {
		buf = make([]byte, n)
	}
	buf = buf[:n]
	clear(buf[:sacnHeaderLen])

	// Root layer.
	binary.BigEndian.PutUint16(buf[0:], 0x0010) // preamble size
	binary.BigEndian.PutUint16(buf[2:], 0x0000) // postamble size
	copy(buf[4:], acnPacketIdentifier[:])
	putFlagsLen(buf[sacnRootFlagsOff:], n-sacnRootFlagsOff)
	binary.BigEndian.PutUint32(buf[18:], 0x00000004) // VECTOR_ROOT_E131_DATA
	copy(buf[sacnCIDOff:], h.CID[:])

	// Framing layer.
	putFlagsLen(buf[sacnFramingFlagsOff:], n-sacnFramingFlagsOff)
	binary.BigEndian.PutUint32(buf[40:], 0x00000002) // VECTOR_E131_DATA_PACKET
	name := h.SourceName
	if len(name) > sacnSourceNameLen-1 {
		name = name[:sacnSourceNameLen-1] // keep a NUL terminator
	}
	copy(buf[sacnSourceNameOff:], name)
	buf[sacnPriorityOff] = h.Priority
	// 109-110 sync address = 0 (no synchronisation)
	buf[sacnSequenceOff] = seq
	if terminated {
		buf[sacnOptionsOff] = sacnOptStreamTerminated
	}
	binary.BigEndian.PutUint16(buf[sacnUniverseOff:], universe)

	// DMP layer.
	putFlagsLen(buf[sacnDMPFlagsOff:], n-sacnDMPFlagsOff)
	buf[117] = 0x02                               // VECTOR_DMP_SET_PROPERTY
	buf[118] = 0xa1                               // address & data type
	binary.BigEndian.PutUint16(buf[119:], 0x0000) // first property address
	binary.BigEndian.PutUint16(buf[121:], 0x0001) // address increment
	binary.BigEndian.PutUint16(buf[sacnPropCountOff:], uint16(len(data)+1))
	buf[sacnStartCodeOff] = 0x00 // DMX512 null start code

	copy(buf[sacnHeaderLen:], data)
	return buf
}

// putFlagsLen writes an ACN PDU flags+length field (high nibble 0x7).
func putFlagsLen(b []byte, length int) {
	binary.BigEndian.PutUint16(b, 0x7000|uint16(length&0x0fff))
}
