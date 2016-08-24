package linux

import "fmt"

func convertPanicToError(v interface{}) (err error) {
	if v != nil {
		switch e := v.(type) {
		case error:
			err = e
		default:
			err = fmt.Errorf("%v", e)
		}
	}
	return
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
