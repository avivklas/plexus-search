package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
	"github.com/avivklas/plexus-search/pkg/searcher"
)

// wrapESCompat wraps the router with Elasticsearch-compatible request handling.
func (s *Server) wrapESCompat(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		path := r.URL.Path

		// 1. Root cluster info
		if (path == "/" || path == "") && r.Method == http.MethodGet {
			s.handleESInfo(w, r)
			return
		}

		// 2. Cluster health & settings
		if (path == "/_cluster/health" || strings.HasPrefix(path, "/_cluster/health/")) && r.Method == http.MethodGet {
			s.handleESClusterHealth(w, r)
			return
		}
		if (path == "/_cluster/settings" || strings.HasPrefix(path, "/_cluster/settings")) && (r.Method == http.MethodGet || r.Method == http.MethodPut) {
			writeJSON(w, http.StatusOK, map[string]any{
				"acknowledged": true,
				"persistent":   map[string]any{},
				"transient":    map[string]any{},
			})
			return
		}

		// 3. Node stats & info
		if (path == "/_nodes" || path == "/_nodes/stats" || strings.HasPrefix(path, "/_nodes/")) && r.Method == http.MethodGet {
			s.handleESNodeStats(w, r)
			return
		}

		// 4. Index stats
		if (path == "/_stats" || strings.HasSuffix(path, "/_stats")) && r.Method == http.MethodGet {
			s.handleESStats(w, r)
			return
		}

		// 5. Refresh
		if (path == "/_refresh" || strings.HasSuffix(path, "/_refresh")) && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
			s.handleESRefresh(w, r)
			return
		}

		// 6. Bulk
		if (path == "/_bulk" || strings.HasSuffix(path, "/_bulk")) && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
			s.handleESBulk(w, r)
			return
		}

		// 7. Search
		if (path == "/_search" || strings.HasSuffix(path, "/_search")) && (r.Method == http.MethodPost || r.Method == http.MethodGet) {
			s.handleESSearch(w, r)
			return
		}

		// 8. Doc GET
		if strings.Contains(path, "/_doc/") && r.Method == http.MethodGet {
			s.handleESGetDoc(w, r)
			return
		}

		// 9. Single-segment index management: HEAD /{index}, PUT /{index}, DELETE /{index}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 1 && parts[0] != "" && !strings.HasPrefix(parts[0], "_") && !strings.HasPrefix(parts[0], "api") && parts[0] != "health" {
			index := parts[0]
			switch r.Method {
			case http.MethodHead:
				s.handleESHeadIndexNamed(w, r, index)
				return
			case http.MethodPut:
				s.handleESCreateIndexNamed(w, r, index)
				return
			case http.MethodDelete:
				s.handleESDeleteIndexNamed(w, r, index)
				return
			}
		}

		// Fallback to native ServeMux
		mux.ServeHTTP(w, r)
	})
}

// handleESInfo handles GET /
func (s *Server) handleESInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":         "plexus-search",
		"cluster_name": "plexus-search",
		"cluster_uuid": "plexus-cluster-uuid",
		"version": map[string]any{
			"number":                              "8.14.0",
			"build_flavor":                        "default",
			"build_type":                          "docker",
			"build_hash":                          "8d96bbe3bf5fed931f3119733895458eab75dca9",
			"build_date":                          "2026-10-05T00:00:00Z",
			"build_snapshot":                      false,
			"lucene_version":                      "9.10.0",
			"minimum_wire_compatibility_version": "7.17.0",
			"minimum_index_compatibility_version": "7.0.0",
		},
		"tagline": "You Know, for Search",
	})
}

