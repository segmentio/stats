package msgpack

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/segmentio/stats/v5/util/objconv"
)

type Parser struct {
	r io.Reader // reader to load bytes from
	i int       // offset of the first unread byte in b
	j int       // offset + 1 of the last unread byte in b
	s []byte    // string buffer
	b [240]byte // read buffer
}

func NewParser(r io.Reader) *Parser {
	return &Parser{r: r}
}

func (p *Parser) Reset(r io.Reader) {
	p.r = r
	p.i = 0
	p.j = 0
}

func (p *Parser) Buffered() io.Reader {
	return bytes.NewReader(p.b[p.i:p.j])
}

func (p *Parser) ParseType() (objconv.Type, error) {
	b, err := p.peek(1)
	if err != nil {
		return objconv.Unknown, err
	}

	tag := b[0]

	switch {
	case (tag & PositiveFixintMask) == PositiveFixintTag:
		return objconv.Int, nil

	case (tag & NegativeFixintMask) == NegativeFixintTag:
		return objconv.Int, nil

	case (tag & FixstrMask) == FixstrTag:
		return objconv.String, nil

	case (tag & FixarrayMask) == FixarrayTag:
		return objconv.Array, nil

	case (tag & FixmapMask) == FixmapTag:
		return objconv.Map, nil
	}

	switch tag {
	case Nil:
		return objconv.Nil, nil

	case False, True:
		return objconv.Bool, nil

	case Int8, Int16, Int32, Int64:
		return objconv.Int, nil

	case Uint8, Uint16, Uint32, Uint64:
		return objconv.Uint, nil

	case Float32, Float64:
		return objconv.Float, nil

	case Str8, Str16, Str32:
		return objconv.String, nil

	case Bin8, Bin16, Bin32:
		return objconv.Bytes, nil

	case Array16, Array32:
		return objconv.Array, nil

	case Map16, Map32:
		return objconv.Map, nil

	case Fixext1, Fixext2, Fixext4, Fixext8, Fixext16:
		if b, err = p.peek(2); err != nil {
			return objconv.Unknown, err
		}

		switch tag = b[1]; int8(tag) {
		case ExtTime:
			return objconv.Time, nil

		default:
			return objconv.Unknown, fmt.Errorf("objconv/msgpack: unsupported extension '%d'", tag)
		}

	case Ext8, Ext16, Ext32: // continue after the switch
	default:
		return objconv.Unknown, fmt.Errorf("objconv/msgpack: unknown tag '%#x'", tag)
	}

	switch tag {
	case Ext8:
		b, err = p.peek(3)
	case Ext16:
		b, err = p.peek(4)
	default:
		b, err = p.peek(6)
	}

	if err != nil {
		return objconv.Unknown, err
	}

	switch tag = b[len(b)-1]; int8(tag) {
	case ExtTime:
		return objconv.Time, nil
	}

	return objconv.Unknown, fmt.Errorf("objconv/msgpack: unknown extension '%d'", tag)
}

func (p *Parser) ParseNil() (err error) {
	p.i++
	return
}

func (p *Parser) ParseBool() (v bool, err error) {
	v = p.b[p.i] == True
	p.i++
	return
}

func (p *Parser) ParseInt() (v int64, err error) {
	tag := p.b[p.i]
	p.i++

	var b []byte
	var n int

	switch {
	case (tag & PositiveFixintMask) == PositiveFixintTag:
		return int64(int8(tag)), nil
	case (tag & NegativeFixintMask) == NegativeFixintTag:
		return int64(int8(tag)), nil
	}

	switch tag {
	case Int8:
		n = 1
	case Int16:
		n = 2
	case Int32:
		n = 4
	default:
		n = 8
	}

	if b, err = p.peek(n); err != nil {
		return
	}

	switch n {
	case 1:
		v = int64(int8(b[0]))
	case 2:
		v = int64(int16(getUint16(b)))
	case 4:
		v = int64(int32(getUint32(b)))
	default:
		v = int64(getUint64(b))
	}

	p.i += n
	return
}

func (p *Parser) ParseUint() (v uint64, err error) {
	tag := p.b[p.i]
	p.i++

	var b []byte
	var n int

	switch tag {
	case Uint8:
		n = 1
	case Uint16:
		n = 2
	case Uint32:
		n = 4
	default:
		n = 8
	}

	if b, err = p.peek(n); err != nil {
		return
	}

	switch n {
	case 1:
		v = uint64(b[0])
	case 2:
		v = uint64(getUint16(b))
	case 4:
		v = uint64(getUint32(b))
	default:
		v = getUint64(b)
	}

	p.i += n
	return
}

func (p *Parser) ParseFloat() (v float64, err error) {
	tag := p.b[p.i]
	p.i++

	var b []byte

	switch tag {
	case Float32:
		b, err = p.peek(4)
	default:
		b, err = p.peek(8)
	}

	if err != nil {
		return
	}

	switch tag {
	case Float32:
		v = float64(math.Float32frombits(getUint32(b)))
		p.i += 4
	default:
		v = math.Float64frombits(getUint64(b))
		p.i += 8
	}

	return
}

