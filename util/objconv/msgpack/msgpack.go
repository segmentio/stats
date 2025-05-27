package msgpack

import "encoding/binary"

const (
	Nil   = 0xC0
	False = 0xC2
	True  = 0xC3

	Bin8  = 0xC4
	Bin16 = 0xC5
	Bin32 = 0xC6

	Ext8  = 0xC7
	Ext16 = 0xC8
	Ext32 = 0xC9

	Float32 = 0xCA
	Float64 = 0xCB

	Uint8  = 0xCC
	Uint16 = 0xCD
	Uint32 = 0xCE
	Uint64 = 0xCF

	Int8  = 0xD0
	Int16 = 0xD1
	Int32 = 0xD2
	Int64 = 0xD3

	Fixext1  = 0xD4
	Fixext2  = 0xD5
	Fixext4  = 0xD6
	Fixext8  = 0xD7
	Fixext16 = 0xD8

	Str8  = 0xD9
	Str16 = 0xDA
	Str32 = 0xDB

	Array16 = 0xDC
	Array32 = 0xDD

	Map16 = 0xDE
	Map32 = 0xDF

	FixmapMask = 0xF0
	FixmapTag  = 0x80

	FixarrayMask = 0xF0
	FixarrayTag  = 0x90

	FixstrMask = 0xE0
	FixstrTag  = 0xA0

	PositiveFixintMask = 0x80
	PositiveFixintTag  = 0x00

	NegativeFixintMask = 0xE0
	NegativeFixintTag  = 0xE0

	ExtTime = int8(-1)
)

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
