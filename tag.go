package stats

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"

	"github.com/segmentio/jutil"
)

type Tag struct {
	Name  string
	Value string
}

type Tags []Tag

func NewTags(v interface{}) Tags {
	if v == nil {
		return nil
	}

	switch x := v.(type) {
	case Tags:
		return copyTags(x)

	case []Tag:
		return copyTags(Tags(x))

	case map[string]string:
		return newTagsFromMapStringString(x)

	case map[string]interface{}:
		return newTagsFromMapStringInterface(x)

	default:
		return newTags(reflect.ValueOf(v))
	}
}

func newTags(v reflect.Value) Tags {
	switch v.Kind() {
	case reflect.Struct:
		return newTagsFromStruct(v)

	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			return newTagsFromMap(v)
		}
	}

	panic("stats.NewTags: tags can only be constructed from structs or map[string]")
}

func newTagsFromStruct(v reflect.Value) Tags {
	s := jutil.LookupStruct(v.Type())
	t := make(Tags, 0, len(s))

	for _, f := range s {
		if fv := v.FieldByIndex(f.Index); !f.Omitempty || !jutil.IsEmptyValue(fv) {
			t = append(t, Tag{
				Name:  f.Name,
				Value: makeString(fv),
			})
		}
	}

	return t
}

func newTagsFromMap(v reflect.Value) Tags {
	keys := make([]string, 0, v.Len())

	for _, k := range v.MapKeys() {
		keys = append(keys, makeString(k))
	}

	sort.Strings(keys)
	t := make(Tags, len(keys))

	for i, k := range keys {
		t[i] = Tag{
			Name:  k,
			Value: makeString(v.MapIndex(reflect.ValueOf(k))),
		}
	}

	return t
}

func newTagsFromMapStringString(m map[string]string) Tags {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	t := make(Tags, len(keys))

	for i, k := range keys {
		t[i] = Tag{
			Name:  k,
			Value: m[k],
		}
	}

	return t
}

func newTagsFromMapStringInterface(m map[string]interface{}) Tags {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	t := make(Tags, len(keys))

	for i, k := range keys {
		t[i] = Tag{
			Name:  k,
			Value: fmt.Sprint(m[k]),
		}
	}

	return t
}

func copyTags(t Tags) Tags {
	x := make(Tags, len(t))
	copy(x, t)
	return x
}

func makeString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprint(v.Interface())
	}
}

func (t Tags) Copy() Tags {
	c := make(Tags, len(t))
	copy(c, t)
	return c
}

func (t Tags) String() string {
	b, _ := t.MarshalJSON()
	return string(b)
}

func (t Tags) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	b.Grow(20 * len(t))
	b.WriteByte('{')

	for i, x := range t {
		if i != 0 {
			b.WriteByte(',')
		}
		jutil.WriteQuoted(b, []byte(x.Name))
		b.WriteByte(':')
		jutil.WriteQuoted(b, []byte(x.Value))
	}

	b.WriteByte('}')
	return b.Bytes(), nil
}

func (tags Tags) Format(f fmt.State, _ rune) {
	for i, tag := range tags {
		if i != 0 {
			io.WriteString(f, " ")
		}
		io.WriteString(f, tag.Name)
		io.WriteString(f, "=")
		io.WriteString(f, tag.Value)
	}
}
