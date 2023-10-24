package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

type SelectorWithPath struct {
	Selector *parser.VectorSelector
	Path     []parser.Node
}

type QueryVisitor struct {
	Selectors []*SelectorWithPath
}

func (v *QueryVisitor) Visit(node parser.Node, path []parser.Node) (parser.Visitor, error) {
	switch n := node.(type) {
	case *parser.VectorSelector:
		selectorWithPath := &SelectorWithPath{
			Selector: n,
			Path:     append([]parser.Node{}, path...), // Copying path to avoid modification
		}
		v.Selectors = append(v.Selectors, selectorWithPath)
	}
	return v, nil
}

func makeSelectorKey(sel *SelectorWithPath) string {
	// Assuming metric name is sufficient for now,
	// but you can add more details like sorted labels etc.
	return sel.Selector.Name
}

func diffSelectors(v1, v2 *QueryVisitor) []*SelectorWithPath {
	// Create maps for easy look-up
	map1 := make(map[string]*SelectorWithPath)
	map2 := make(map[string]*SelectorWithPath)

	for _, sel := range v1.Selectors {
		map1[makeSelectorKey(sel)] = sel
	}
	for _, sel := range v2.Selectors {
		map2[makeSelectorKey(sel)] = sel
	}

	// Find differing nodes
	diffNodes := []*SelectorWithPath{}
	for k, v1 := range map1 {
		if v2, ok := map2[k]; ok {
			// Compare label matchers
			allNamesSame, differingLabels, _ := labelsEqual(v1.Selector.LabelMatchers, v2.Selector.LabelMatchers)
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

// GenerateSignature generates a signature for a VectorSelector in a query.
func GenerateSignature(sel *parser.VectorSelector) string {
	var sb strings.Builder
	sb.WriteString(sel.Name) // assuming metric name will be part of signature
	sb.WriteString("{")
	for i, matcher := range sel.LabelMatchers {
		sb.WriteString(matcher.Name)
		sb.WriteString("=")
		if matcher.Type == labels.MatchEqual {
			sb.WriteString("<val>")
		} else {
			sb.WriteString("<complex_val>")
		}
		if i < len(sel.LabelMatchers)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// GenerateExprSignature generates a signature for an entire expression in a query.
func GenerateExprSignature(node parser.Node) string {
	var sb strings.Builder
	parser.Inspect(node, func(n parser.Node, path []parser.Node) error {
		switch expr := n.(type) {
		case *parser.AggregateExpr:
			sb.WriteString(expr.Op.String())
			sb.WriteString("_")
		case *parser.VectorSelector:
			sb.WriteString(GenerateSignature(expr))
			sb.WriteString("_")
		}
		return nil
	})
	return sb.String()
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
	queries := []string{
		`sum(http_request_duration_seconds_bucket{service="service-a",le="+Inf"}) by (service, le)`,
		`sum(http_request_duration_seconds_bucket{service="service-b",le="+Inf"}) by (service, le)`,
		`sum(http_request_duration_seconds_bucket{service="service-a",le="+Inf"}) by (le)`,
		`sum(http_request_duration_seconds_bucket{service="service-a",le="+Inf"}) by (le)`,
		`sum(http_request_duration_seconds_bucket{service="service-b"}) by (le)`,
	}

	signatureMap := make(map[string][]parser.Expr)

	for _, query := range queries {
		expr, err := parser.ParseExpr(query)
		if err != nil {
			log.Fatalf("Error while parsing the query: %v", err)
		}
		sig := GenerateExprSignature(expr)
		signatureMap[sig] = append(signatureMap[sig], expr)
	}

	// Now, signatureMap contains all the selectors grouped by their signatures.
	// You can now proceed with additional checks or optimizations on each group.
	for signature, selectors := range signatureMap {
		fmt.Printf("Signature: %s\n", signature)
		for _, sel := range selectors {
			fmt.Printf("  - %s\n", sel)
		}
	}
}
