package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/sunshine-walker-93/assistant_gateway_admin/internal/config"
	"github.com/sunshine-walker-93/assistant_gateway_admin/internal/handler"
	"github.com/sunshine-walker-93/assistant_gateway_admin/internal/middleware"
)

func main() {
	// Get database DSN from environment
	dsn := os.Getenv("ADMIN_DB_DSN")
	if dsn == "" {
		log.Fatal("ADMIN_DB_DSN environment variable is required")
	}

	// Get HTTP listen address
	listenAddr := getEnv("ADMIN_HTTP_LISTEN", ":8081")

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Create MySQL store
	store, err := config.NewMySQLStore(dsn)
	if err != nil {
		logger.Fatal("failed to create mysql store", zap.Error(err))
	}
	defer store.Close()

	// Build router
	r := chi.NewRouter()

	// Register middlewares
	r.Use(middleware.RequestLogger(logger))

	// Create handlers
	backendHandler := handler.NewBackendHandler(store, logger)
	routeHandler := handler.NewRouteHandler(store, logger)
	historyHandler := handler.NewHistoryHandler(store, logger)

	// Register API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Backend management
		r.Get("/backends", backendHandler.ListBackends)
		r.Get("/backends/{name}", backendHandler.GetBackend)
		r.Post("/backends", backendHandler.CreateBackend)
		r.Put("/backends/{name}", backendHandler.UpdateBackend)
		r.Delete("/backends/{name}", backendHandler.DeleteBackend)

		// Route management
		r.Get("/routes", routeHandler.ListRoutes)
		r.Get("/routes/{id}", routeHandler.GetRoute)
		r.Post("/routes", routeHandler.CreateRoute)
		r.Put("/routes/{id}", routeHandler.UpdateRoute)
		r.Delete("/routes/{id}", routeHandler.DeleteRoute)

		// Configuration history
		r.Get("/history", historyHandler.ListHistory)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("admin service listening", zap.String("addr", listenAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down admin service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
