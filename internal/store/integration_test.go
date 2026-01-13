package store

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/effectivemobile/subscriptions/internal/model"
	"github.com/google/uuid"
)

func TestIntegration_Aggregate(t *testing.T) {
	// If POSTGRES_DSN is provided (e.g. in CI), use it directly, else start a docker postgres via dockertest
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		db, err := sqlx.Connect("postgres", dsn)
		if err != nil {
			t.Fatalf("could not connect to provided POSTGRES_DSN: %v", err)
		}
		defer db.Close()
		runIntegrationAgainstDB(t, db)
		return
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("could not connect to docker: %v", err)
	}

	opts := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=subscriptions_db",
		},
	}
	resource, err := pool.RunWithOptions(opts)
	if err != nil {
		t.Fatalf("could not start resource: %v", err)
	}
	defer func() {
		_ = pool.Purge(resource)
	}()

	var db *sqlx.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		port := resource.GetPort("5432/tcp")
		dsn := fmt.Sprintf("host=localhost port=%s user=postgres password=postgres dbname=subscriptions_db sslmode=disable", port)
		var err error
		db, err = sqlx.Connect("postgres", dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("could not connect to docker postgres: %v", err)
	}
	defer db.Close()

	runIntegrationAgainstDB(t, db)
}

func runIntegrationAgainstDB(t *testing.T, db *sqlx.DB) {
	// Run migrations from EnsureMigrations
	if err := EnsureMigrations(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	repo := NewPostgresRepository(db, nil)

	// Prepare sample data
	uid := uuid.New()
	// active across Jul-Sep 2025, price 100 -> 3 months = 300
	s1 := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: "S1",
		Price:       100,
		UserID:      uid,
		StartDate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}
	if err := repo.Create(s1); err != nil {
		t.Fatalf("failed create s1: %v", err)
	}

	// subscription that ends August 2025, price 200 -> Jul-Aug overlap with Jul-Sep = 2 months -> 400
	s2 := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: "S2",
		Price:       200,
		UserID:      uid,
		StartDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     ptrTime(time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)),
	}
	if err := repo.Create(s2); err != nil {
		t.Fatalf("failed create s2: %v", err)
	}

	// different user - should be ignored when filtering
	s3 := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: "S3",
		Price:       1000,
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.Create(s3); err != nil {
		t.Fatalf("failed create s3: %v", err)
	}

	from := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC)

	total, err := repo.AggregateSum(&uid, nil, from, to)
	if err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	if total != 700 { // 300 + 400
		t.Fatalf("expected total 700, got %d", total)
	}

	// test filtering by service name
	total2, err := repo.AggregateSum(nil, strPtr("S1"), from, to)
	if err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	if total2 == 0 {
		t.Fatalf("expected non-zero total for service S1, got 0")
	}
}
	// Run migrations from EnsureMigrations
	if err := EnsureMigrations(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	repo := NewPostgresRepository(db, nil)

	// Prepare sample data
	uid := uuid.New()
	// active across Jul-Sep 2025, price 100 -> 3 months = 300
	s1 := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: "S1",
		Price:       100,
		UserID:      uid,
		StartDate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}
	if err := repo.Create(s1); err != nil {
		t.Fatalf("failed create s1: %v", err)
	}

	// subscription that ends August 2025, price 200 -> Jul-Aug overlap with Jul-Sep = 2 months -> 400
	s2 := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: "S2",
		Price:       200,
		UserID:      uid,
		StartDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     ptrTime(time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)),
	}
	if err := repo.Create(s2); err != nil {
		t.Fatalf("failed create s2: %v", err)
	}

	// different user - should be ignored when filtering
	s3 := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: "S3",
		Price:       1000,
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.Create(s3); err != nil {
		t.Fatalf("failed create s3: %v", err)
	}

	from := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC)

	total, err := repo.AggregateSum(&uid, nil, from, to)
	if err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	if total != 700 { // 300 + 400
		t.Fatalf("expected total 700, got %d", total)
	}

	// test filtering by service name
	total2, err := repo.AggregateSum(nil, strPtr("S1"), from, to)
	if err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	if total2 == 0 {
		t.Fatalf("expected non-zero total for service S1, got 0")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
func strPtr(s string) *string { return &s }
