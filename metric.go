package stats

type Metric interface {
	Name() string

	Tags() Tags
}

func NewMetric(name string, tags ...Tag) Metric {
	return metric{name: name, tags: Tags(tags)}
}

type metric struct {
	name string
	tags Tags
}

func (m metric) Name() string { return m.name }

func (m metric) Tags() Tags { return m.tags }

func (m metric) String() string { return m.name + " " + m.tags.String() }