func (p *Parser) ParseString() (v []byte, err error) {
	tag := p.b[p.i]
	p.i++

	n := 0

	if (tag & FixstrMask) == FixstrTag {
		n = int(tag & ^byte(FixstrMask))
	} else {
		var b []byte

		switch tag {
		case Str8:
			n = 1
		case Str16:
			n = 2
		default:
			n = 4
		}

		if b, err = p.peek(n); err != nil {
			return
		}
		p.i += n

		switch n {
		case 1:
			n = int(b[0])
		case 2:
			n = int(getUint16(b))
		default:
			n = int(getUint32(b))
		}
	}

	return p.read(n)
}

func (p *Parser) ParseBytes() (v []byte, err error) {
	tag := p.b[p.i]
	p.i++

	var b []byte
	var n int

	switch tag {
	case Bin8:
		n = 1
	case Bin16:
		n = 2
	default:
		n = 4
	}

	if b, err = p.peek(n); err != nil {
		return
	}
	p.i += n

	switch n {
	case 1:
		n = int(b[0])
	case 2:
		n = int(getUint16(b))
	default:
		n = int(getUint32(b))
	}

	return p.read(n)
}

func (p *Parser) ParseTime() (v time.Time, err error) {
	tag := p.b[p.i]
	p.i++

	var b []byte
	var s int64
	var ns int64

	switch tag {
	case Fixext4: // 32-bit unsigned timestamp
		p.i++
		if b, err = p.peek(4); err != nil {
			return
		}
		p.i += 4
		s = int64(getUint32(b))

	case Fixext8: // 30-bit unsigned nanoseconds + 34-bit unsigned timestamp
		p.i++
		if b, err = p.peek(8); err != nil {
			return
		}
		p.i += 8
		ts := getUint64(b)
		s, ns = int64(ts&0x3FFFFFFFF), int64(ts>>34)

	case Ext8: // 32-bit unsigned nanoseconds + 64 bits signed timestamp
		if b, err = p.peek(1); err != nil {
			return
		}
		if b[0] != 12 {
			err = fmt.Errorf("objconv/msgpack: invalid timestamp length, expected 12 but found %d", int(b[0]))
			return
		}
		p.i += 2 // skip the extension length and type
		if b, err = p.peek(12); err != nil {
			return
		}
		p.i += 12
		s, ns = int64(getUint64(b[4:])), int64(getUint32(b))

	default:
		err = fmt.Errorf("objconv/msgpack: invalid extension tag found while decoding a timestamp '%d'", tag)
		return
	}

	v = time.Unix(s, ns).In(time.UTC)
	return
}

func (p *Parser) ParseDuration() (v time.Duration, err error) {
	panic("objconv/msgpack: ParseDuration should never be called because MessagePack has no duration type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseError() (v error, err error) {
	panic("objconv/msgpack: ParseError should never be called because MessagePack has no error type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseArrayBegin() (n int, err error) {
	tag := p.b[p.i]
	p.i++

	if (tag & FixarrayMask) == FixarrayTag {
		n = int(int8(tag & ^byte(FixarrayMask)))
	} else {
		var b []byte

		switch tag {
		case Array16:
			b, err = p.peek(2)
		default:
			b, err = p.peek(4)
		}

		if err != nil {
			return
		}

		switch len(b) {
		case 2:
			n = int(getUint16(b))
		default:
			n = int(getUint32(b))
		}

		p.i += len(b)
	}

	return
}

func (p *Parser) ParseArrayEnd(n int) (err error) {
	return
}

func (p *Parser) ParseArrayNext(n int) (err error) {
	return
}

func (p *Parser) ParseMapBegin() (n int, err error) {
	tag := p.b[p.i]
	p.i++

	if (tag & FixmapMask) == FixmapTag {
		n = int(int8(tag & ^byte(FixmapMask)))
	} else {
		var b []byte

		switch tag {
		case Map16:
			b, err = p.peek(2)
		default:
			b, err = p.peek(4)
		}

		if err != nil {
			return
		}

		switch len(b) {
		case 2:
			n = int(getUint16(b))
		default:
			n = int(getUint32(b))
		}

		p.i += len(b)
	}

	return
}

func (p *Parser) ParseMapEnd(n int) (err error) {
	return
}

func (p *Parser) ParseMapValue(n int) (err error) {
	return
}

func (p *Parser) ParseMapNext(n int) (err error) {
	return
}

func (p *Parser) read(n int) (b []byte, err error) {
	if n <= (p.j - p.i) { // check if the string is already buffered
		b = p.b[p.i : p.i+n]
		p.i += n
		return
	}

	if n <= len(p.b) { // check if the string can be loaded in the read buffer
		if b, err = p.peek(n); err != nil {
			return
		}
		p.i += n
		return
	}

	if cap(p.s) < n {
		p.s = make([]byte, n, align(n, 1024))
	} else {
		p.s = p.s[:n]
	}

	copy(p.s, p.b[p.i:p.j])
	n = p.j - p.i
	p.i = 0
	p.j = 0

	if _, err = io.ReadFull(p.r, p.s[n:]); err != nil {
		return
	}

	b = p.s
	return
}

func (p *Parser) peek(n int) (b []byte, err error) {
	for (p.i + n) > p.j {
		if err = p.fill(); err != nil {
			return
		}
	}
	b = p.b[p.i : p.i+n]
	return
}

func (p *Parser) fill() (err error) {
	n := p.j - p.i
	copy(p.b[:], p.b[p.i:p.j])
	p.i = 0
	p.j = n

	if n, err = p.r.Read(p.b[n:]); n > 0 {
		err = nil
		p.j += n
	} else if err != nil {
		return
	} else {
		err = io.ErrNoProgress
		return
	}

	return
}