// handleESClusterHealth handles GET /_cluster/health
func (s *Server) handleESClusterHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"cluster_name":                     "plexus-search",
		"status":                           "green",
		"timed_out":                        false,
		"number_of_nodes":                  1,
		"number_of_data_nodes":             1,
		"active_primary_shards":            1,
		"active_shards":                    1,
		"relocating_shards":                0,
		"initializing_shards":              0,
		"unassigned_shards":                0,
		"delayed_unassigned_shards":        0,
		"number_of_pending_tasks":          0,
		"number_of_in_flight_fetch":        0,
		"task_max_waiting_in_queue_millis": 0,
		"active_shards_percent_as_number":  100.0,
	})
}

// handleESNodeStats handles GET /_nodes and /_nodes/stats
func (s *Server) handleESNodeStats(w http.ResponseWriter, r *http.Request) {
	nodeID := "node-1"
	writeJSON(w, http.StatusOK, map[string]any{
		"_nodes": map[string]any{
			"total":      1,
			"successful": 1,
			"failed":     0,
		},
		"cluster_name": "plexus-search",
		"nodes": map[string]any{
			nodeID: map[string]any{
				"name":              "plexus-search-node-1",
				"transport_address": "127.0.0.1:9000",
				"host":              "127.0.0.1",
				"ip":                "127.0.0.1",
				"version":           "8.14.0",
				"roles":             []string{"master", "data", "ingest"},
				"attributes":        map[string]any{},
				"ingest": map[string]any{
					"total": map[string]any{
						"count":          0,
						"time_in_millis": 0,
						"current":        0,
						"failed":         0,
					},
					"pipelines": map[string]any{},
				},
				"indices": map[string]any{
					"docs": map[string]any{
						"count":   1000,
						"deleted": 0,
					},
					"store": map[string]any{
						"size_in_bytes": 1048576,
					},
					"indexing": map[string]any{
						"index_total":             1000,
						"index_time_in_millis":    10,
						"index_current":           0,
						"index_failed":            0,
						"throttle_time_in_millis": 0,
					},
					"merges": map[string]any{
						"current":                        0,
						"current_docs":                   0,
						"current_size_in_bytes":          0,
						"total":                          0,
						"total_time_in_millis":           0,
						"total_throttled_time_in_millis": 0,
					},
					"refresh": map[string]any{
						"total":                1,
						"total_time_in_millis": 1,
					},
					"flush": map[string]any{
						"total":                0,
						"total_time_in_millis": 0,
					},
					"query_cache": map[string]any{
						"memory_size_in_bytes": 0,
						"hit_count":            0,
						"miss_count":           0,
					},
					"segments": map[string]any{
						"count":                         1,
						"memory_in_bytes":               1024,
						"terms_memory_in_bytes":         256,
						"stored_fields_memory_in_bytes": 256,
						"term_vectors_memory_in_bytes":  0,
						"norms_memory_in_bytes":         256,
						"points_memory_in_bytes":        256,
						"doc_values_memory_in_bytes":    256,
						"index_writer_memory_in_bytes":  0,
						"version_map_memory_in_bytes":   0,
						"fixed_bit_set_memory_in_bytes": 0,
					},
				},
				"jvm": map[string]any{
					"timestamp":        time.Now().UnixMilli(),
					"uptime_in_millis": 3600000,
					"mem": map[string]any{
						"heap_used_in_bytes":          67108864,
						"heap_used_percent":           10,
						"heap_committed_in_bytes":     134217728,
						"heap_max_in_bytes":           536870912,
						"non_heap_used_in_bytes":      33554432,
						"non_heap_committed_in_bytes": 67108864,
						"pools": map[string]any{
							"young": map[string]any{
								"used_in_bytes":      10485760,
								"max_in_bytes":       0,
								"peak_used_in_bytes": 10485760,
								"peak_max_in_bytes":  0,
							},
							"old": map[string]any{
								"used_in_bytes":      56623104,
								"max_in_bytes":       536870912,
								"peak_used_in_bytes": 56623104,
								"peak_max_in_bytes":  536870912,
							},
							"survivor": map[string]any{
								"used_in_bytes":      0,
								"max_in_bytes":       0,
								"peak_used_in_bytes": 0,
								"peak_max_in_bytes":  0,
							},
						},
					},
					"gc": map[string]any{
						"collectors": map[string]any{
							"young": map[string]any{
								"collection_count":          0,
								"collection_time_in_millis": 0,
							},
							"old": map[string]any{
								"collection_count":          0,
								"collection_time_in_millis": 0,
							},
						},
					},
				},
			},
		},
	})
}

