package database

import (
	"testing"
	"time"
)

func TestEncodeDecodeSearchCursor(t *testing.T) {
	now := time.Date(2026, 4, 9, 12, 0, 0, 123000000, time.UTC)

	cursor, err := encodeSearchCursor(now, 12345)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	decoded, err := decodeSearchCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}

	if decoded.ID != 12345 {
		t.Fatalf("expected id 12345, got %d", decoded.ID)
	}
	if decoded.Timestamp != now.Format(time.RFC3339Nano) {
		t.Fatalf("expected timestamp %s, got %s", now.Format(time.RFC3339Nano), decoded.Timestamp)
	}
}

func TestBuildSearchEventsRequestIncludesFiltersAndCursor(t *testing.T) {
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 9, 23, 59, 59, 0, time.UTC)
	cursor, err := encodeSearchCursor(time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC), 99)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	request, err := buildSearchEventsRequest(SearchEventsParams{
		Query:  "button",
		Action: "click",
		UserID: "user-1",
		From:   &from,
		To:     &to,
		Size:   25,
		Cursor: cursor,
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if request["size"].(int) != 25 {
		t.Fatalf("expected size 25, got %v", request["size"])
	}

	searchAfter, ok := request["search_after"].([]any)
	if !ok || len(searchAfter) != 2 {
		t.Fatalf("expected search_after with 2 values, got %#v", request["search_after"])
	}

	query, ok := request["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query map, got %#v", request["query"])
	}

	boolQuery, ok := query["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool query, got %#v", query)
	}

	filterClauses, ok := boolQuery["filter"].([]any)
	if !ok || len(filterClauses) != 3 {
		t.Fatalf("expected 3 filter clauses, got %#v", boolQuery["filter"])
	}
}

func TestBuildSearchEventsRequestDefaultsToMatchAll(t *testing.T) {
	request, err := buildSearchEventsRequest(SearchEventsParams{})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	query, ok := request["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query map, got %#v", request["query"])
	}

	if _, ok := query["match_all"]; !ok {
		t.Fatalf("expected match_all query, got %#v", query)
	}
}

func TestBuildSearchEventsRequestSupportsNumericIDSearch(t *testing.T) {
	request, err := buildSearchEventsRequest(SearchEventsParams{Query: "12345"})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	query := request["query"].(map[string]any)
	boolQuery := query["bool"].(map[string]any)

	if _, ok := boolQuery["must"]; ok {
		t.Fatalf("expected numeric id search to avoid must-only text query, got %#v", boolQuery)
	}

	shouldClauses, ok := boolQuery["should"].([]any)
	if !ok || len(shouldClauses) != 2 {
		t.Fatalf("expected 2 should clauses, got %#v", boolQuery["should"])
	}
}
