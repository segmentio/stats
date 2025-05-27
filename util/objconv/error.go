package objconv

import (
	"errors"
	"fmt"
)

func typeConversionError(from, to Type) error {
	return fmt.Errorf("objconv: cannot convert from %s to %s", from, to)
}

// End is expected to be returned to indicate that a function has completed
// its work, this is usually employed in generic algorithms.
//
//revive:disable:error-naming
var End = errors.New("end") //nolint:staticcheck
