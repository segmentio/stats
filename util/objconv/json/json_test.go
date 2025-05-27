package json

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/segmentio/stats/v5/util/objconv/objtests"
)

func TestCodec(t *testing.T) {
	objtests.TestCodec(t, Codec)
}

func BenchmarkCodec(b *testing.B) {
	objtests.BenchmarkCodec(b, Codec)
}

func TestPrettyCodec(t *testing.T) {
	objtests.TestCodec(t, PrettyCodec)
}

func BenchmarkPrettyCodec(b *testing.B) {
	objtests.BenchmarkCodec(b, PrettyCodec)
}

func TestUnicode(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{`"\u2022"`, "•"},
		{`"\uDC00D800"`, "�"},
	}

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			var s string

			if err := Unmarshal([]byte(test.in), &s); err != nil {
				t.Error(err)
			}

			if s != test.out {
				t.Error(s)
			}
		})
	}
}

func TestMapValueOverflow(t *testing.T) {
	src := fmt.Sprintf(
		`{"A":"good","skip1":"%s","B":"bad","skip2":"%sA"}`,
		strings.Repeat("0", 102),
		strings.Repeat("0", 110),
	)

	val := struct{ A string }{}

	if err := Unmarshal([]byte(src), &val); err != nil {
		t.Error(err)
		return
	}

	if val.A != "good" {
		t.Error(val.A)
	}
}

func TestEmitImpossibleFloats(t *testing.T) {
	values := []float64{
		math.NaN(),
		math.Inf(+1),
		math.Inf(-1),
	}

	for _, v := range values {
		t.Run(fmt.Sprintf("emitting %v must return an error", v), func(t *testing.T) {
			e := Emitter{}

			if err := e.EmitFloat(v, 64); err == nil {
				t.Error("no error was returned")
			}
		})
	}
}
