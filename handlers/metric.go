package handlers

import (
	"analytics-backend/database"
	"github.com/gin-gonic/gin"
)

func FetchEvents(c *gin.Context) {
	events, err := database.GetEvents(50) // get latest 50 events
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"events": events})
}
