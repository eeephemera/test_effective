package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/sirupsen/logrus"
)

func TestCreateHandler_InvalidUserID(t *testing.T) {
	h := NewHandler(nil, logrus.New())
	body := map[string]interface{}{
		"service_name": "X",
		"price": 100,
		"user_id": "not-a-uuid",
		"start_date": "07-2025",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/subscriptions/", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	h.Create(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateHandler_InvalidStartDate(t *testing.T) {
	h := NewHandler(nil, logrus.New())
	body := map[string]interface{}{
		"service_name": "X",
		"price": 100,
		"user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
		"start_date": "2025-07",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/subscriptions/", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	h.Create(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
