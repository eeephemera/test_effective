package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/effectivemobile/subscriptions/internal/model"
	"github.com/effectivemobile/subscriptions/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	repo store.Repository
	log  *logrus.Logger
	val  *validator.Validate
}

func NewHandler(r store.Repository, l *logrus.Logger) *Handler {
	return &Handler{repo: r, log: l, val: validator.New()}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warnf("invalid create body: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.val.Struct(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	uid, _ := uuid.Parse(req.UserID)
	start, err := parseMonthYear(req.StartDate)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid start_date format, expected MM-YYYY")
		return
	}
	var end *time.Time
	if req.EndDate != nil {
		et, err := parseMonthYear(*req.EndDate)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid end_date format, expected MM-YYYY")
			return
		}
		end = &et
	}
	sub := &model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      uid,
		StartDate:   start,
		EndDate:     end,
	}
	if err := h.repo.Create(sub); err != nil {
		h.log.Errorf("create failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create")
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, err := h.repo.Get(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not found")
		return
	}
	json.NewEncoder(w).Encode(s)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req model.SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.val.Struct(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	uid, _ := uuid.Parse(req.UserID)
	start, err := parseMonthYear(req.StartDate)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid start_date format, expected MM-YYYY")
		return
	}
	var end *time.Time
	if req.EndDate != nil {
		et, err := parseMonthYear(*req.EndDate)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid end_date format, expected MM-YYYY")
			return
		}
		end = &et
	}
	sub := &model.Subscription{
		ID:          id,
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      uid,
		StartDate:   start,
		EndDate:     end,
	}
	if err := h.repo.Update(sub); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to update")
		return
	}
	json.NewEncoder(w).Encode(sub)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(id); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := map[string]interface{}{}
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		if u, err := uuid.Parse(uid); err == nil {
			filter["user_id"] = u
		} else {
			h.writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	}
	if s := r.URL.Query().Get("service_name"); s != "" {
		filter["service_name"] = s
	}
	res, err := h.repo.List(filter)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed")
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Aggregate(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		h.writeError(w, http.StatusBadRequest, "from and to are required in MM-YYYY format")
		return
	}
	from, err := parseMonthYear(fromStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid from format")
		return
	}
	// set from to first day of month
	from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	toMonth, err := parseMonthYear(toStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid to format")
		return
	}
	// set to to last day of month
	to := time.Date(toMonth.Year(), toMonth.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, -1)

	var uid *uuid.UUID
	if v := r.URL.Query().Get("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		uid = &id
	}
	var serviceName *string
	if v := r.URL.Query().Get("service_name"); v != "" {
		serviceName = &v
	}
	res, err := h.repo.AggregateSum(uid, serviceName, from, to)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "aggregation failed")
		return
	}
	json.NewEncoder(w).Encode(map[string]int64{"total": res})
}

// utilities

func (h *Handler) writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func parseMonthYear(s string) (time.Time, error) {
	// expected MM-YYYY
	return time.Parse("01-2006", s)
}
