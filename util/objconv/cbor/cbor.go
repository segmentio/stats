package cbor

import (
	"encoding/binary"

	"github.com/segmentio/stats/v5/util/objconv/objutil"
)

const ( // major types
	majorType0 byte = iota
	majorType1
	majorType2
	majorType3
	majorType4
	majorType5
	majorType6
	majorType7
)

const ( // integer sizes
	iUint8 = 24 + iota
	iUint16
	iUint32
	iUint64
)

const ( // simple values
	svFalse = 20 + iota
	svTrue
	svNull
	svUndefined
	svExtension
	svFloat16
	svFloat32
	svFloat64
	svBreak = 31
)

const ( // tags
	tagDateTime  = 0
	tagTimestamp = 1
)

const (
	intMax   = uint64(objutil.IntMax)
	int64Max = uint64(objutil.Int64Max)
	noTag    = uint64(objutil.Uint64Max)
)

func majorByte(maj byte, val byte) byte {
	return (maj << 5) | val
}

func majorType(b byte) (maj byte, val byte) {
	const mask = byte(0xE0)
	return ((b & mask) >> 5), (b & ^mask)
}

func putUint16(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

func putUint32(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

func putUint64(b []byte, v uint64) {
	binary.BigEndian.PutUint64(b, v)
}

func getUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func getUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func getUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func align(n int, a int) int {
	if (n % a) == 0 {
		return n
	}
	return ((n / a) + 1) * a
}

func f16tof32bits(yy uint16) (d uint32) {
	y := uint32(yy)
	s := (y >> 15) & 0x01
	e := (y >> 10) & 0x1f
	m := y & 0x03ff

	if e == 0 {
		if m == 0 { // +/- 0
			return s << 31
		} else { // Denormalized number -- renormalize it
			for (m & 0x00000400) == 0 {
				m <<= 1
				e -= 1
			}
			e += 1
			const zz uint32 = 0x0400
			m &= ^zz
		}
	} else if e == 31 {
		if m == 0 { // Inf
			return (s << 31) | 0x7f800000
		} else { // NaN
			return (s << 31) | 0x7f800000 | (m << 13)
		}
	}

	e = e + (127 - 15)
	m = m << 13
	return (s << 31) | (e << 23) | m
}
