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

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	addr := ":8080"
	if a := os.Getenv("ADDR"); a != "" {
		addr = a
	}
	log.Printf("HTTP-сервер запущен на %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