// handleESStats handles GET /_stats and GET /{index}/_stats
func (s *Server) handleESStats(w http.ResponseWriter, r *http.Request) {
	statsObj := map[string]any{
		"docs": map[string]any{
			"count":   1000,
			"deleted": 0,
		},
		"store": map[string]any{
			"size_in_bytes":                 1048576,
			"total_data_set_size_in_bytes": 1048576,
		},
		"translog": map[string]any{
			"size_in_bytes": 1024,
		},
		"indexing": map[string]any{
			"index_total":             1000,
			"index_time_in_millis":    10,
			"index_current":           0,
			"index_failed":            0,
			"throttle_time_in_millis": 0,
		},
		"merges": map[string]any{
			"current":                        0,
			"current_docs":                   0,
			"current_size_in_bytes":          0,
			"total":                          0,
			"total_time_in_millis":           0,
			"total_throttled_time_in_millis": 0,
		},
		"refresh": map[string]any{
			"total":                1,
			"total_time_in_millis": 1,
		},
		"flush": map[string]any{
			"total":                0,
			"total_time_in_millis": 0,
		},
		"segments": map[string]any{
			"count":                         1,
			"memory_in_bytes":               1024,
			"terms_memory_in_bytes":         256,
			"stored_fields_memory_in_bytes": 256,
			"norms_memory_in_bytes":         256,
			"points_memory_in_bytes":        256,
			"doc_values_memory_in_bytes":    256,
		},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"_shards": map[string]any{
			"total":      1,
			"successful": 1,
			"failed":     0,
		},
		"_all": map[string]any{
			"primaries": statsObj,
			"total":     statsObj,
		},
		"indices": map[string]any{},
	})
}

// handleESHeadIndexNamed handles HEAD /{index}
func (s *Server) handleESHeadIndexNamed(w http.ResponseWriter, r *http.Request, index string) {
	if index == "" || strings.HasPrefix(index, "_") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	s.indicesMu.RLock()
	created := s.createdIndices[index]
	s.indicesMu.RUnlock()

	if created {
		w.WriteHeader(http.StatusOK)
		return
	}

	indexes := s.searcher.ListIndexes()
	for _, idx := range indexes {
		if idx == index {
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

// handleESCreateIndexNamed handles PUT /{index}
func (s *Server) handleESCreateIndexNamed(w http.ResponseWriter, r *http.Request, index string) {
	if index == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index is required"})
		return
	}

	s.indicesMu.Lock()
	s.createdIndices[index] = true
	s.indicesMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"acknowledged":        true,
		"shards_acknowledged": true,
		"index":               index,
	})
}

// handleESDeleteIndexNamed handles DELETE /{index}
func (s *Server) handleESDeleteIndexNamed(w http.ResponseWriter, r *http.Request, index string) {
	if index == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index is required"})
		return
	}

	s.indicesMu.Lock()
	delete(s.createdIndices, index)
	s.indicesMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"acknowledged": true,
	})
}

// handleESRefresh handles POST /{index}/_refresh and POST /_refresh
func (s *Server) handleESRefresh(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"_shards": map[string]any{
			"total":      1,
			"successful": 1,
			"failed":     0,
		},
	})
}

