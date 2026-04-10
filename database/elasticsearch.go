package database

import (
	"analytics-backend/config"
	"analytics-backend/models"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultElasticsearchIndex = "events-search"

type ElasticsearchClient struct {
	baseURL    string
	index      string
	username   string
	password   string
	httpClient *http.Client
}

type SearchEventsParams struct {
	Query  string
	Action string
	UserID string
	From   *time.Time
	To     *time.Time
	Size   int
	Cursor string
}

type SearchEventsResponse struct {
	Items      []models.Event `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
	Total      int64          `json:"total"`
	TookMS     int            `json:"took_ms"`
	Source     string         `json:"source"`
}

type searchCursor struct {
	Timestamp string `json:"timestamp"`
	ID        int64  `json:"id"`
}

var ES *ElasticsearchClient

func InitElasticsearch(cfg config.ElasticsearchConfig) error {
	indexName := strings.TrimSpace(cfg.Index)
	if indexName == "" {
		indexName = DefaultElasticsearchIndex
	}

	ES = &ElasticsearchClient{
		baseURL:  strings.TrimRight(cfg.Addr, "/"),
		index:    indexName,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if ES.baseURL == "" {
		return fmt.Errorf("elasticsearch addr is required")
	}

	if err := ES.ping(context.Background()); err != nil {
		return err
	}

	return ES.ensureIndex(context.Background())
}

func BulkIndexEvents(ctx context.Context, events []models.Event) error {
	if ES == nil || len(events) == 0 {
		return nil
	}
	return ES.bulkIndexEvents(ctx, events)
}

func SearchEvents(ctx context.Context, params SearchEventsParams) (*SearchEventsResponse, error) {
	if ES == nil {
		return nil, fmt.Errorf("elasticsearch client not initialized")
	}
	return ES.searchEvents(ctx, params)
}

func BackfillEventsToElasticsearch(ctx context.Context, batchSize int) error {
	if ES == nil {
		return fmt.Errorf("elasticsearch client not initialized")
	}

	return FindEventsInBatches(batchSize, func(events []models.Event) error {
		return BulkIndexEvents(ctx, events)
	})
}

func encodeSearchCursor(timestamp time.Time, id int64) (string, error) {
	payload, err := json.Marshal(searchCursor{
		Timestamp: timestamp.UTC().Format(time.RFC3339Nano),
		ID:        id,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeSearchCursor(cursor string) (*searchCursor, error) {
	if strings.TrimSpace(cursor) == "" {
		return nil, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}

	var decoded searchCursor
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}

	if decoded.Timestamp == "" {
		return nil, fmt.Errorf("cursor timestamp is required")
	}

	return &decoded, nil
}

func buildSearchEventsRequest(params SearchEventsParams) (map[string]any, error) {
	size := params.Size
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	filterClauses := make([]any, 0, 4)
	if params.Action != "" {
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{
				"action.keyword": params.Action,
			},
		})
	}
	if params.UserID != "" {
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{
				"user_id.keyword": params.UserID,
			},
		})
	}
	if params.From != nil || params.To != nil {
		rangeQuery := map[string]any{}
		if params.From != nil {
			rangeQuery["gte"] = params.From.UTC().Format(time.RFC3339Nano)
		}
		if params.To != nil {
			rangeQuery["lte"] = params.To.UTC().Format(time.RFC3339Nano)
		}
		filterClauses = append(filterClauses, map[string]any{
			"range": map[string]any{
				"timestamp": rangeQuery,
			},
		})
	}

	var query any = map[string]any{"match_all": map[string]any{}}
	if params.Query != "" || len(filterClauses) > 0 {
		boolQuery := map[string]any{}

		if params.Query != "" {
			textSearch := map[string]any{
				"multi_match": map[string]any{
					"query":  params.Query,
					"fields": []string{"user_id", "action", "element"},
				},
			}

			if idValue, err := strconv.ParseInt(params.Query, 10, 64); err == nil {
				boolQuery["should"] = []any{
					textSearch,
					map[string]any{
						"term": map[string]any{
							"id": idValue,
						},
					},
				}
				boolQuery["minimum_should_match"] = 1
			} else {
				boolQuery["must"] = []any{textSearch}
			}
		}

		if len(filterClauses) > 0 {
			boolQuery["filter"] = filterClauses
		}

		query = map[string]any{"bool": boolQuery}
	}

	request := map[string]any{
		"size":             size,
		"track_total_hits": true,
		"query":            query,
		"sort": []any{
			map[string]any{"timestamp": map[string]any{"order": "desc"}},
			map[string]any{"id": map[string]any{"order": "desc"}},
		},
	}

	cursor, err := decodeSearchCursor(params.Cursor)
	if err != nil {
		return nil, err
	}
	if cursor != nil {
		request["search_after"] = []any{cursor.Timestamp, cursor.ID}
	}

	return request, nil
}

func (c *ElasticsearchClient) ping(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch ping failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *ElasticsearchClient) ensureIndex(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodHead, "/"+c.index, nil, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to check index %s: %s", c.index, resp.Status)
	}

	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"id": map[string]any{
					"type": "long",
				},
				"user_id": map[string]any{
					"type": "text",
					"fields": map[string]any{
						"keyword": map[string]any{"type": "keyword"},
					},
				},
				"action": map[string]any{
					"type": "text",
					"fields": map[string]any{
						"keyword": map[string]any{"type": "keyword"},
					},
				},
				"element": map[string]any{
					"type": "text",
					"fields": map[string]any{
						"keyword": map[string]any{"type": "keyword"},
					},
				},
				"duration": map[string]any{
					"type": "float",
				},
				"timestamp": map[string]any{
					"type": "date",
				},
			},
		},
	}

	payload, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	createResp, err := c.doRequest(ctx, http.MethodPut, "/"+c.index, bytes.NewReader(payload), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	defer createResp.Body.Close()

	if createResp.StatusCode >= 300 {
		body, _ := io.ReadAll(createResp.Body)
		return fmt.Errorf("failed to create elasticsearch index: %s", strings.TrimSpace(string(body)))
	}

	return nil
}

func (c *ElasticsearchClient) bulkIndexEvents(ctx context.Context, events []models.Event) (err error) {
	started := time.Now()
	defer func() {
		observeDBOperation("elasticsearch", "bulk_index", c.index, started, err)
	}()

	var payload bytes.Buffer

	for _, event := range events {
		actionMeta := map[string]any{
			"index": map[string]any{
				"_index": c.index,
				"_id":    strconv.FormatInt(event.ID, 10),
			},
		}
		metaBytes, err := json.Marshal(actionMeta)
		if err != nil {
			return err
		}

		doc := map[string]any{
			"id":        event.ID,
			"user_id":   event.UserId,
			"action":    event.Action,
			"element":   event.Element,
			"duration":  event.Duration,
			"timestamp": event.Timestamp.UTC().Format(time.RFC3339Nano),
		}
		docBytes, err := json.Marshal(doc)
		if err != nil {
			return err
		}

		payload.Write(metaBytes)
		payload.WriteByte('\n')
		payload.Write(docBytes)
		payload.WriteByte('\n')
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/_bulk", &payload, map[string]string{
		"Content-Type": "application/x-ndjson",
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("bulk index request failed: %s", strings.TrimSpace(string(body)))
		return err
	}

	var result struct {
		Errors bool `json:"errors"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.Errors {
		err = fmt.Errorf("bulk index completed with item errors")
		return err
	}

	return nil
}

func (c *ElasticsearchClient) searchEvents(ctx context.Context, params SearchEventsParams) (response *SearchEventsResponse, err error) {
	started := time.Now()
	defer func() {
		observeDBOperation("elasticsearch", "search", c.index, started, err)
	}()

	requestedSize := params.Size
	if requestedSize <= 0 {
		requestedSize = 20
	}
	if requestedSize > 100 {
		requestedSize = 100
	}

	requestBody, err := buildSearchEventsRequest(params)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/"+c.index+"/_search", bytes.NewReader(payload), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("search request failed: %s", strings.TrimSpace(string(body)))
		return nil, err
	}

	var result struct {
		Took int `json:"took"`
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source models.Event `json:"_source"`
				Sort   []any        `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]models.Event, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		items = append(items, hit.Source)
	}

	nextCursor := ""
	if len(result.Hits.Hits) == requestedSize {
		last := result.Hits.Hits[len(result.Hits.Hits)-1]
		if len(last.Sort) >= 2 {
			sortTimestamp, ok := last.Sort[0].(string)
			if ok {
				idValue, err := anyToInt64(last.Sort[1])
				if err == nil {
					timestamp, err := time.Parse(time.RFC3339Nano, sortTimestamp)
					if err == nil {
						nextCursor, _ = encodeSearchCursor(timestamp, idValue)
					}
				}
			}
		}
	}

	response = &SearchEventsResponse{
		Items:      items,
		NextCursor: nextCursor,
		Total:      result.Hits.Total.Value,
		TookMS:     result.Took,
		Source:     "elasticsearch",
	}
	return response, nil
}

func (c *ElasticsearchClient) doRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	requestURL, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	return c.httpClient.Do(req)
}

func anyToInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric sort value %T", value)
	}
}
