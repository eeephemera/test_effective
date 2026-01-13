package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/effectivemobile/subscriptions/internal/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// mock repository
type mockRepo struct {
	createFn     func(sub *model.Subscription) error
	listFn       func(filter map[string]interface{}) ([]model.Subscription, error)
	aggregateFn   func(userID *uuid.UUID, serviceName *string, from, to time.Time) (int64, error)
}

func (m *mockRepo) Create(sub *model.Subscription) error { if m.createFn != nil { return m.createFn(sub) } return nil }
func (m *mockRepo) Get(id uuid.UUID) (*model.Subscription, error) { return nil, nil }
func (m *mockRepo) Update(sub *model.Subscription) error { return nil }
func (m *mockRepo) Delete(id uuid.UUID) error { return nil }
func (m *mockRepo) List(filter map[string]interface{}) ([]model.Subscription, error) { if m.listFn != nil { return m.listFn(filter) } return nil, nil }
func (m *mockRepo) AggregateSum(userID *uuid.UUID, serviceName *string, from, to time.Time) (int64, error) { if m.aggregateFn != nil { return m.aggregateFn(userID, serviceName, from, to) } return 0, nil }

func readBody(t *testing.T, r io.Reader, v interface{}) {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestCreateHandler_Valid(t *testing.T) {
	mr := &mockRepo{}
	mr.createFn = func(sub *model.Subscription) error {
		// check some fields
		if sub.ServiceName != "Yandex Plus" {
			t.Fatalf("unexpected service name: %s", sub.ServiceName)
		}
		return nil
	}
	lg := logrus.New()
	h := NewHandler(mr, lg)

	body := map[string]interface{}{
		"service_name": "Yandex Plus",
		"price": 400,
		"user_id": uuid.New().String(),
		"start_date": "07-2025",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/subscriptions/", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	var got model.Subscription
	readBody(t, rr.Body, &got)
	if got.ServiceName != "Yandex Plus" || got.Price != 400 {
		t.Fatalf("unexpected body: %+v", got)
	}
}

func TestAggregateHandler(t *testing.T) {
	mr := &mockRepo{}
	mr.aggregateFn = func(userID *uuid.UUID, serviceName *string, from, to time.Time) (int64, error) {
		return 1200, nil
	}
	lg := logrus.New()
	h := NewHandler(mr, lg)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/aggregate?from=07-2025&to=09-2025", nil)
	rr := httptest.NewRecorder()

	h.Aggregate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var res map[string]int64
	readBody(t, rr.Body, &res)
	if res["total"] != 1200 {
		t.Fatalf("unexpected total: %d", res["total"])
	}
}

func TestListHandler(t *testing.T) {
	sample := model.Subscription{ServiceName: "A", Price: 100}
	mr := &mockRepo{}
	mr.listFn = func(filter map[string]interface{}) ([]model.Subscription, error) { return []model.Subscription{sample}, nil }
	lg := logrus.New()
	h := NewHandler(mr, lg)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var arr []model.Subscription
	readBody(t, rr.Body, &arr)
	if len(arr) != 1 || arr[0].ServiceName != "A" {
		t.Fatalf("unexpected list response: %+v", arr)
	}
}