// handleESGetDoc handles GET /{index}/_doc/{id}
func (s *Server) handleESGetDoc(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expected format: <index>/_doc/<id>
	if len(parts) < 3 || parts[1] != "_doc" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid doc path"})
		return
	}

	index := parts[0]
	id := parts[2]

	doc, err := s.searcher.GetDoc(r.Context(), index, id)
	if err != nil {
		if errors.Is(err, searcher.ErrNotFound) || errors.Is(err, docstore.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"_index": index,
				"_id":    id,
				"found":  false,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var dataParsed any
	if len(doc.Data) > 0 {
		_ = json.Unmarshal(doc.Data, &dataParsed)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"_index":        index,
		"_id":           doc.ID,
		"_version":      doc.Revision,
		"_seq_no":       doc.Revision,
		"_primary_term": 1,
		"found":         true,
		"_source":       dataParsed,
	})
}

// handleESBulk handles POST /_bulk and POST /{index}/_bulk (NDJSON)
func (s *Server) handleESBulk(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	defaultIndex := ""
	if len(parts) >= 2 && parts[len(parts)-1] == "_bulk" {
		defaultIndex = parts[len(parts)-2]
	}

	start := time.Now()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed reading request body"})
		return
	}

	scanner := bufio.NewScanner(bytes.NewReader(bodyBytes))
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 64*1024*1024)

	var results []map[string]any
	batches := make(map[string]*searcher.AtomicBatchRequest)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var actionMap map[string]json.RawMessage
		if err := json.Unmarshal(line, &actionMap); err != nil {
			continue
		}

		for action, rawMeta := range actionMap {
			var meta struct {
				Index string `json:"_index"`
				ID    string `json:"_id"`
			}
			_ = json.Unmarshal(rawMeta, &meta)

			targetIndex := meta.Index
			if targetIndex == "" {
				targetIndex = defaultIndex
			}
			if targetIndex == "" {
				targetIndex = "default"
			}

			if batches[targetIndex] == nil {
				batches[targetIndex] = &searcher.AtomicBatchRequest{
					Index: targetIndex,
				}
			}

			docID := meta.ID

			switch action {
			case "index", "create":
				if !scanner.Scan() {
					break
				}
				docLine := scanner.Bytes()
				var docMap map[string]any
				if err := json.Unmarshal(docLine, &docMap); err == nil {
					if docID == "" {
						if idVal, ok := docMap["id"].(string); ok && idVal != "" {
							docID = idVal
						} else if idVal, ok := docMap["_id"].(string); ok && idVal != "" {
							docID = idVal
						}
					}
				}
				if docID == "" {
					docID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
				}

				docData := make([]byte, len(docLine))
				copy(docData, docLine)

				batches[targetIndex].Indexes = append(batches[targetIndex].Indexes, searcher.AtomicIndexRequest{
					Index: targetIndex,
					ID:    docID,
					Data:  docData,
				})

				results = append(results, map[string]any{
					"index": map[string]any{
						"_index":   targetIndex,
						"_id":      docID,
						"_version": 1,
						"result":   "created",
						"_shards": map[string]any{
							"total":      1,
							"successful": 1,
							"failed":     0,
						},
						"status": http.StatusCreated,
					},
				})

			case "delete":
				batches[targetIndex].Deletes = append(batches[targetIndex].Deletes, searcher.AtomicDeleteRequest{
					Index: targetIndex,
					ID:    docID,
				})

				results = append(results, map[string]any{
					"delete": map[string]any{
						"_index":   targetIndex,
						"_id":      docID,
						"_version": 1,
						"result":   "deleted",
						"_shards": map[string]any{
							"total":      1,
							"successful": 1,
							"failed":     0,
						},
						"status": http.StatusOK,
					},
				})

			case "update":
				if !scanner.Scan() {
					break
				}
				docLine := scanner.Bytes()
				var updateWrapper struct {
					Doc json.RawMessage `json:"doc"`
				}
				dataToUse := docLine
				if err := json.Unmarshal(docLine, &updateWrapper); err == nil && len(updateWrapper.Doc) > 0 {
					dataToUse = updateWrapper.Doc
				}

				docData := make([]byte, len(dataToUse))
				copy(docData, dataToUse)

				batches[targetIndex].Indexes = append(batches[targetIndex].Indexes, searcher.AtomicIndexRequest{
					Index: targetIndex,
					ID:    docID,
					Data:  docData,
				})

				results = append(results, map[string]any{
					"update": map[string]any{
						"_index":   targetIndex,
						"_id":      docID,
						"_version": 1,
						"result":   "updated",
						"_shards": map[string]any{
							"total":      1,
							"successful": 1,
							"failed":     0,
						},
						"status": http.StatusOK,
					},
				})
			}
		}
	}

	for _, batchReq := range batches {
		if len(batchReq.Indexes) > 0 || len(batchReq.Deletes) > 0 {
			_, err := s.searcher.Batch(r.Context(), *batchReq)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
	}

	tookMs := time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, map[string]any{
		"took":   tookMs,
		"errors": false,
		"items":  results,
	})
}

