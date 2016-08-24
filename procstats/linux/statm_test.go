package linux

import (
	"reflect"
	"testing"
)

func TestParseProcStatm(t *testing.T) {
	text := `1134 172 153 12 0 115 0`

	proc, err := ParseProcStatm(text)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(proc, ProcStatm{
		Size:     1134,
		Resident: 172,
		Share:    153,
		Text:     12,
		Lib:      0,
		Data:     115,
		Dt:       0,
	}) {
		t.Error(proc)
	}
}
