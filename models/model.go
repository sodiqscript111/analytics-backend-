package models

import "time"

type Event struct {
	ID        int64     `json:"id,omitempty"`
	UserId    string    `json:"user_id"`
	Action    string    `json:"action"`
	Element   string    `json:"element"`
	Duration  float64   `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}

type AggregatedEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Action    string    `json:"action" gorm:"index;size:100"`
	Element   string    `json:"element" gorm:"index;size:100"`
	Count     int       `json:"count" gorm:"default:1"`
	Window    time.Time `json:"window" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}

type UserEventMap struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	AggregatedEventID uint   `json:"aggregated_event_id" gorm:"index"`
	UserID            string `json:"user_id" gorm:"index;size:255"`
}