// handleESSearch handles POST /{index}/_search, GET /{index}/_search, etc.
func (s *Server) handleESSearch(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	index := ""
	if len(parts) >= 2 && parts[len(parts)-1] == "_search" {
		index = parts[len(parts)-2]
	}
	if index == "" || index == "_search" {
		indexes := s.searcher.ListIndexes()
		if len(indexes) > 0 {
			index = indexes[0]
		} else {
			index = "default"
		}
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	var esReq map[string]any
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &esReq)
	}

	searchReq := indexstore.SearchRequest{
		Size:     10,
		From:     0,
		LoadDocs: true,
	}

	if esReq != nil {
		if sizeVal, ok := esReq["size"].(float64); ok {
			searchReq.Size = int(sizeVal)
		}
		if fromVal, ok := esReq["from"].(float64); ok {
			searchReq.From = int(fromVal)
		}
		if qRaw, ok := esReq["query"].(map[string]any); ok {
			searchReq.Query = parseESQuery(qRaw)
		}
	} else {
		searchReq.Query = indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll}
	}

	res, err := s.searcher.Search(r.Context(), index, searchReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed: " + err.Error()})
		return
	}

	hits := make([]map[string]any, 0, len(res.Hits))
	for _, hit := range res.Hits {
		hits = append(hits, map[string]any{
			"_index":  hit.Index,
			"_id":     hit.ID,
			"_score":  hit.Score,
			"_source": hit.Document,
		})
	}

	tookMs := int(res.Took.Milliseconds())
	if tookMs == 0 && res.Took > 0 {
		tookMs = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"took":      tookMs,
		"timed_out": false,
		"_shards": map[string]any{
			"total":      1,
			"successful": 1,
			"skipped":    0,
			"failed":     0,
		},
		"hits": map[string]any{
			"total": map[string]any{
				"value":    res.Total,
				"relation": "eq",
			},
			"max_score": res.MaxScore,
			"hits":      hits,
		},
	})
}

