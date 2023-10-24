package main

import (
	"fmt"
	"log"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

type QueryVisitor struct {
	Selectors []*parser.VectorSelector
}

func (v *QueryVisitor) Visit(node parser.Node, path []parser.Node) (parser.Visitor, error) {
	switch n := node.(type) {
	case *parser.VectorSelector:
		v.Selectors = append(v.Selectors, n)
	}
	return v, nil
}

func makeSelectorKey(sel *parser.VectorSelector) string {
	// Assuming metric name is sufficient for now,
	// but you can add more details like sorted labels etc.
	return sel.Name
}

func diffSelectors(v1, v2 *QueryVisitor) []*parser.VectorSelector {
	// Create maps for easy look-up
	map1 := make(map[string]*parser.VectorSelector)
	map2 := make(map[string]*parser.VectorSelector)

	for _, sel := range v1.Selectors {
		map1[makeSelectorKey(sel)] = sel
	}
	for _, sel := range v2.Selectors {
		map2[makeSelectorKey(sel)] = sel
	}

	// Find differing nodes
	diffNodes := []*parser.VectorSelector{}
	for k, v1 := range map1 {
		if v2, ok := map2[k]; ok {
			// Compare label matchers
			allNamesSame, differingLabels, _ := labelsEqual(v1.LabelMatchers, v2.LabelMatchers)
			if !allNamesSame || len(differingLabels) > 0 {
				diffNodes = append(diffNodes, v1, v2)
			}
		} else {
			diffNodes = append(diffNodes, v1)
		}
	}

	// Add any remaining selectors from map2
	for k := range map2 {
		if _, ok := map1[k]; !ok {
			diffNodes = append(diffNodes, map2[k])
		}
	}

	return diffNodes
}

// labelsEqual compares two slices of *labels.Matcher for equality
// Returns:
// - bool indicating if all label names are the same
// - slice of label names with differing values
// - slice of label names with the same values
func labelsEqual(l1, l2 []*labels.Matcher) (bool, []string, []string) {
	labelMap1 := make(map[string]*labels.Matcher, len(l1))
	labelMap2 := make(map[string]*labels.Matcher, len(l2))

	// Initialize maps for quick lookup
	for _, lm := range l1 {
		labelMap1[lm.Name] = lm
	}
	for _, lm := range l2 {
		labelMap2[lm.Name] = lm
	}

	allNamesSame := true
	differingLabels := []string{}
	sameLabels := []string{}

	// Check labelMap1 against labelMap2
	for k, v1 := range labelMap1 {
		if v2, ok := labelMap2[k]; ok {
			if v1.Type != v2.Type || v1.Value != v2.Value {
				differingLabels = append(differingLabels, k)
			} else {
				sameLabels = append(sameLabels, k)
			}
		} else {
			allNamesSame = false
			differingLabels = append(differingLabels, k)
		}
	}

	// Check for extra labels in labelMap2
	for k := range labelMap2 {
		if _, ok := labelMap1[k]; !ok {
			allNamesSame = false
			differingLabels = append(differingLabels, k)
		}
	}

	return allNamesSame, differingLabels, sameLabels
}

func main() {
	query1 := `sum(http_request_duration_seconds_bucket{service="service-a",le="+Inf"}) by (service, le)`
	query2 := `sum(http_request_duration_seconds_bucket{service="service-b",le="+Inf"}) by (service, le)`

	// Parse and walk query1
	expr1, err := parser.ParseExpr(query1)
	if err != nil {
		log.Fatalf("Error while parsing the query: %v", err)
	}
	visitor1 := &QueryVisitor{}
	parser.Walk(visitor1, expr1, nil)

	// Parse and walk query2
	expr2, err := parser.ParseExpr(query2)
	if err != nil {
		log.Fatalf("Error while parsing the query: %v", err)
	}
	visitor2 := &QueryVisitor{}
	parser.Walk(visitor2, expr2, nil)

	// Find the differing nodes
	differingNodes := diffSelectors(visitor1, visitor2)

	// You can now use these node references to modify the AST and create a new query
	fmt.Printf("Differing Nodes: %v\n", differingNodes)
}
