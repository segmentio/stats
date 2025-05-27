package cbor

import (
	"bytes"
	"errors"
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

	// Last tag loaded while parsing the type of the next available item.
	tag uint64
	typ objconv.Type

	// This stack is used to keep track of the array map lengths being parsed.
	// The sback array is the initial backend array for the stack.
	stack []int
	sback [16]int
}

func NewParser(r io.Reader) *Parser {
	p := &Parser{r: r}
	p.tag = noTag
	p.stack = p.sback[:0]
	return p
}

func (p *Parser) Reset(r io.Reader) {
	p.r = r
	p.i = 0
	p.j = 0
	p.tag = noTag
	p.stack = p.stack[:0]
}

func (p *Parser) Buffered() io.Reader {
	return bytes.NewReader(p.b[p.i:p.j])
}

func (p *Parser) ParseType() (typ objconv.Type, err error) {
	if p.tag != noTag {
		typ = p.typ
		return
	}

	var t = false
	var s []byte

	if s, err = p.peek(1); err != nil {
		return
	}

	for {
		switch m, b := majorType(s[0]); m {
		case majorType0:
			typ = objconv.Uint

		case majorType1:
			typ = objconv.Int

		case majorType2:
			typ = objconv.Bytes

		case majorType3:
			typ = objconv.String

		case majorType4:
			typ = objconv.Array

		case majorType5:
			typ = objconv.Map

		case majorType6:
			var indef bool
			if t {
				err = errors.New("objconv/cbor: multiple tags found for a single item")
				return
			}
			if p.tag, indef, err = p.parseUint(); err != nil {
				return
			}
			if indef {
				err = errors.New("objconv/cbor: invalid indefinite length for major type 6")
				return
			}
			switch p.tag {
			case tagDateTime, tagTimestamp:
				typ = objconv.Time
			default: // unsupported tag, just fallback to use the base type
				t = true
				continue
			}
			p.typ = typ

		default:
			switch b {
			case svNull, svUndefined:
				typ = objconv.Nil

			case svFalse, svTrue:
				typ = objconv.Bool

			case svFloat16, svFloat32, svFloat64:
				typ = objconv.Float

			case svExtension:
				typ = objconv.Nil // ignore unsupported extensions

			default:
				err = fmt.Errorf("objconv/cbor: unexpected value in major type 7: %d", b)
			}
		}

		return
	}
}

func (p *Parser) ParseNil() (err error) {
	_, err = p.parseType7()
	p.tag = noTag
	return
}

func (p *Parser) ParseBool() (v bool, err error) {
	var b byte

	if b, err = p.parseType7(); err != nil {
		return
	}

	v = b == svTrue
	p.tag = noTag
	return
}

func (p *Parser) ParseInt() (v int64, err error) {
	var u uint64
	var indef bool

	if u, indef, err = p.parseUint(); err != nil {
		return
	}

	if indef {
		err = errors.New("objconv/cbor: invalid indefinite length for major type 1")
		return
	}

	v = -int64(u + 1)
	p.tag = noTag
	return
}

func (p *Parser) ParseUint() (v uint64, err error) {
	var indef bool

	if v, indef, err = p.parseUint(); err != nil {
		return
	}

	if indef {
		err = errors.New("objconv/cbor: invalid indefinite length for major type 0")
		return
	}

	p.tag = noTag
	return
}

func (p *Parser) ParseFloat() (v float64, err error) {
	var s []byte
	var n int
	var b byte

	if b, err = p.parseType7(); err != nil {
		return
	}

	switch b {
	case svFloat16:
		n = 2
	case svFloat32:
		n = 4
	default:
		n = 8
	}

	if s, err = p.peek(n); err != nil {
		return
	}

	switch b {
	case svFloat16:
		v = float64(math.Float32frombits(f16tof32bits(getUint16(s))))
	case svFloat32:
		v = float64(math.Float32frombits(getUint32(s)))
	default:
		v = math.Float64frombits(getUint64(s))
	}

	p.i += n
	p.tag = noTag
	return
}

func (p *Parser) ParseString() (v []byte, err error) {
	if v, err = p.parseBytes(majorType3); err != nil {
		return
	}
	p.tag = noTag
	return
}

func (p *Parser) ParseBytes() (v []byte, err error) {
	if v, err = p.parseBytes(majorType2); err != nil {
		return
	}
	p.tag = noTag
	return
}

func (p *Parser) ParseTime() (v time.Time, err error) {
	var s []byte

	if p.tag == tagDateTime {
		if s, err = p.ParseString(); err != nil {
			return
		}
		if v, err = time.Parse(time.RFC3339Nano, string(s)); err != nil {
			return
		}
	} else {
		if s, err = p.peek(1); err != nil {
			return
		}

		switch m, _ := majorType(s[0]); m {
		case majorType0:
			var u uint64
			if u, err = p.ParseUint(); err != nil {
				return
			}
			if u > int64Max {
				err = fmt.Errorf("objconv/cbor: cannot decode a time value because %d cannot be represented by a signed 64bits integer", u)
				return
			}
			v = time.Unix(int64(u), 0)

		case majorType1:
			var i int64
			if i, err = p.ParseInt(); err != nil {
				return
			}
			v = time.Unix(i, 0)

		case majorType7:
			var f float64
			if f, err = p.ParseFloat(); err != nil {
				return
			}
			s := int64(f)
			ns := int64((f - float64(s)) * float64(time.Second))
			v = time.Unix(s, ns)

		default:
			err = fmt.Errorf("objconv/cbor: cannot decode time value from an item with major type %d", m)
			return
		}
	}

	p.tag = noTag
	return
}

