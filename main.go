package main

import (
	"fmt"
	"log"

	"github.com/prometheus/prometheus/promql/parser"
)

type queryDiffVisitor struct {
	Selectors []string
}

func (v *queryDiffVisitor) Visit(node parser.Node, path []parser.Node) (parser.Visitor, error) {
	switch n := node.(type) {
	case *parser.VectorSelector:
		v.Selectors = append(v.Selectors, n.String())
		// You can extend this for MatrixSelector or other types if needed
	}
	return v, nil
}

func getSelectorsFromQuery(query string) []string {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		log.Fatalf("Error while parsing the query: %v", err)
	}

	visitor := &queryDiffVisitor{}
	parser.Walk(visitor, expr, nil)

	return visitor.Selectors
}

func main() {
	// // Your PromQL query string
	// query := `http_requests_total{method="GET"}`

	// // Parse the expression
	// expr, err := parser.ParseExpr(query)
	// if err != nil {
	// 	log.Fatalf("Error while parsing the PromQL query: %v", err)
	// }

	// // Do something with the parsed expression
	// handleExpression(expr)
	query1 := `sum(http_request_duration_seconds_bucket{service="service-a",le="+Inf"}) by (service, le)`
	query2 := `sum(http_request_duration_seconds_bucket{service="service-b",le="+Inf"}) by (service, le)`

	selectors1 := getSelectorsFromQuery(query1)
	selectors2 := getSelectorsFromQuery(query2)

	// Now compare selectors1 and selectors2 to find differences and generate the generalized query
	fmt.Println("Selectors from Query 1:", selectors1)
	fmt.Println("Selectors from Query 2:", selectors2)
}

func handleExpression(expr parser.Expr) {
	// Switch based on the type of the expression
	switch e := expr.(type) {
	case *parser.VectorSelector:
		// Handle Vector Selector
		fmt.Printf("Vector Selector: %s\n", e.Name)
	case *parser.MatrixSelector:
		// Handle Matrix Selector
		fmt.Printf("Matrix Selector: %s[%s]\n", e.VectorSelector.Pretty(0), e.Range)
	// Add more cases as needed
	default:
		fmt.Printf("Unhandled expression type: %T\n", e)
	}
}
