package resp

import (
	"strings"
	"unicode"
)

// The Error type represents redis errors.
type Error string

// NewError returns a new redis error.
func NewError(s string) *Error {
	e := Error(s)
	return &e
}

// Error satsifies the error interface.
func (e *Error) Error() string {
	return string(*e)
}

// Type returns the RESP error type, which is represented by the leading
// uppercase word in the error string.
func (e *Error) Type() string {
	s := e.Error()

	if i := strings.IndexByte(s, ' '); i < 0 {
		s = ""
	} else {
		s = s[:i]

		for _, c := range s {
			if !unicode.IsUpper(c) {
				s = ""
				break
			}
		}
	}

	return s
}
