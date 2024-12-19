package version

import "testing"

func TestDevel(t *testing.T) {
	if isDevel("1.23.4") {
		t.Errorf("expected 1.23.4 to return false; got true")
	}
	if !isDevel("devel go1.24-d1d9312950 Wed Jan 1 21:18:59 2025 -0800 darwin/arm64") {
		t.Errorf("expected tip version to return true; got false")
	}
}
