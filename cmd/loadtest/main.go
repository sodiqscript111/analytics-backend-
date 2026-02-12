package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Event struct {
	UserId    string    `json:"user_id"`
	Action    string    `json:"action"`
	Element   string    `json:"element"`
	Duration  float64   `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	baseURL      = "http://localhost:8080"
	requestsSent uint64
	errorsCount  uint64
	latencies    []time.Duration
	latencyMu    sync.Mutex
)

func main() {
	level := flag.String("level", "low", "Stress level: low, medium, high")
	duration := flag.Duration("duration", 10*time.Second, "Duration of the test")
	output := flag.String("output", "stress_test_report.md", "Output file for the report")
	workersFlag := flag.Int("workers", 0, "Number of workers (overrides level)")
	delayFlag := flag.Duration("delay", -1, "Delay between requests (overrides level)")
	flag.Parse()

	var workers int
	var delay time.Duration

	switch *level {
	case "low":
		workers = 5
		delay = 100 * time.Millisecond
	case "medium":
		workers = 20
		delay = 10 * time.Millisecond
	case "high":
		workers = 100
		delay = 0
	default:
		log.Fatalf("Invalid level: %s", *level)
	}

	if *workersFlag > 0 {
		workers = *workersFlag
	}
	if *delayFlag >= 0 {
		delay = *delayFlag
	}

	fmt.Printf("Running stress test with %d workers and %s delay for %s\n", workers, delay, *duration)

	var wg sync.WaitGroup
	start := time.Now()
	done := make(chan struct{})

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
		Timeout: 5 * time.Second,
	}

	// Stop timer
	go func() {
		time.Sleep(*duration)
		close(done)
	}()

	// Stats printer
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				sent := atomic.LoadUint64(&requestsSent)
				errs := atomic.LoadUint64(&errorsCount)
				elapsed := time.Since(start).Seconds()
				rps := float64(sent) / elapsed
				fmt.Printf("RPS: %.2f | Total: %d | Errors: %d\n", rps, sent, errs)
			}
		}
	}()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					sendEvent(client)
					if delay > 0 {
						time.Sleep(delay)
					}
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	generateReport(*output, *level, workers, elapsed)
}

func sendEvent(client *http.Client) {
	actions := []string{"click", "view", "scroll", "hover"}
	elements := []string{"button_a", "banner_b", "link_c", "image_d"}

	event := Event{
		UserId:    fmt.Sprintf("user_%d", rand.Intn(1000)),
		Action:    actions[rand.Intn(len(actions))],
		Element:   elements[rand.Intn(len(elements))],
		Duration:  rand.Float64() * 5.0,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(event)
	startReq := time.Now()
	resp, err := client.Post(baseURL+"/event", "application/json", bytes.NewBuffer(data))
	latency := time.Since(startReq)

	if err != nil {
		atomic.AddUint64(&errorsCount, 1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		atomic.AddUint64(&requestsSent, 1)
		latencyMu.Lock()
		latencies = append(latencies, latency)
		latencyMu.Unlock()
	} else {
		atomic.AddUint64(&errorsCount, 1)
	}
}

func generateReport(filename, level string, workers int, elapsed time.Duration) {
	sent := atomic.LoadUint64(&requestsSent)
	errs := atomic.LoadUint64(&errorsCount)
	rps := float64(sent) / elapsed.Seconds()

	latencyMu.Lock()
	defer latencyMu.Unlock()
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var p50, p95, p99 time.Duration
	if len(latencies) > 0 {
		p50 = latencies[len(latencies)*50/100]
		p95 = latencies[len(latencies)*95/100]
		p99 = latencies[len(latencies)*99/100]
	}

	report := fmt.Sprintf(`
# Stress Test Report

**Date:** %s
**Level:** %s
**Workers:** %d
**Duration:** %s

## Summary
- **Total Requests:** %d
- **Successful Requests:** %d
- **Failed Requests:** %d
- **Requests Per Second (RPS):** %.2f

## Latency Metrics
- **P50 Latency:** %v
- **P95 Latency:** %v
- **P99 Latency:** %v
`, time.Now().Format(time.RFC1123), level, workers, elapsed, sent+errs, sent, errs, rps, p50, p95, p99)

	err := os.WriteFile(filename, []byte(report), 0644)
	if err != nil {
		log.Printf("Failed to write report: %v", err)
	} else {
		fmt.Printf("\nReport generated: %s\n", filename)
	}
}
