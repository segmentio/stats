package stats

import "bytes"

type multiError []error

func appendError(list error, errors ...error) error {
	for _, err := range errors {
		if err != nil {
			if list == nil {
				list = err
			} else if l, ok := list.(multiError); ok {
				if e, ok := err.(multiError); ok {
					list = append(l, e...)
				} else {
					list = append(l, err)
				}
			} else {
				list = multiError{list, err}
			}
		}
	}
	return list
}

func (m multiError) Error() string {
	switch len(m) {
	case 0:
		return ""
	case 1:
		return m[0].Error()
	default:
		b := &bytes.Buffer{}
		b.Grow(100 * len(m))

		for _, e := range m {
			b.WriteString(e.Error())
			b.WriteByte('\n')
		}

		return b.String()
	}
}
