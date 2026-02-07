package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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
)

func main() {
	level := flag.String("level", "low", "Stress level: low, medium, high")
	flag.Parse()

	var workers int
	var delay time.Duration

	switch *level {
	case "low":
		workers = 5
		delay = 100 * time.Millisecond
		fmt.Println("Running LOW stress test (5 workers, 100ms delay)")
	case "medium":
		workers = 20
		delay = 10 * time.Millisecond
		fmt.Println("Running MEDIUM stress test (20 workers, 10ms delay)")
	case "high":
		workers = 100
		delay = 0
		fmt.Println("Running HIGH stress test (100 workers, NO delay)")
	default:
		log.Fatalf("Invalid level: %s", *level)
	}

	var wg sync.WaitGroup
	start := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
		Timeout: 5 * time.Second,
	}

	// Stats printer
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			sent := atomic.LoadUint64(&requestsSent)
			errs := atomic.LoadUint64(&errorsCount)
			elapsed := time.Since(start).Seconds()
			rps := float64(sent) / elapsed
			fmt.Printf("RPS: %.2f | Total: %d | Errors: %d\n", rps, sent, errs)
		}
	}()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				sendEvent(client)
				if delay > 0 {
					time.Sleep(delay)
				}
			}
		}()
	}

	wg.Wait()
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
	resp, err := client.Post(baseURL+"/event", "application/json", bytes.NewBuffer(data))
	if err != nil {
		if atomic.AddUint64(&errorsCount, 1) == 1 {
			log.Printf("First error: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		atomic.AddUint64(&requestsSent, 1)
	} else {
		if atomic.AddUint64(&errorsCount, 1) == 1 {
			log.Printf("First error response: Status %d", resp.StatusCode)
		}
	}
}
