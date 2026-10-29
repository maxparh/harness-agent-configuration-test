package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer возвращает тестовый HTTP-сервер с той же
// конфигурацией, что и основное приложение.
func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	return httptest.NewServer(mux)
}

// TestHealthGet проверяет, что GET /health возвращает HTTP 200
// и JSON-ответ {"status":"ok"}.
func TestHealthGet(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: код ответа = %d, ожидается %d", resp.StatusCode, http.StatusOK)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("GET /health: ошибка декодирования JSON: %v", err)
	}
	if payload.Status != "ok" {
		t.Errorf("GET /health: status = %q, ожидается %q", payload.Status, "ok")
	}
}

// TestHealthNonGetMethods проверяет, что методы отличные от GET
// на /health возвращают HTTP 405.
func TestHealthNonGetMethods(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, srv.URL+"/health", nil)
			if err != nil {
				t.Fatalf("создание запроса %s: %v", method, err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("%s /health: ошибка запроса: %v", method, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("%s /health: код ответа = %d, ожидается %d", method, resp.StatusCode, http.StatusMethodNotAllowed)
			}
		})
	}
}
