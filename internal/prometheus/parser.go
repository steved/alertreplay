package prometheus

import "github.com/prometheus/prometheus/promql/parser"

var Parser = parser.NewParser(parser.Options{})
