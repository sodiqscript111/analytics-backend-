package handlers

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"sync"

	"github.com/gin-gonic/gin"
)

type AnalyticsResult struct {
	ActionCounts map[string]int `json:"action_counts"`
	AvgDuration  float64        `json:"avg_duration"`
	TotalEvents  int            `json:"total_events"`
	Processing   string         `json:"processing_type"`
}

const FetchLimit = 1000000

func GetAnalyticsSequential(c *gin.Context) {
	events, err := database.GetEvents(FetchLimit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	counts := make(map[string]int)
	var totalDuration float64

	for _, e := range events {
		counts[e.Action]++
		totalDuration += e.Duration
	}

	avgDuration := 0.0
	if len(events) > 0 {
		avgDuration = totalDuration / float64(len(events))
	}

	c.JSON(200, AnalyticsResult{
		ActionCounts: counts,
		AvgDuration:  avgDuration,
		TotalEvents:  len(events),
		Processing:   "sequential",
	})
}

func GetAnalyticsMapReduce(c *gin.Context) {
	events, err := database.GetEvents(FetchLimit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(events) == 0 {
		c.JSON(200, AnalyticsResult{
			ActionCounts: make(map[string]int),
			AvgDuration:  0,
			TotalEvents:  0,
			Processing:   "mapreduce",
		})
		return
	}

	numWorkers := 8
	chunkSize := (len(events) + numWorkers - 1) / numWorkers

	type partialResult struct {
		counts   map[string]int
		duration float64
	}

	resultsChan := make(chan partialResult, numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize
		if end > len(events) {
			end = len(events)
		}

		wg.Add(1)
		go func(chunk []models.Event) {
			defer wg.Done()
			localCounts := make(map[string]int)
			localDuration := 0.0

			for _, e := range chunk {
				localCounts[e.Action]++
				localDuration += e.Duration
			}
			resultsChan <- partialResult{counts: localCounts, duration: localDuration}
		}(events[i:end])
	}

	wg.Wait()
	close(resultsChan)

	finalCounts := make(map[string]int)
	totalDuration := 0.0

	for res := range resultsChan {
		for action, count := range res.counts {
			finalCounts[action] += count
		}
		totalDuration += res.duration
	}

	c.JSON(200, AnalyticsResult{
		ActionCounts: finalCounts,
		AvgDuration:  totalDuration / float64(len(events)),
		TotalEvents:  len(events),
		Processing:   "mapreduce",
	})
}
