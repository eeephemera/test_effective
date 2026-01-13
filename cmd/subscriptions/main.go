package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/effectivemobile/subscriptions/internal/config"
	"github.com/effectivemobile/subscriptions/internal/handlers"
	"github.com/effectivemobile/subscriptions/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	log := logrus.New()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Infof("starting subscriptions service on %s", cfg.Server.Address)

	db, err := sqlx.Connect("postgres", cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("can't connect to db: %v", err)
	}
	defer db.Close()

	// run simple migration on startup
	if err := store.EnsureMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repo := store.NewPostgresRepository(db, log)
	h := handlers.NewHandler(repo, log)

	r := chi.NewRouter()
	// middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware(log))

	r.Route("/subscriptions", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/aggregate", h.Aggregate)
	})

	// serve swagger spec and UI
	r.Get("/docs/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.yaml")
	})
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger-ui/index.html")
	})

}

func loggingMiddleware(log *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := middleware.GetReqID(r.Context())
			start := time.Now()
			next.ServeHTTP(w, r)
			log.WithFields(logrus.Fields{
				"req_id": rid,
				"method": r.Method,
				"path":   r.URL.Path,
				"dur_ms": time.Since(start).Milliseconds(),
			}).Info("handled request")
		})
	}
}

	srv := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Info("server stopped")
}
