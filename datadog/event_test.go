package datadog

import (
	"github.com/segmentio/stats"
)

var testEvents = []struct {
	s string
	e Event
}{
	{
		s: "_e{10,9}:test title|test text\n",
		e: Event{
			Title:     "test title",
			Text:      "test text",
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,24}:test title|test\\line1\\nline2\\nline3\n",
		e: Event{
			Title:     "test title",
			Text:      "test\\line1\nline2\nline3",
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,24}:test|title|test\\line1\\nline2\\nline3\n",
		e: Event{
			Title:     "test|title",
			Text:      "test\\line1\nline2\nline3",
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,9}:test title|test text|d:21\n",
		e: Event{
			Title:     "test title",
			Text:      "test text",
			Ts:        int64(21),
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,9}:test title|test text|p:low\n",
		e: Event{
			Title:     "test title",
			Text:      "test text",
			Priority:  EventPriorityLow,
			AlertType: EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,9}:test title|test text|h:localhost\n",
		e: Event{
			Title:     "test title",
			Text:      "test text",
			Host:      "localhost",
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,9}:test title|test text|t:warning\n",
		e: Event{
			Title:     "test title",
			Text:      "test text",
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeWarning,
		},
	},
	{
		s: "_e{10,9}:test title|test text|k:some aggregation key\n",
		e: Event{
			Title:          "test title",
			Text:           "test text",
			AggregationKey: "some aggregation key",
			Priority:       EventPriorityNormal,
			AlertType:      EventAlertTypeInfo,
		},
	},
	{
		s: "_e{10,9}:test title|test text|s:this is the source\n",
		e: Event{
			Title:          "test title",
			Text:           "test text",
			Priority:       EventPriorityNormal,
			AlertType:      EventAlertTypeInfo,
			SourceTypeName: "this is the source",
		},
	},
	{
		s: "_e{10,9}:test title|test text|#tag1,tag2:test\n",
		e: Event{
			Title:     "test title",
			Text:      "test text",
			Priority:  EventPriorityNormal,
			AlertType: EventAlertTypeInfo,
			Tags: []stats.Tag{
				stats.T("tag1", ""),
				stats.T("tag2", "test"),
			},
		},
	},
}
