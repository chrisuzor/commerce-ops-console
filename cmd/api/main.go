package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chrisuzor/commerce-ops-console/internal/ai"
	"github.com/chrisuzor/commerce-ops-console/internal/graphqlapi"
	"github.com/chrisuzor/commerce-ops-console/internal/service"
	"github.com/chrisuzor/commerce-ops-console/internal/store"
)

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	repository, seed, err := openStore(ctx)
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	defer repository.Close()

	orders := service.NewOrders(repository, ai.EvidenceSummarizer{}, time.Now)
	if seed {
		if err = seedDemo(ctx, orders, time.Now().UTC()); err != nil {
			logger.Error("seed demo", "error", err)
			os.Exit(1)
		}
	}
	graph, err := graphqlapi.New(orders)
	if err != nil {
		logger.Error("build graphql schema", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /graphql", func(writer http.ResponseWriter, request *http.Request) {
		var payload graphqlRequest
		if err := decodeJSON(request.Body, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		result := graph.Execute(payload.Query, payload.Variables)
		status := http.StatusOK
		if len(result.Errors) > 0 {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(writer, status, result)
	})
	mux.HandleFunc("POST /webhooks/events", webhookHandler(orders, os.Getenv("WEBHOOK_SECRET")))
	if staticDir := strings.TrimSpace(os.Getenv("STATIC_DIR")); staticDir != "-" {
		if staticDir == "" {
			staticDir = "web/dist"
		}
		if info, statErr := os.Stat(staticDir); statErr == nil && info.IsDir() {
			mux.Handle("/", http.FileServer(http.Dir(staticDir)))
			logger.Info("serving dashboard", "directory", staticDir)
		}
	}

	port := envOr("PORT", "8080")
	server := &http.Server{
		Addr: ":" + port, Handler: withMiddleware(mux, logger),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		logger.Info("api listening", "port", port, "storage", storageName(seed))
		if listenErr := server.ListenAndServe(); !errors.Is(listenErr, http.ErrServerClosed) {
			logger.Error("server stopped", "error", listenErr)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}

func openStore(ctx context.Context) (store.Store, bool, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return store.NewMemory(), true, nil
	}
	repository, err := store.NewPostgres(ctx, databaseURL)
	return repository, false, err
}

func webhookHandler(orders *service.Orders, secret string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 1<<20))
		if err != nil {
			writeError(writer, http.StatusBadRequest, errors.New("request body is invalid or exceeds 1 MiB"))
			return
		}
		if secret != "" && !validSignature(body, request.Header.Get("X-Webhook-Signature"), secret) {
			writeError(writer, http.StatusUnauthorized, errors.New("invalid webhook signature"))
			return
		}
		var input service.IngestInput
		if err = json.Unmarshal(body, &input); err != nil {
			writeError(writer, http.StatusBadRequest, errors.New("invalid JSON payload"))
			return
		}
		if input.IdempotencyKey == "" {
			input.IdempotencyKey = request.Header.Get("X-Idempotency-Key")
		}
		result, err := orders.Ingest(request.Context(), input)
		if err != nil {
			writeError(writer, http.StatusUnprocessableEntity, err)
			return
		}
		writeJSON(writer, http.StatusAccepted, result)
	}
}

func validSignature(body []byte, header, secret string) bool {
	provided, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func withMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		writer.Header().Set("Access-Control-Allow-Origin", envOr("CORS_ORIGIN", "http://localhost:5173"))
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Idempotency-Key, X-Webhook-Signature")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
		logger.Info("request", "method", request.Method, "path", request.URL.Path, "durationMs", time.Since(started).Milliseconds())
	})
}

func decodeJSON(reader io.Reader, destination any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}

func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func storageName(memory bool) string {
	if memory {
		return "memory-demo"
	}
	return "postgres"
}
