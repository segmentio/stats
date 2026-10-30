package linux

import (
	"reflect"
	"testing"
	"time"
)

func TestParseProcCGroupUnified(t *testing.T) {
	proc, err := ParseProcCGroup("0::/kubepods.slice/pod1\n")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(proc, ProcCGroup{{0, "", "/kubepods.slice/pod1"}}) {
		t.Error(proc)
	}
	if cg, ok := proc.Unified(); !ok || cg.Path != "/kubepods.slice/pod1" {
		t.Error(cg, ok)
	}
	if _, ok := proc.Lookup("memory"); ok {
		t.Error("v2 entry must not match v1 controller names")
	}
}

func TestParseCPUMax(t *testing.T) {
	tests := []struct {
		in            string
		quota, period time.Duration
	}{
		{"200000 100000\n", 200 * time.Millisecond, 100 * time.Millisecond},
		{"max 100000\n", -1 * time.Microsecond, 100 * time.Millisecond},
	}
	for _, test := range tests {
		if q, p := parseCPUMax(test.in); q != test.quota || p != test.period {
			t.Errorf("parseCPUMax(%q) = %v, %v", test.in, q, p)
		}
	}
}

func TestWeightToShares(t *testing.T) {
	for weight, shares := range map[int64]int64{0: 0, 1: 2, 40: 1024, 10000: 262144} {
		if got := weightToShares(weight); got != shares {
			t.Errorf("weightToShares(%d) = %d, want %d", weight, got, shares)
		}
	}
}
