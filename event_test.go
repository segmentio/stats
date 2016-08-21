package stats

import "testing"

func TestEventString(t *testing.T) {
	tests := []struct {
		e Event
		s string
	}{
		{
			e: Event{},
			s: `{ type: "", name: "", value: 0, sample: 0, time: "0001-01-01 00:00:00 +0000 UTC", tags: [] }`,
		},
	}

	for _, test := range tests {
		if s := test.e.String(); s != test.s {
			t.Errorf("invalid event representation: %s != %s", test.s, s)
		}
	}
}
