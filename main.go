package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

type helloResponse struct {
	Message   string `json:"message"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./web/dist"
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/hello", helloHandler)
	r.Get("/api/healthz", healthHandler)

	// Serve the built front-end (Vite output) and fall back to index.html
	// so client-side routing keeps working.
	r.Handle("/*", spaHandler(staticDir))

	log.Printf("claims-processor %s listening on :%s (static=%s)", version, port, staticDir)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, helloResponse{
		Message:   "Hello from the Claims Processor back-end!",
		Service:   "claims-processor",
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// spaHandler serves static assets from dir and falls back to index.html
// for any path that does not map to an existing file.
func spaHandler(dir string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))
	return func(w http.ResponseWriter, req *http.Request) {
		path := dir + req.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, req, dir+"/index.html")
			return
		}
		fs.ServeHTTP(w, req)
	}
}
