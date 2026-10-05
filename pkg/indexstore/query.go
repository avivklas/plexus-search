package indexstore

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// QueryType defines the variety of search queries supported.
type QueryType string

const (
	QueryTypeMatch        QueryType = "match"
	QueryTypeMatchPhrase  QueryType = "match_phrase"
	QueryTypeTerm         QueryType = "term"
	QueryTypePrefix       QueryType = "prefix"
	QueryTypeFuzzy        QueryType = "fuzzy"
	QueryTypeBoolean      QueryType = "boolean"
	QueryTypeNumericRange QueryType = "numeric_range"
	QueryTypeMatchAll     QueryType = "match_all"
	QueryTypeQueryString  QueryType = "query_string"
)

// QueryDefinition describes a structured or flexible search query.
type QueryDefinition struct {
	Type          QueryType         `json:"type,omitempty"`
	Field         string            `json:"field,omitempty"`
	Value         any               `json:"value,omitempty"`
	Fuzziness     int               `json:"fuzziness,omitempty"`
	Min           *float64          `json:"min,omitempty"`
	Max           *float64          `json:"max,omitempty"`
	MinInclusive  *bool             `json:"min_inclusive,omitempty"`
	MaxInclusive  *bool             `json:"max_inclusive,omitempty"`
	Must          []QueryDefinition `json:"must,omitempty"`
	Should        []QueryDefinition `json:"should,omitempty"`
	MustNot       []QueryDefinition `json:"must_not,omitempty"`
}

// UnmarshalJSON allows QueryDefinition to be parsed from either a string or an object.
func (q *QueryDefinition) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil
	}

	// If query is a plain string: e.g. "raft consensus"
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		q.Type = QueryTypeMatch
		q.Value = s
		return nil
	}

	// Normal struct unmarshaling
	type alias QueryDefinition
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*q = QueryDefinition(a)
	return nil
}

// BuildBleveQuery converts QueryDefinition into a concrete Bleve query.Query.
func (q *QueryDefinition) BuildBleveQuery() (query.Query, error) {
	typ := q.Type
	if typ == "" {
		if len(q.Must) > 0 || len(q.Should) > 0 || len(q.MustNot) > 0 {
			typ = QueryTypeBoolean
		} else if q.Min != nil || q.Max != nil {
			typ = QueryTypeNumericRange
		} else if q.Value != nil {
			typ = QueryTypeMatch
		} else {
			typ = QueryTypeMatchAll
		}
	}

	valStr := ""
	if q.Value != nil {
		valStr = fmt.Sprintf("%v", q.Value)
	}

	switch typ {
	case QueryTypeMatch:
		mq := bleve.NewMatchQuery(valStr)
		if q.Field != "" {
			mq.SetField(q.Field)
		}
		return mq, nil

	case QueryTypeMatchPhrase:
		mpq := bleve.NewMatchPhraseQuery(valStr)
		if q.Field != "" {
			mpq.SetField(q.Field)
		}
		return mpq, nil

	case QueryTypeTerm:
		tq := bleve.NewTermQuery(valStr)
		if q.Field != "" {
			tq.SetField(q.Field)
		}
		return tq, nil

	case QueryTypePrefix:
		pq := bleve.NewPrefixQuery(valStr)
		if q.Field != "" {
			pq.SetField(q.Field)
		}
		return pq, nil

	case QueryTypeFuzzy:
		fq := bleve.NewFuzzyQuery(valStr)
		if q.Field != "" {
			fq.SetField(q.Field)
		}
		if q.Fuzziness > 0 {
			fq.SetFuzziness(q.Fuzziness)
		}
		return fq, nil

	case QueryTypeNumericRange:
		minInc := true
		maxInc := true
		if q.MinInclusive != nil {
			minInc = *q.MinInclusive
		}
		if q.MaxInclusive != nil {
			maxInc = *q.MaxInclusive
		}
		nq := bleve.NewNumericRangeInclusiveQuery(q.Min, q.Max, &minInc, &maxInc)
		if q.Field != "" {
			nq.SetField(q.Field)
		}
		return nq, nil

	case QueryTypeBoolean:
		bq := bleve.NewBooleanQuery()
		for _, m := range q.Must {
			sub, err := m.BuildBleveQuery()
			if err != nil {
				return nil, err
			}
			bq.AddMust(sub)
		}
		for _, s := range q.Should {
			sub, err := s.BuildBleveQuery()
			if err != nil {
				return nil, err
			}
			bq.AddShould(sub)
		}
		for _, mn := range q.MustNot {
			sub, err := mn.BuildBleveQuery()
			if err != nil {
				return nil, err
			}
			bq.AddMustNot(sub)
		}
		return bq, nil

	case QueryTypeQueryString:
		return bleve.NewQueryStringQuery(valStr), nil

	case QueryTypeMatchAll:
		return bleve.NewMatchAllQuery(), nil

	default:
		return nil, fmt.Errorf("unsupported query type: %s", typ)
	}
}

// FacetRequestDef defines facet parameters.
type FacetRequestDef struct {
	Field          string            `json:"field"`
	Size           int               `json:"size,omitempty"`
	NumericRanges  []NumericRangeDef `json:"numeric_ranges,omitempty"`
	DateTimeRanges []DateTimeRangeDef `json:"date_ranges,omitempty"`
}

// NumericRangeDef defines a range for numeric facets.
type NumericRangeDef struct {
	Name string   `json:"name"`
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
}

