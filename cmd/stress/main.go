package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL     = "http://localhost:8080"
	totalEvents = 5000
	concurrency = 50
)

var (
	actions  = []string{"click", "scroll", "hover", "submit", "navigate", "focus", "blur", "keypress"}
	elements = []string{"button", "link", "input", "form", "nav", "header", "footer", "sidebar", "card", "modal"}
	userIDs  = []string{"user_001", "user_002", "user_003", "user_004", "user_005", "user_006", "user_007", "user_008", "user_009", "user_010"}
)

type Event struct {
	UserID   string  `json:"user_id"`
	Action   string  `json:"action"`
	Element  string  `json:"element"`
	Duration float64 `json:"duration"`
}

func randomEvent() Event {
	return Event{
		UserID:   userIDs[rand.Intn(len(userIDs))],
		Action:   actions[rand.Intn(len(actions))],
		Element:  elements[rand.Intn(len(elements))],
		Duration: rand.Float64() * 5000,
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("==============================================")
	fmt.Println("    Analytics Backend Stress Test              ")
	fmt.Println("==============================================")
	fmt.Printf("  Events: %d | Concurrency: %d\n", totalEvents, concurrency)
	fmt.Println("==============================================")
	fmt.Println()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	var (
		successCount int64
		failCount    int64
		totalLatency int64
		minLatency   int64 = 1<<63 - 1
		maxLatency   int64
	)

	// ---- Phase 1: Ingest Events ----
	fmt.Println("Phase 1: Ingesting events...")
	start := time.Now()

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < totalEvents; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			event := randomEvent()
			body, _ := json.Marshal(event)

			reqStart := time.Now()
			resp, err := client.Post(baseURL+"/event", "application/json", bytes.NewReader(body))
			latency := time.Since(reqStart).Microseconds()

			if err != nil {
				atomic.AddInt64(&failCount, 1)
				return
			}
			resp.Body.Close()

			if resp.StatusCode == 202 {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}

			atomic.AddInt64(&totalLatency, latency)

			for {
				old := atomic.LoadInt64(&minLatency)
				if latency >= old || atomic.CompareAndSwapInt64(&minLatency, old, latency) {
					break
				}
			}
			for {
				old := atomic.LoadInt64(&maxLatency)
				if latency <= old || atomic.CompareAndSwapInt64(&maxLatency, old, latency) {
					break
				}
			}
		}()
	}

	wg.Wait()
	ingestDuration := time.Since(start)

	success := atomic.LoadInt64(&successCount)
	fail := atomic.LoadInt64(&failCount)
	avgLatency := float64(0)
	if success > 0 {
		avgLatency = float64(atomic.LoadInt64(&totalLatency)) / float64(success) / 1000.0
	}
	rps := float64(success) / ingestDuration.Seconds()

	fmt.Println()
	fmt.Println("---------- INGEST RESULTS ----------")
	fmt.Printf("  Success:     %d\n", success)
	fmt.Printf("  Failed:      %d\n", fail)
	fmt.Printf("  Total Time:  %s\n", ingestDuration.Round(time.Millisecond))
	fmt.Printf("  Throughput:  %.0f req/s\n", rps)
	fmt.Printf("  Avg Latency: %.2f ms\n", avgLatency)
	fmt.Printf("  Min Latency: %.2f ms\n", float64(atomic.LoadInt64(&minLatency))/1000.0)
	fmt.Printf("  Max Latency: %.2f ms\n", float64(atomic.LoadInt64(&maxLatency))/1000.0)
	fmt.Println("------------------------------------")

	// ---- Phase 2: Query Endpoints ----
	fmt.Println()
	fmt.Println("Phase 2: Querying analytics endpoints...")
	fmt.Println()

	endpoints := []struct {
		name string
		url  string
	}{
		{"GET /events", baseURL + "/events"},
		{"GET /events/recent", baseURL + "/events/recent"},
		{"GET /analytics/clickhouse", baseURL + "/analytics/clickhouse"},
		{"GET /analytics/sequential", baseURL + "/analytics/sequential"},
		{"GET /analytics/mapreduce", baseURL + "/analytics/mapreduce"},
	}

	fmt.Println("---------- QUERY RESULTS ----------")

	for _, ep := range endpoints {
		qStart := time.Now()
		resp, err := client.Get(ep.url)
		qLatency := time.Since(qStart)
		if err != nil {
			fmt.Printf("  FAIL %-30s  ERROR: %v\n", ep.name, err)
			continue
		}
		resp.Body.Close()
		fmt.Printf("  OK   %-30s  %s\n", ep.name, qLatency.Round(time.Microsecond))
	}

	fmt.Println("------------------------------------")
	fmt.Println()
	fmt.Println("Done!")
}
