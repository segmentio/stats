package stats

import (
	"reflect"
	"testing"
)

func TestTimerStart(t *testing.T) {
	h := &handler{}
	e := NewEngine("E")
	e.Register(h)

	m := e.Timer("A", Tag{"base", "tag"}, Tag{"extra", "tag"})
	c := m.Start()

	if name := c.Name(); name != "A" {
		t.Error("bad timer name:", name)
	}

	if tags := c.Tags(); !reflect.DeepEqual(tags, []Tag{{"base", "tag"}, {"extra", "tag"}}) {
		t.Error("bad timer tags:", tags)
	}
}

func TestTimerClone(t *testing.T) {
	e := NewEngine("E")
	c1 := e.Timer("A", Tag{"base", "tag"})
	c2 := c1.Clone(Tag{"extra", "tag"})

	if name := c2.Name(); name != "A" {
		t.Error("bad timer name:", name)
	}

	if tags := c2.Tags(); !reflect.DeepEqual(tags, []Tag{{"base", "tag"}, {"extra", "tag"}}) {
		t.Error("bad timer tags:", tags)
	}
}
