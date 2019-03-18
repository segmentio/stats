package stats

import (
	"reflect"
	"testing"
	"time"
	"unsafe"
)

func TestMeasureSize(t *testing.T) {
	size := unsafe.Sizeof(Measure{})
	t.Log("measure size:", size)
}

func TestMakeMeasures(t *testing.T) {
	var testMetrics struct {
		Error struct {
			Count int    `metric:"count" type:"counter"`
			Host  string `tag:"host"` // tests tag overwrite
			Type  string `tag:"type"` // tests local tags
		} `metric:"error"`

		// tests each metric type
		Simple struct {
			A bool `metric:"a" type:"gauge"`

			B int   `metric:"b" type:"counter"`
			C int8  `metric:"c" type:"counter"`
			D int16 `metric:"d" type:"counter"`
			E int32 `metric:"e" type:"counter"`
			F int64 `metric:"f" type:"counter"`

			G uint    `metric:"g" type:"counter"`
			H uint8   `metric:"h" type:"counter"`
			I uint16  `metric:"i" type:"counter"`
			J uint32  `metric:"j" type:"counter"`
			K uint64  `metric:"k" type:"counter"`
			L uintptr `metric:"l" type:"counter"`

			M float32 `metric:"m" type:"histogram"`
			N float64 `metric:"n" type:"histogram"`

			O time.Duration `metric:"o" type:"histogram"`
		}

		Array [3]struct {
			V int  `metric:"v" type:"counter"`
			X bool `metric:"x" type:"gauge"`
		} `metric:"array"`

		// tests that global tags are inherited by sub-fields
		Host        string `tag:"host"`
		Environment string `tag:"environment"`
	}

	testMetrics.Error.Count = 42
	testMetrics.Error.Host = "192.168.0.1"
	testMetrics.Error.Type = "timeout"

	testMetrics.Simple.A = true
	testMetrics.Simple.B = 1
	testMetrics.Simple.C = 2
	testMetrics.Simple.D = 3
	testMetrics.Simple.E = 4
	testMetrics.Simple.F = 5
	testMetrics.Simple.G = 6
	testMetrics.Simple.H = 7
	testMetrics.Simple.I = 8
	testMetrics.Simple.J = 9
	testMetrics.Simple.K = 10
	testMetrics.Simple.L = 11
	testMetrics.Simple.M = 12
	testMetrics.Simple.N = 13
	testMetrics.Simple.O = 14

	testMetrics.Array[0].V = 1
	testMetrics.Array[0].X = true
	testMetrics.Array[1].V = 2
	testMetrics.Array[1].X = false
	testMetrics.Array[2].V = 3
	testMetrics.Array[2].X = true

	testMetrics.Host = "localhost"
	testMetrics.Environment = "development"

	testMeasures := []Measure{
		{
			Name:   "test.error",
			Fields: []Field{MakeField("count", 42, Counter)},
			Tags: []Tag{
				{"environment", "development"},
				{"host", "192.168.0.1"},
				{"service", "test-service"},
				{"type", "timeout"},
			},
		},

		{
			Name: "test",
			Fields: []Field{
				MakeField("a", true, Gauge),
				MakeField("b", int(1), Counter),
				MakeField("c", int8(2), Counter),
				MakeField("d", int16(3), Counter),
				MakeField("e", int32(4), Counter),
				MakeField("f", int64(5), Counter),
				MakeField("g", uint(6), Counter),
				MakeField("h", uint8(7), Counter),
				MakeField("i", uint16(8), Counter),
				MakeField("j", uint32(9), Counter),
				MakeField("k", uint64(10), Counter),
				MakeField("l", uintptr(11), Counter),
				MakeField("m", float32(12), Histogram),
				MakeField("n", float64(13), Histogram),
				MakeField("o", time.Duration(14), Histogram),
			},
			Tags: []Tag{
				{"environment", "development"},
				{"host", "localhost"},
				{"service", "test-service"},
			},
		},

		{
			Name: "test.array",
			Fields: []Field{
				MakeField("v", 1, Counter),
				MakeField("x", true, Gauge),
			},
			Tags: []Tag{
				{"environment", "development"},
				{"host", "localhost"},
				{"service", "test-service"},
			},
		},

		{
			Name: "test.array",
			Fields: []Field{
				MakeField("v", 2, Counter),
				MakeField("x", false, Gauge),
			},
			Tags: []Tag{
				{"environment", "development"},
				{"host", "localhost"},
				{"service", "test-service"},
			},
		},

		{
			Name: "test.array",
			Fields: []Field{
				MakeField("v", 3, Counter),
				MakeField("x", true, Gauge),
			},
			Tags: []Tag{
				{"environment", "development"},
				{"host", "localhost"},
				{"service", "test-service"},
			},
		},
	}

	measures := MakeMeasures("test", testMetrics,
		Tag{"service", "test-service"},
	)

	if !reflect.DeepEqual(measures, testMeasures) {
		t.Error("bad measures:")
		t.Logf("expected: %#v", testMeasures)
		t.Logf("found:    %#v", measures)
	}
}
