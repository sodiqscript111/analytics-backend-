package handlers

import (
	"analytics-backend/database"
	"io"

	"github.com/gin-gonic/gin"
)

func GetEventsStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Create a new Redis PubSub client for this connection
	// We need a dedicated connection for Subscribe
	pubsub := database.Rdb.Subscribe(c.Request.Context(), "events:stream")
	defer pubsub.Close()

	// Wait for confirmation that subscription is created
	_, err := pubsub.Receive(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to subscribe to events"})
		return
	}

	ch := pubsub.Channel()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("message", msg.Payload)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
