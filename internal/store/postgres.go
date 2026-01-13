package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/effectivemobile/subscriptions/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/google/uuid"
)

type Repository interface {
	Create(sub *model.Subscription) error
	Get(id uuid.UUID) (*model.Subscription, error)
	Update(sub *model.Subscription) error
	Delete(id uuid.UUID) error
	List(filter map[string]interface{}) ([]model.Subscription, error)
	AggregateSum(userID *uuid.UUID, serviceName *string, from, to time.Time) (int64, error)
}

type PostgresRepo struct {
	db  *sqlx.DB
	log *logrus.Logger
}

func NewPostgresRepository(db *sqlx.DB, log *logrus.Logger) *PostgresRepo {
	return &PostgresRepo{db: db, log: log}
}

func EnsureMigrations(db *sqlx.DB) error {
	// minimal programmatic migration: create extension and table
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto;`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_name TEXT NOT NULL,
			price INTEGER NOT NULL,
			user_id UUID NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresRepo) Create(sub *model.Subscription) error {
	q := `INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
	VALUES ($1,$2,$3,$4,$5,$6)`
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	_, err := p.db.Exec(q, sub.ID, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate)
	return err
}

func (p *PostgresRepo) Get(id uuid.UUID) (*model.Subscription, error) {
	var s model.Subscription
	q := `SELECT id,service_name,price,user_id,start_date,end_date FROM subscriptions WHERE id=$1`
	if err := p.db.Get(&s, q, id); err != nil {
		return nil, err
	}
	return &s, nil
}

func (p *PostgresRepo) Update(sub *model.Subscription) error {
	q := `UPDATE subscriptions SET service_name=$1, price=$2, user_id=$3, start_date=$4, end_date=$5 WHERE id=$6`
	_, err := p.db.Exec(q, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate, sub.ID)
	return err
}

func (p *PostgresRepo) Delete(id uuid.UUID) error {
	q := `DELETE FROM subscriptions WHERE id=$1`
	_, err := p.db.Exec(q, id)
	return err
}

func (p *PostgresRepo) List(filter map[string]interface{}) ([]model.Subscription, error) {
	q := `SELECT id,service_name,price,user_id,start_date,end_date FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	idx := 1
	if v, ok := filter["user_id"]; ok {
		q += ` AND user_id=$` + itoa(idx)
		args = append(args, v)
		idx++
	}
	if v, ok := filter["service_name"]; ok {
		q += ` AND service_name ILIKE $` + itoa(idx)
		args = append(args, "%"+v.(string)+"%")
		idx++
	}
	var rows []model.Subscription
	if err := p.db.Select(&rows, q, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func (p *PostgresRepo) AggregateSum(userID *uuid.UUID, serviceName *string, from, to time.Time) (int64, error) {
	// sum months * price for subscriptions overlapping [from,to]
	// For each subscription: overlap months = months_between(min(end or to), max(start,from)) + 1
	q := `SELECT price, start_date, end_date FROM subscriptions WHERE (end_date IS NULL OR end_date >= $1) AND start_date <= $2`
	args := []interface{}{from, to}
	if userID != nil {
		q += ` AND user_id = $3`
		args = append(args, *userID)
	}
	if serviceName != nil {
		q += ` AND service_name = $4`
		args = append(args, *serviceName)
	}

	tx, err := p.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	type row struct {
		Price     int       `db:"price"`
		StartDate time.Time `db:"start_date"`
		EndDate   sql.NullTime `db:"end_date"`
	}
	var total int64
	var rrow row
	rows, err := tx.Queryx(q, args...)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		if err := rows.StructScan(&rrow); err != nil {
			return 0, err
		}
		s := rrow.StartDate
		e := rrow.EndDate
		if e.Valid && e.Time.Before(from) { // finished before period
			continue
		}
		start := maxTime(s, from)
		end := to
		if e.Valid && e.Time.Before(to) {
			end = e.Time
		}
		months := monthsInclusive(start, end)
		if months <= 0 {
			continue
		}
		total += int64(months * rrow.Price)
	}
	return total, nil
}

// helpers
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func monthsInclusive(a, b time.Time) int {
	if b.Before(a) {
		return 0
	}
	y1, m1, _ := a.Date()
	y2, m2, _ := b.Date()
	return (y2-y1)*12 + int(m2-m1) + 1
}
