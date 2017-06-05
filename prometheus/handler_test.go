package prometheus

import (
	"fmt"
	"testing"
)

func TestAcceptEncoding(t *testing.T) {
	tests := []struct {
		accept string
		check  string
		expect bool
	}{
		{
			accept: "",
			check:  "gzip",
			expect: false,
		},

		{
			accept: "gzip",
			check:  "gzip",
			expect: true,
		},

		{
			accept: "gzip, deflate, sdch, br",
			check:  "gzip",
			expect: true,
		},

		{
			accept: "deflate, sdch, br",
			check:  "gzip",
			expect: false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s?", test.accept, test.check), func(t *testing.T) {
			if ok := acceptEncoding(test.accept, test.check); ok != test.expect {
				t.Error(ok)
			}
		})
	}
}