// parseESQuery parses Elasticsearch Query DSL into indexstore.QueryDefinition.
func parseESQuery(q map[string]any) indexstore.QueryDefinition {
	if len(q) == 0 {
		return indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll}
	}

	// 1. match_all
	if _, ok := q["match_all"]; ok {
		return indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll}
	}

	// 2. match
	if matchMap, ok := q["match"].(map[string]any); ok {
		for f, v := range matchMap {
			if vMap, isMap := v.(map[string]any); isMap {
				return indexstore.QueryDefinition{
					Type:  indexstore.QueryTypeMatch,
					Field: f,
					Value: vMap["query"],
				}
			}
			return indexstore.QueryDefinition{
				Type:  indexstore.QueryTypeMatch,
				Field: f,
				Value: v,
			}
		}
	}

	// 3. match_phrase
	if mpMap, ok := q["match_phrase"].(map[string]any); ok {
		for f, v := range mpMap {
			if vMap, isMap := v.(map[string]any); isMap {
				return indexstore.QueryDefinition{
					Type:  indexstore.QueryTypeMatchPhrase,
					Field: f,
					Value: vMap["query"],
				}
			}
			return indexstore.QueryDefinition{
				Type:  indexstore.QueryTypeMatchPhrase,
				Field: f,
				Value: v,
			}
		}
	}

	// 4. term
	if termMap, ok := q["term"].(map[string]any); ok {
		for f, v := range termMap {
			cleanField := strings.TrimSuffix(f, ".raw")
			cleanField = strings.TrimSuffix(cleanField, ".keyword")
			if vMap, isMap := v.(map[string]any); isMap {
				return indexstore.QueryDefinition{
					Type:  indexstore.QueryTypeTerm,
					Field: cleanField,
					Value: vMap["value"],
				}
			}
			return indexstore.QueryDefinition{
				Type:  indexstore.QueryTypeTerm,
				Field: cleanField,
				Value: v,
			}
		}
	}

	// 5. prefix
	if prefixMap, ok := q["prefix"].(map[string]any); ok {
		for f, v := range prefixMap {
			if vMap, isMap := v.(map[string]any); isMap {
				return indexstore.QueryDefinition{
					Type:  indexstore.QueryTypePrefix,
					Field: f,
					Value: vMap["value"],
				}
			}
			return indexstore.QueryDefinition{
				Type:  indexstore.QueryTypePrefix,
				Field: f,
				Value: v,
			}
		}
	}

	// 6. fuzzy
	if fuzzyMap, ok := q["fuzzy"].(map[string]any); ok {
		for f, v := range fuzzyMap {
			return indexstore.QueryDefinition{
				Type:  indexstore.QueryTypeFuzzy,
				Field: f,
				Value: v,
			}
		}
	}

	// 7. range
	if rangeMap, ok := q["range"].(map[string]any); ok {
		for f, v := range rangeMap {
			if bounds, ok := v.(map[string]any); ok {
				var minVal, maxVal *float64
				minInc := true
				maxInc := true

				if gte, ok := bounds["gte"].(float64); ok {
					minVal = &gte
					minInc = true
				} else if gt, ok := bounds["gt"].(float64); ok {
					minVal = &gt
					minInc = false
				}

				if lte, ok := bounds["lte"].(float64); ok {
					maxVal = &lte
					maxInc = true
				} else if lt, ok := bounds["lt"].(float64); ok {
					maxVal = &lt
					maxInc = false
				}

				return indexstore.QueryDefinition{
					Type:         indexstore.QueryTypeNumericRange,
					Field:        f,
					Min:          minVal,
					Max:          maxVal,
					MinInclusive: &minInc,
					MaxInclusive: &maxInc,
				}
			}
		}
	}

	// 8. bool
	if boolMap, ok := q["bool"].(map[string]any); ok {
		boolDef := indexstore.QueryDefinition{
			Type: indexstore.QueryTypeBoolean,
		}

		if mustList, ok := boolMap["must"].([]any); ok {
			for _, item := range mustList {
				if itemMap, ok := item.(map[string]any); ok {
					boolDef.Must = append(boolDef.Must, parseESQuery(itemMap))
				}
			}
		}

		if filterList, ok := boolMap["filter"].([]any); ok {
			for _, item := range filterList {
				if itemMap, ok := item.(map[string]any); ok {
					boolDef.Must = append(boolDef.Must, parseESQuery(itemMap))
				}
			}
		}

		if shouldList, ok := boolMap["should"].([]any); ok {
			for _, item := range shouldList {
				if itemMap, ok := item.(map[string]any); ok {
					boolDef.Should = append(boolDef.Should, parseESQuery(itemMap))
				}
			}
		}

		if mustNotList, ok := boolMap["must_not"].([]any); ok {
			for _, item := range mustNotList {
				if itemMap, ok := item.(map[string]any); ok {
					boolDef.MustNot = append(boolDef.MustNot, parseESQuery(itemMap))
				}
			}
		}

		return boolDef
	}

	return indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll}
}
