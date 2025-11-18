package worker

import (
	"analytics-backend/database"
	"analytics-backend/models"
	"log"
	"sync"
	"time"
)

const (
	ThresholdCount = 1000
	MaxWaitTime    = 30 * time.Second
)

var (
	workerMu      sync.Mutex
	lastFlushTime = time.Now()
)

func StartThresholdWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Starting threshold-based worker...")

	for range ticker.C {
		if shouldFlush() {
			if err := flushToDatabase(); err != nil {
				log.Printf("Error flushing to database: %v", err)
			}
		}
	}
}

func shouldFlush() bool {
	workerMu.Lock()
	defer workerMu.Unlock()

	length, err := database.CheckStreamLength(database.StreamName)
	if err != nil {
		log.Printf("Error checking stream length: %v", err)
		return false
	}

	timeSinceFlush := time.Since(lastFlushTime)

	if length >= ThresholdCount {
		log.Printf("Threshold reached: %d events in stream", length)
		return true
	}

	if length > 0 && timeSinceFlush >= MaxWaitTime {
		log.Printf("Max wait time exceeded: %d events waiting for %v", length, timeSinceFlush)
		return true
	}

	return false
}

func flushToDatabase() error {
	workerMu.Lock()
	defer workerMu.Unlock()

	result, err := database.ReadFromGroup()
	if err != nil {
		return err
	}

	if len(result) == 0 {
		return nil
	}

	log.Printf("Flushing batch of %d events to database", len(result))

	var events []models.Event
	var messageIDs []string

	for _, msg := range result {
		event := parseEvent(msg)
		events = append(events, event)
		messageIDs = append(messageIDs, msg.ID)
	}

	if err := database.BatchAddToDatabase(events); err != nil {
		return err
	}

	if err := database.AckMessage(messageIDs...); err != nil {
		return err
	}

	lastFlushTime = time.Now()
	log.Printf("Successfully flushed %d events", len(events))
	return nil
}
