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

	req, _ := http.NewRequest("POST", "/event", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
