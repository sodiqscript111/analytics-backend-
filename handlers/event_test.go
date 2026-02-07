package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetEvent_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/event", GetEvent)

	// Test with invalid JSON
	req, _ := http.NewRequest("POST", "/event", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Note: TestGetEvent_Success requires mocking database.AddToStreamWithContext
// which is currently a direct call to a global function.
// For now, we verify that invalid input is handled correctly.
