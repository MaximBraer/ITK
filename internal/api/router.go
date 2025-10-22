package api

import (
	"log/slog"
	"net/http"
	"time"

	"ITK/internal/api/handlers"
	"ITK/internal/api/middleware/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter(log *slog.Logger, walletHandler *handlers.Handler) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(5 * time.Second))

	router.Use(logger.New(log))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fs := http.FileServer(http.Dir(".static/swagger"))
	router.Handle("/static/swagger/*", http.StripPrefix("/static/swagger", fs))

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/static/swagger/swagger.json"),
	))

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/wallet/create", walletHandler.Create)
		r.Post("/wallet", walletHandler.Operation)
		r.Get("/wallets/{id}", walletHandler.GetBalance)
	})

	return router
}
