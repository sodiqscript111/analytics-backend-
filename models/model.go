package models

import "time"

type Event struct {
	UserId    string    `json:"user_id"`
	Action    string    `json:"action"`
	Element   string    `json:"element"`
	Duration  float64   `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}
