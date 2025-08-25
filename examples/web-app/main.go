package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sarff/iSlogger"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Server struct {
	logger *iSlogger.Logger
}

func main() {
	// Initialize logger for web application
	config := iSlogger.DefaultConfig().
		WithAppName("webapp").
		WithLogLevel(slog.LevelWarn). // Production mode
		WithLogDir("web-logs").
		WithJSONFormat(true) // JSON format for log aggregation

	if err := iSlogger.Init(config); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer iSlogger.Close()

	// Create server instance
	server := &Server{
		logger: iSlogger.GetGlobalLogger(),
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.loggingMiddleware(server.homeHandler))
	mux.HandleFunc("/users", server.loggingMiddleware(server.usersHandler))
	mux.HandleFunc("/users/", server.loggingMiddleware(server.userHandler))
	mux.HandleFunc("/health", server.loggingMiddleware(server.healthHandler))

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		iSlogger.Info("Starting web server",
			"port", 8080,
			"debug_mode", false,
			"log_format", "json",
		)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			iSlogger.Error("Server failed to start", "error", err)
			os.Exit(1) // just example, dont do this ^)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	iSlogger.Info("Server is shutting down...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := httpServer.Shutdown(ctx); err != nil {
		iSlogger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	iSlogger.Info("Server stopped gracefully")
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create request-specific logger
		requestLogger := s.logger.With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"request_id", generateRequestID(),
		)

		requestLogger.Info("Request started")

		// Custom response writer to capture status code
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// Call next handler
		next(wrapper, r.WithContext(context.WithValue(r.Context(), "logger", requestLogger)))

		// Log request completion
		duration := time.Since(start)
		requestLogger.Info("Request completed",
			"status", wrapper.statusCode,
			"duration_ms", duration.Milliseconds(),
			"duration", duration.String(),
		)

		// Log slow requests as warnings
		if duration > 1*time.Second {
			requestLogger.Warn("Slow request detected",
				"threshold", "1s",
				"actual_duration", duration.String(),
			)
		}

		// Log errors
		if wrapper.statusCode >= 400 {
			if wrapper.statusCode >= 500 {
				requestLogger.Error("Server error",
					"status", wrapper.statusCode,
					"category", "server_error",
				)
			} else {
				requestLogger.Warn("Client error",
					"status", wrapper.statusCode,
					"category", "client_error",
				)
			}
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getLogger extracts logger from request context
func getLogger(r *http.Request) *iSlogger.Logger {
	if logger, ok := r.Context().Value("logger").(*iSlogger.Logger); ok {
		return logger
	}
	return iSlogger.GetGlobalLogger()
}

// homeHandler handles the root endpoint
func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	logger.Debug("Serving home page")

	response := map[string]string{
		"message": "Welcome to iSlogger demo API",
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// usersHandler handles /users endpoint
func (s *Server) usersHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	switch r.Method {
	case http.MethodGet:
		logger.Debug("Fetching users list")

		users := []User{
			{ID: 1, Name: "John Doe"},
			{ID: 2, Name: "Jane Smith"},
			{ID: 3, Name: "Bob Johnson"},
		}

		logger.Info("Users retrieved successfully", "count", len(users))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)

	case http.MethodPost:
		logger.Debug("Creating new user")

		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			logger.Error("Failed to decode user data", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Simulate user creation
		user.ID = 4 // Mock ID assignment
		logger.Info("User created successfully",
			"user_id", user.ID,
			"user_name", user.Name,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		logger.Warn("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// userHandler handles /users/{id} endpoint
func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	// Extract user ID from path
	idStr := r.URL.Path[len("/users/"):]
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Warn("Invalid user ID format", "id_string", idStr, "error", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	logger = logger.With("user_id", userID)
	logger.Debug("Processing user request")

	// Simulate user lookup
	if userID <= 0 || userID > 3 {
		logger.Warn("User not found", "requested_id", userID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user := User{
		ID:   userID,
		Name: fmt.Sprintf("User %d", userID),
	}

	logger.Info("User retrieved successfully", "user_name", user.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// healthHandler handles health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	logger.Debug("Health check requested")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// generateRequestID creates a simple request ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

var startTime = time.Now()
