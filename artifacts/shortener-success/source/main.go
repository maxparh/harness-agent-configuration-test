// Команда agent-test запускает минимальное HTTP-приложение
// с эндпоинтом /health на стандартной библиотеке Go.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// healthHandler обрабатывает запросы к /health.
// GET возвращает HTTP 200 и JSON {"status":"ok"},
// остальные методы — HTTP 405.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// newAppMux собирает mux со всеми маршрутами приложения:
// /health, POST /api/links, GET /api/links/{code}/stats и GET /{code}.
func newAppMux(store *LinkStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /api/links", createLinkHandler(store))
	mux.HandleFunc("GET /api/links/{code}/stats", statsHandler(store))
	mux.HandleFunc("GET /{code}", redirectHandler(store))
	return mux
}

func main() {
	store := NewLinkStore()
	mux := newAppMux(store)

	addr := ":8080"
	if a := os.Getenv("ADDR"); a != "" {
		addr = a
	}
	log.Printf("HTTP-сервер запущен на %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
