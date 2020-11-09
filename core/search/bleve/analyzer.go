package bleve

import (
	"regexp"

	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	filterregex "github.com/blevesearch/bleve/analysis/char/regexp"
	"github.com/blevesearch/bleve/registry"
)

const CustomerAnalyzerName = "space_search_analyzer"

/// Customer Analyzer extends the standard analyzer by registering a regexp character filter
func CustomAnalyzerConstructor(config map[string]interface{}, cache *registry.Cache) (*analysis.Analyzer, error) {
	rv, err := standard.AnalyzerConstructor(config, cache)
	if err != nil {
		return nil, err
	}

	// replace . with white space - helps to improve results on filenames
	pattern, err := regexp.Compile("\\.")
	if err != nil {
		return nil, err
	}
	replacement := []byte(" ")
	regexpCharFilter := filterregex.New(pattern, replacement)
	rv.CharFilters = append(rv.CharFilters, regexpCharFilter)

	return rv, nil
}

func init() {
	registry.RegisterAnalyzer(CustomerAnalyzerName, CustomAnalyzerConstructor)
}
