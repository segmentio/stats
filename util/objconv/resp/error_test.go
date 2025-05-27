package resp

import "testing"

func TestErrorType(t *testing.T) {
	tests := []struct {
		err Error
		typ string
	}{
		{
			err: Error("ERR wrong number of arguments for 'set' command"),
			typ: "ERR",
		},
		{
			err: Error(""),
			typ: "",
		},
		{
			err: Error("hello world!"),
			typ: "",
		},
	}

	for _, test := range tests {
		t.Run(test.err.Error(), func(t *testing.T) {
			if typ := test.err.Type(); typ != test.typ {
				t.Error("bad error type:", typ)
			}
		})
	}
}
