package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
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

// GenerateHashedMetricName generates a collision-free, Prometheus-compliant metric name
func GenerateHashedMetricName(baseSignature string, prefix string) string {
	// Create a new hash
	hash := sha256.New()

	// Write your string to the hash
	hash.Write([]byte(baseSignature))

	// Generate the hash value
	hashValue := hash.Sum(nil)

	// Convert the hash value to a human-readable hexadecimal string
	hashString := hex.EncodeToString(hashValue)[:12] // Take first 12 characters of the hash

	// Create the final metric name by appending the hash string to the prefix
	metricName := strings.Join([]string{prefix, hashString}, "_")

	return metricName
}

// GenerateSignature generates a signature for a VectorSelector in a query.
func GenerateSignature(sel *parser.VectorSelector) string {
	var sb strings.Builder
	sb.WriteString(sel.Name) // metric name
	sb.WriteString("{")
	for i, matcher := range sel.LabelMatchers {
		sb.WriteString(matcher.Name)
		sb.WriteString("=")
		sb.WriteString(matcher.Value) // include actual value
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
			if expr.Grouping != nil { // include dimensions in 'by' clause
				sb.WriteString("_by(")
				for i, dim := range expr.Grouping {
					sb.WriteString(dim)
					if i < len(expr.Grouping)-1 {
						sb.WriteString(",")
					}
				}
				sb.WriteString(")_")
			}
			sb.WriteString("_")
		case *parser.VectorSelector:
			sb.WriteString(GenerateSignature(expr))
			sb.WriteString("_")
		case *parser.NumberLiteral:
			sb.WriteString(fmt.Sprintf("NUM_%f_", expr.Val))
		default:
			if expr != nil {
				panic(fmt.Sprintf("Unexpected node type in safe subtree: %v", expr))
			}
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

// func generateSignatures(queries []string) map[string][]*parser.VectorSelector {
// 	signatureToQueries := make(map[string][]*parser.VectorSelector)

// 	for _, queries := range signatureToQueries {
// 		staticLabels := make(map[string]string)
// 		changingLabels := make(map[string]bool)

// 		// Initialize with the labels from the first query
// 		for _, matcher := range queries[0].LabelMatchers {
// 			staticLabels[matcher.Name] = matcher.Value
// 		}

// 		// Check the remaining queries
// 		for _, query := range queries[1:] {
// 			for name := range staticLabels {
// 				found := false
// 				for _, matcher := range query.LabelMatchers {
// 					if name == matcher.Name {
// 						found = true
// 						if staticLabels[name] != matcher.Value {
// 							// Move to changingLabels and break
// 							changingLabels[name] = true
// 							delete(staticLabels, name)
// 						}
// 						break
// 					}
// 				}
// 				if !found {
// 					// If the label is not found in one of the queries, it's changing.
// 					changingLabels[name] = true
// 					delete(staticLabels, name)
// 				}
// 			}
// 		}

// 		// At this point, `staticLabels` contains labels that are the same across all queries,
// 		// and `changingLabels` contains labels that are different.

// 		// Now you can generate your recording rule
// 	}
// }

type SafeSubtreeFinder struct {
	SafeRoots []parser.Node // store the root nodes of safe subtrees
}

// Visit navigates through the AST and appends safe roots to SafeRoots
func (v *SafeSubtreeFinder) Visit(node parser.Node, path []parser.Node) (parser.Visitor, error) {
	if isSafeNode(node) {
		v.SafeRoots = append(v.SafeRoots, node)
		// Returning nil to stop further traversal of this subtree.
		return nil, nil
	}
	return v, nil
}

// isSafeNode checks if a given node and its subtree are safe for recording rules.
func isSafeNode(node parser.Node) bool {
	switch n := node.(type) {
	case *parser.VectorSelector:
		return true
	case *parser.AggregateExpr:
		return isSafeAggregateExpr(n)
	default:
		return false
	}
}

// isSafeAggregateExpr checks if an aggregate expression is safe.
func isSafeAggregateExpr(expr *parser.AggregateExpr) bool {
	switch expr.Op {
	case parser.AVG, parser.SUM, parser.COUNT:
		return isSafeNode(expr.Expr)
	default:
		return false
	}
}

func ProcessQuery(query string) {
	// Parse the query
	expr, err := parser.ParseExpr(query)
	if err != nil {
		log.Fatalf("Error while parsing the query: %v", err)
	}

	// Initialize the SafeSubtreeFinder
	visitor := &SafeSubtreeFinder{}

	// Walk the AST to find safe subtrees
	parser.Walk(visitor, expr, nil)

	// Generate signatures for each safe subtree and print them
	for _, root := range visitor.SafeRoots {
		sig := GenerateExprSignature(root)
		hashedMetricName := GenerateHashedMetricName(sig, "recording_rule")
		fmt.Printf("Expr: %v\nSignature: %s\nHashedMetricName: %s\n", root, sig, hashedMetricName)
	}
}

func main() {
	// Create a scanner to read from stdin
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Enter queries, one per line (Ctrl-D to terminate):")

	// Read queries from stdin
	for scanner.Scan() {
		query := scanner.Text()
		ProcessQuery(query)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Reading standard input:", err)
	}
}
