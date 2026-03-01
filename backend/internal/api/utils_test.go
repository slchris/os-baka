package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseIDParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		paramVal string
		wantID   int
		wantOK   bool
		wantCode int
	}{
		{"valid ID", "1", 1, true, 0},
		{"valid large ID", "999", 999, true, 0},
		{"zero ID", "0", 0, false, http.StatusBadRequest},
		{"negative ID", "-1", 0, false, http.StatusBadRequest},
		{"non-numeric", "abc", 0, false, http.StatusBadRequest},
		{"empty string", "", 0, false, http.StatusBadRequest},
		{"float", "1.5", 0, false, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.paramVal}}

			gotID, gotOK := ParseIDParam(c, "id")

			if gotOK != tt.wantOK {
				t.Errorf("ParseIDParam() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotOK && gotID != tt.wantID {
				t.Errorf("ParseIDParam() id = %v, want %v", gotID, tt.wantID)
			}
			if !gotOK && w.Code != tt.wantCode {
				t.Errorf("ParseIDParam() status = %v, want %v", w.Code, tt.wantCode)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorResponse(c, http.StatusNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("ErrorResponse() status = %v, want %v", w.Code, http.StatusNotFound)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("ErrorResponse() body is empty")
	}
}
