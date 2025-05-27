package resp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/stats/v5/util/objconv"
	"github.com/segmentio/stats/v5/util/objconv/objutil"
)

var (
	null = [...]byte{'-', '1'}
)

type Parser struct {
	r io.Reader // reader to load bytes from
	i int       // offset of the end of line in s
	n int       // offset of the first unread byte in s
	s []byte    // buffer used for building strings
	a [128]byte // initial backend array for s
	b [128]byte // buffer where bytes are loaded from the reader
}

func NewParser(r io.Reader) *Parser {
	return &Parser{r: r}
}

func (p *Parser) Reset(r io.Reader) {
	p.r = r
	p.n = 0
	p.s = nil
}

func (p *Parser) Buffered() io.Reader {
	return bytes.NewReader(p.s[p.n:])
}

func (p *Parser) ParseType() (t objconv.Type, err error) {
	var line []byte

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of the stream")
		return
	}

	switch line[0] {
	case '+':
		t = objconv.String

	case '-':
		t = objconv.Error

	case ':':
		t = objconv.Int

	case '$':
		if bytes.Equal(line[1:], null[:]) {
			t = objconv.Nil
		} else {
			t = objconv.Bytes
		}

	case '*':
		if bytes.Equal(line[1:], null[:]) {
			t = objconv.Nil
		} else {
			t = objconv.Array
		}

	default:
		err = fmt.Errorf("objconv/resp: expected type token but found %#v", string(line))
	}

	return
}

func (p *Parser) ParseNil() (err error) {
	var line []byte

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of a null value")
		return
	}

	switch line[0] {
	case '$', '*':
	default:
		goto failure
	}

	if !bytes.Equal(line[1:], null[:]) {
		goto failure
	}

	p.skipLine()
	return
failure:
	err = fmt.Errorf("objconv/resp: expected null value but found %#v", string(line))
	return
}

func (p *Parser) ParseBool() (v bool, err error) {
	panic("objconv/resp: ParseBool should never be called because RESP has no boolean type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseInt() (v int64, err error) {
	var line []byte

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of an integer value")
		return
	}

	if line[0] != ':' {
		goto failure
	}

	if v, err = objutil.ParseInt(line[1:]); err != nil {
		goto failure
	}

	p.skipLine()
	return
failure:
	err = fmt.Errorf("objconv/resp: expected integer value but found %#v", string(line))
	return
}

func (p *Parser) ParseUint() (v uint64, err error) {
	panic("objconv/resp: ParseBool should never be called because RESP has no unsigned integer type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseFloat() (v float64, err error) {
	panic("objconv/resp: ParseFloat should never be called because RESP has no floating point type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseString() (v []byte, err error) {
	var line []byte

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of a simple string value")
		return
	}

	if line[0] != '+' {
		goto failure
	}

	v = line[1:]
	p.skipLine()
	return
failure:
	err = fmt.Errorf("objconv/resp: expected simple string value but found %#v", string(line))
	return
}

func (p *Parser) ParseBytes() (v []byte, err error) {
	var line []byte
	var size int64

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of a bulk string value")
		return
	}

	if line[0] != '$' {
		goto failure
	}

	if size, err = objutil.ParseInt(line[1:]); err != nil || size < 0 || size > int64(objutil.IntMax) {
		goto failure
	}
	p.skipLine()

	if v, err = p.peekChunk(int(size)); err != nil {
		return
	}
	p.n += len(v) + 2
	return
failure:
	err = fmt.Errorf("objconv/resp: expected bulk string value but found %#v", string(line))
	return
}

func (p *Parser) ParseTime() (v time.Time, err error) {
	panic("objconv/resp: ParseTime should never be called because RESP has no time type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseDuration() (v time.Duration, err error) {
	panic("objconv/resp: ParseDuration should never be called because RESP has no duration type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseError() (v error, err error) {
	var line []byte

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of an error value")
		return
	}

	if line[0] != '-' {
		goto failure
	}

	v = NewError(string(line[1:]))
	p.skipLine()
	return
failure:
	err = fmt.Errorf("objconv/resp: expected simple string value but found %#v", string(line))
	return
}

func (p *Parser) ParseArrayBegin() (n int, err error) {
	var line []byte
	var size int64

	if line, err = p.peekLine(); err != nil {
		return
	}

	if len(line) == 0 {
		err = errors.New("objconv/resp: invalid empty line at the beginning of an array value")
		return
	}

	if line[0] != '*' {
		goto failure
	}

	if size, err = objutil.ParseInt(line[1:]); err != nil || size < 0 || size > int64(objutil.IntMax) {
		goto failure
	}

	p.skipLine()
	n = int(size)
	return
failure:
	err = fmt.Errorf("objconv/resp: expected bulk string value but found %#v", string(line))
	return
}

func (p *Parser) ParseArrayEnd(n int) (err error) {
	return
}

func (p *Parser) ParseArrayNext(n int) (err error) {
	return
}

func (p *Parser) ParseMapBegin() (n int, err error) {
	panic("objconv/resp: ParseMapBegin should never be called because RESP has no map type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseMapEnd(n int) (err error) {
	panic("objconv/resp: ParseMapEnd should never be called because RESP has no map type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseMapValue(n int) (err error) {
	panic("objconv/resp: ParseMapValue should never be called because RESP has no map type, this is likely a bug in the decoder code")
}

func (p *Parser) ParseMapNext(n int) (err error) {
	panic("objconv/resp: ParseMapNext should never be called because RESP has no map type, this is likely a bug in the decoder code")
}

func (p *Parser) peekLine() (line []byte, err error) {
	if p.i != 0 {
		line = p.s[p.n : p.i-2]
		return
	}

	if p.s == nil {
		p.s = p.a[:0]
	}

	for {
		if i := bytesIndexCRLF(p.s[p.n:]); i >= 0 {
			line, p.i = p.s[p.n:p.n+i], p.n+i+2
			return
		}

		if p.n != 0 { // pack
			copy(p.s, p.s[p.n:])
			p.s = p.s[:len(p.s)-p.n]
			p.n = 0
		}

		var n int
		if n, err = p.r.Read(p.b[:]); n > 0 {
			err = nil
			p.s = append(p.s, p.b[:n]...)
		}

		if err != nil {
			return
		}
	}
}

func (p *Parser) peekChunk(size int) (chunk []byte, err error) {
	size += 2 // CRLF

	for {
		if len(p.s) >= (size + p.n) {
			chunk = p.s[p.n : p.n+size]
			break
		}

		var n int

		if n, err = p.r.Read(p.b[:]); n > 0 {
			err = nil
			p.s = append(p.s, p.b[:n]...)
		} else if err != nil {
			return
		} else {
			err = io.ErrNoProgress
			return
		}
	}

	chunk = p.s[p.n : p.n+size]

	if !bytes.HasSuffix(chunk, crlfBytes[:]) {
		err = fmt.Errorf("objconv/resp: expected a CRLF sequence at the end of a bulk string but found %#v", string(chunk))
	} else {
		chunk = chunk[:len(chunk)-2]
	}

	return
}

func (p *Parser) skipLine() {
	p.n, p.i = p.i, 0
}

func bytesIndexCRLF(b []byte) int {
	for i, n := 0, len(b); i != n; i++ {
		j := bytes.IndexByte(b[i:], '\r')

		if j < 0 {
			break
		}

		if j++; j == n {
			break
		}

		if b[j] == '\n' {
			return j - 1
		}
	}
	return -1
}