func (p *Parser) ParseDuration() (v time.Duration, err error) {
	panic("objconv/cbor: ParseDuration should never be called because CBOR has no duration type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseError() (v error, err error) {
	panic("objconv/cbor: ParseError should never be called because CBOR has no error type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseArrayBegin() (n int, err error) {
	var u uint64
	var indef bool

	if u, indef, err = p.parseUint(); err != nil {
		return
	}

	if indef {
		n = -1
	} else {
		if u > intMax {
			err = fmt.Errorf("objconv/cbor: array of length %d is greater than what an int can represent", u)
			return
		}
		n = int(u)
	}

	p.stack = append(p.stack, n)
	p.tag = noTag
	return
}

func (p *Parser) ParseArrayEnd(n int) (err error) {
	p.stack = p.stack[:len(p.stack)-1]
	return
}

func (p *Parser) ParseArrayNext(n int) (err error) {
	if p.stack[len(p.stack)-1] < 0 {
		var s []byte

		if s, err = p.peek(1); err != nil {
			return
		}

		if s[0] == 0xFF {
			err = objconv.End
		}
	}
	return
}

func (p *Parser) ParseMapBegin() (n int, err error) {
	var u uint64
	var indef bool

	if u, indef, err = p.parseUint(); err != nil {
		return
	}

	if indef {
		n = -1
	} else {
		if u > intMax {
			err = fmt.Errorf("objconv/cbor: map of length %d is greater than what an int can represent", u)
			return
		}
		n = int(u)
	}

	p.stack = append(p.stack, n)
	p.tag = noTag
	return
}

func (p *Parser) ParseMapEnd(n int) (err error) {
	p.stack = p.stack[:len(p.stack)-1]
	return
}

func (p *Parser) ParseMapValue(n int) (err error) {
	return
}

func (p *Parser) ParseMapNext(n int) (err error) {
	if p.stack[len(p.stack)-1] < 0 {
		var s []byte

		if s, err = p.peek(1); err != nil {
			return
		}

		if s[0] == 0xFF {
			err = objconv.End
		}
	}
	return
}

func (p *Parser) parseUint() (v uint64, indef bool, err error) {
	var s []byte
	var n int

	if s, err = p.peek(1); err != nil {
		return
	}

	_, b := majorType(s[0])

	switch {
	case b <= 23:
		v = uint64(b)
		p.i++
		return
	case b == 31:
		indef = true
		p.i++
		return
	case b == iUint8:
		n = 2
	case b == iUint16:
		n = 3
	case b == iUint32:
		n = 5
	default:
		n = 9
	}

	if s, err = p.peek(n); err != nil {
		n = 0
		return
	}

	switch b {
	case iUint8:
		v = uint64(s[1])
	case iUint16:
		v = uint64(getUint16(s[1:]))
	case iUint32:
		v = uint64(getUint32(s[1:]))
	default:
		v = getUint64(s[1:])
	}

	p.i += n
	return
}

func (p *Parser) parseBytes(m byte) (v []byte, err error) {
	var s []byte
	var u uint64
	var indef bool

	p.s = p.s[:0]

	if u, indef, err = p.parseUint(); err != nil {
		return
	}

	if !indef {
		if u > intMax {
			err = fmt.Errorf("objconv/cbor: byte string of length %d is greater than what an int can represent", u)
			return
		}
		return p.load(int(u))
	}

	for {
		if u, indef, err = p.parseUint(); err != nil {
			return
		}

		if indef {
			break
		}

		if n := uint64(len(s)); u > (intMax - n) {
			err = fmt.Errorf("objconv/cbor: byte string of length %d is greater than what an int can represent", u+n)
			return
		}

		if v, err = p.load(int(u)); err != nil {
			return
		}
	}

	return
}

func (p *Parser) parseType7() (b byte, err error) {
	var s []byte

	if s, err = p.peek(1); err != nil {
		return
	}

	if _, b = majorType(s[0]); b != svExtension {
		p.i++
	} else {
		if s, err = p.peek(2); err != nil {
			return
		}
		if b = s[1]; b < 32 {
			err = fmt.Errorf("objconv/cbor: invalid extended simple value in major type 7: %d", b)
			return
		}
		p.i += 2
	}

	return
}

func (p *Parser) load(n int) (b []byte, err error) {
	i := len(p.s)
	j := i + n

	if cap(p.s) < j {
		p.s = make([]byte, j, align(j, 1024))
	} else {
		p.s = p.s[:j]
	}

	if p.i != p.j {
		n1 := n
		n2 := p.j - p.i

		if n1 > n2 {
			n1 = n2
		}

		copy(p.s[i:], p.b[p.i:p.i+n1])

		if p.i += n1; p.i == p.j {
			p.i = 0
			p.j = 0
		}

		i += n1
	}

	if i != j {
		if _, err = io.ReadFull(p.r, p.s[i:]); err != nil {
			return
		}
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
