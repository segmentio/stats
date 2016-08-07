package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
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
	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			return newTagsFromMap(v)
		}
	}
	panic("stats.NewTags: tags can only be constructed from other tags or map[string]")
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
	if len(t) == 0 {
		return nil
	}
	x := make(Tags, len(t))
	copy(x, t)
	return x
}

func concatTags(t1 Tags, t2 Tags) Tags {
	n1 := len(t1)
	n2 := len(t2)

	if n1 == 0 {
		return t2
	}

	if n2 == 0 {
		return t1
	}

	t3 := make(Tags, n1+n2)
	copy(t3, t1)
	copy(t3[n1:], t2)
	return t3
}

func makeString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprint(v.Interface())
	}
}

func (t Tags) Get(name string) string {
	for _, tag := range t {
		if tag.Name == name {
			return tag.Value
		}
	}
	return ""
}

func (t Tags) String() string {
	b, _ := t.MarshalJSON()
	return string(b)
}

func (t Tags) MarshalJSON() ([]byte, error) {
	m := make(map[string]string, len(t))

	for _, x := range t {
		m[x.Name] = x.Value
	}

	return json.Marshal(m)
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
