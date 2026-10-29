package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// writeJSON пишет значение v в ответ в формате JSON с указанным кодом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErrorJSON пишет JSON-ошибку {"error": msg} с указанным кодом.
func writeErrorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// requestScheme определяет схему (http или https) входящего запроса.
// r.URL.Scheme на стороне сервера всегда пуст, поэтому схема
// определяется по TLS-соединению либо заголовку X-Forwarded-Proto.
func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if p := r.Header.Get("X-Forwarded-Proto"); p == "http" || p == "https" {
		return p
	}
	return "http"
}

// createLinkHandler обрабатывает POST /api/links.
// Ожидает JSON-тело {"url": "https://..."} и возвращает 201
// с созданным code.
func createLinkHandler(store *LinkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "тело запроса не является корректным JSON")
			return
		}
		target := strings.TrimSpace(payload.URL)
		if target == "" {
			writeErrorJSON(w, http.StatusBadRequest, "поле url не указано")
			return
		}

		u, err := url.Parse(target)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			writeErrorJSON(w, http.StatusBadRequest, "url должен быть абсолютной ссылкой http/https")
			return
		}

		link, err := store.Create(target)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "не удалось создать ссылку")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"code":      link.Code,
			"url":       link.URL,
			"short_url": requestScheme(r) + "://" + r.Host + "/" + link.Code,
		})
	}
}

// redirectHandler обрабатывает GET /{code}: увеличивает счётчик
// переходов и возвращает 302 с заголовком Location.
func redirectHandler(store *LinkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		code := r.PathValue("code")
		if code == "" {
			writeErrorJSON(w, http.StatusBadRequest, "code не указан")
			return
		}
		if !store.RecordHit(code) {
			writeErrorJSON(w, http.StatusNotFound, "ссылка не найдена")
			return
		}
		link, _ := store.Snapshot(code)
		http.Redirect(w, r, link.URL, http.StatusFound)
	}
}

// statsHandler обрабатывает GET /api/links/{code}/stats и возвращает
// статистику переходов в формате JSON.
func statsHandler(store *LinkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		code := r.PathValue("code")
		link, ok := store.Snapshot(code)
		if !ok {
			writeErrorJSON(w, http.StatusNotFound, "ссылка не найдена")
			return
		}
		writeJSON(w, http.StatusOK, link)
	}
}
