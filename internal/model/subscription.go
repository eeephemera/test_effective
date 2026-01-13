package model

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	ServiceName string     `db:"service_name" json:"service_name"`
	Price       int        `db:"price" json:"price"`
	UserID      uuid.UUID  `db:"user_id" json:"user_id"`
	StartDate   time.Time  `db:"start_date" json:"start_date"`
	EndDate     *time.Time `db:"end_date" json:"end_date,omitempty"`
}

// Create/Update request body
type SubscriptionRequest struct {
	ServiceName string  `json:"service_name" validate:"required,min=1"`
	Price       int     `json:"price" validate:"required,min=0"`
	UserID      string  `json:"user_id" validate:"required,uuid4"`
	StartDate   string  `json:"start_date" validate:"required"`
	EndDate     *string `json:"end_date,omitempty"`
}
