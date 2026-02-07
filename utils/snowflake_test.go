package utils

import (
	"testing"
	"time"
)

func TestInitSnowflake(t *testing.T) {
	// Initialize snowflake node
	InitSnowflake(1)

	// Generate an ID
	id := GenerateID()

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}
}

func TestGenerateID_UniqueAndIncreasing(t *testing.T) {
	InitSnowflake(1)

	id1 := GenerateID()
	time.Sleep(1 * time.Millisecond) // Ensure time moves forward slightly
	id2 := GenerateID()

	if id1 == id2 {
		t.Errorf("Generated IDs are not unique: %d", id1)
	}

	if id2 <= id1 {
		t.Errorf("IDs are not increasing: %d should be greater than %d", id2, id1)
	}
}

func TestGenerateID_Concurrent(t *testing.T) {
	InitSnowflake(1)

	count := 1000
	ids := make(chan int64, count)

	for i := 0; i < count; i++ {
		go func() {
			ids <- GenerateID()
		}()
	}

	idMap := make(map[int64]bool)
	for i := 0; i < count; i++ {
		id := <-ids
		if idMap[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		idMap[id] = true
	}
}