// DateTimeRangeDef defines a range for datetime facets.
type DateTimeRangeDef struct {
	Name  string     `json:"name"`
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// HighlightRequestDef configures snippet generation and keyword highlighting.
type HighlightRequestDef struct {
	Fields []string `json:"fields,omitempty"`
	Style  string   `json:"style,omitempty"`
}

// SearchRequest represents a high-level search request for an index.
type SearchRequest struct {
	Query     QueryDefinition            `json:"query"`
	Size      int                        `json:"size,omitempty"`
	From      int                        `json:"from,omitempty"`
	Sort      []string                   `json:"sort,omitempty"`
	Facets    map[string]FacetRequestDef `json:"facets,omitempty"`
	Highlight *HighlightRequestDef       `json:"highlight,omitempty"`
	Fields    []string                   `json:"fields,omitempty"`
	LoadDocs  bool                       `json:"load_docs,omitempty"`
}

// BuildBleveSearchRequest translates a SearchRequest into *bleve.SearchRequest.
func (sr *SearchRequest) BuildBleveSearchRequest() (*bleve.SearchRequest, error) {
	q, err := sr.Query.BuildBleveQuery()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	size := sr.Size
	if size <= 0 {
		size = 10
	}

	from := sr.From
	if from < 0 {
		from = 0
	}

	bleveReq := bleve.NewSearchRequestOptions(q, size, from, false)

	if len(sr.Sort) > 0 {
		bleveReq.SortBy(sr.Sort)
	}

	if len(sr.Fields) > 0 {
		bleveReq.Fields = sr.Fields
	} else {
		bleveReq.Fields = []string{"*"}
	}

	if sr.Highlight != nil {
		hl := bleve.NewHighlight()
		if len(sr.Highlight.Fields) > 0 {
			for _, f := range sr.Highlight.Fields {
				hl.AddField(f)
			}
		}
		bleveReq.Highlight = hl
	}

	if len(sr.Facets) > 0 {
		for name, f := range sr.Facets {
			fSize := f.Size
			if fSize <= 0 {
				fSize = 10
			}
			facetReq := bleve.NewFacetRequest(f.Field, fSize)
			for _, nr := range f.NumericRanges {
				facetReq.AddNumericRange(nr.Name, nr.Min, nr.Max)
			}
			for _, dr := range f.DateTimeRanges {
				var start, end time.Time
				if dr.Start != nil {
					start = *dr.Start
				}
				if dr.End != nil {
					end = *dr.End
				}
				facetReq.AddDateTimeRange(dr.Name, start, end)
			}
			bleveReq.AddFacet(name, facetReq)
		}
	}

	return bleveReq, nil
}

// SearchHit represents a single matched document in search results.
type SearchHit struct {
	ID        string              `json:"id"`
	Index     string              `json:"index"`
	Score     float64             `json:"score"`
	Fields    map[string]any      `json:"fields,omitempty"`
	Fragments map[string][]string `json:"fragments,omitempty"`
	Document  any                 `json:"document,omitempty"`
}

// FacetTermResult represents term counts for a facet.
type FacetTermResult struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

// FacetRangeResult represents range counts for numeric or date facets.
type FacetRangeResult struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Min   *float64 `json:"min,omitempty"`
	Max   *float64 `json:"max,omitempty"`
}

// FacetResult represents aggregations for a field.
type FacetResult struct {
	Field       string             `json:"field"`
	Total       int                `json:"total"`
	Missing     int                `json:"missing"`
	Other       int                `json:"other"`
	Terms       []FacetTermResult  `json:"terms,omitempty"`
	Ranges      []FacetRangeResult `json:"ranges,omitempty"`
}

// SearchResult is the clean JSON response returned to clients.
type SearchResult struct {
	Total      uint64                 `json:"total"`
	Took       time.Duration          `json:"took_ns"`
	TookString string                 `json:"took"`
	MaxScore   float64                `json:"max_score"`
	Hits       []SearchHit            `json:"hits"`
	Facets     map[string]FacetResult `json:"facets,omitempty"`
}

// ConvertBleveResult maps Bleve's SearchResult to our clean SearchResult.
func ConvertBleveResult(indexName string, res *bleve.SearchResult) *SearchResult {
	if res == nil {
		return &SearchResult{Hits: []SearchHit{}}
	}

	hits := make([]SearchHit, 0, len(res.Hits))
	for _, hit := range res.Hits {
		hits = append(hits, SearchHit{
			ID:        hit.ID,
			Index:     indexName,
			Score:     hit.Score,
			Fields:    hit.Fields,
			Fragments: hit.Fragments,
		})
	}

	facets := make(map[string]FacetResult)
	for name, fr := range res.Facets {
		fResult := FacetResult{
			Field:   fr.Field,
			Total:   fr.Total,
			Missing: fr.Missing,
			Other:   fr.Other,
		}
		for _, term := range fr.Terms.Terms() {
			fResult.Terms = append(fResult.Terms, FacetTermResult{
				Term:  term.Term,
				Count: term.Count,
			})
		}
		for _, nr := range fr.NumericRanges {
			fResult.Ranges = append(fResult.Ranges, FacetRangeResult{
				Name:  nr.Name,
				Count: nr.Count,
				Min:   nr.Min,
				Max:   nr.Max,
			})
		}
		facets[name] = fResult
	}

	return &SearchResult{
		Total:      res.Total,
		Took:       res.Took,
		TookString: res.Took.String(),
		MaxScore:   res.MaxScore,
		Hits:       hits,
		Facets:     facets,
	}
}
