package main

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
)

func TestLabelsEqual(t *testing.T) {
	t.Run("equal labels", func(t *testing.T) {
		l1 := []*labels.Matcher{
			{Name: "job", Type: labels.MatchEqual, Value: "node_exporter"},
			{Name: "instance", Type: labels.MatchEqual, Value: "localhost:9100"},
		}
		l2 := []*labels.Matcher{
			{Name: "job", Type: labels.MatchEqual, Value: "node_exporter"},
			{Name: "instance", Type: labels.MatchEqual, Value: "localhost:9100"},
		}
		if !labelsEqual(l1, l2) {
			t.Errorf("expected labelsEqual to return true, but got false")
		}
	})

	t.Run("unequal labels", func(t *testing.T) {
		l1 := []*labels.Matcher{
			{Name: "job", Type: labels.MatchEqual, Value: "node_exporter"},
			{Name: "instance", Type: labels.MatchEqual, Value: "localhost:9100"},
		}
		l2 := []*labels.Matcher{
			{Name: "job", Type: labels.MatchEqual, Value: "node_exporter"},
			{Name: "instance", Type: labels.MatchEqual, Value: "localhost:9090"},
		}
		if labelsEqual(l1, l2) {
			t.Errorf("expected labelsEqual to return false, but got true")
		}
	})

	t.Run("unequal number of labels", func(t *testing.T) {
		l1 := []*labels.Matcher{
			{Name: "job", Type: labels.MatchEqual, Value: "node_exporter"},
			{Name: "instance", Type: labels.MatchEqual, Value: "localhost:9100"},
		}
		l2 := []*labels.Matcher{
			{Name: "job", Type: labels.MatchEqual, Value: "node_exporter"},
		}
		if labelsEqual(l1, l2) {
			t.Errorf("expected labelsEqual to return false, but got true")
		}
	})
}
