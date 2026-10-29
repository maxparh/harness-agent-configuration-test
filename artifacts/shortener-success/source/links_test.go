package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// newLinksTestServer возвращает тестовый HTTP-сервер со всеми
// маршрутами приложения: /health, POST /api/links,
// GET /api/links/{code}/stats и GET /{code}.
func newLinksTestServer() *httptest.Server {
	return httptest.NewServer(newAppMux(NewLinkStore()))
}

// noRedirectClient возвращает HTTP-клиент, который не следует
// за редиректами, чтобы тестировать ответ 302 напрямую.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// createTestLink создаёт ссылку через POST /api/links и
// возвращает её code.
func createTestLink(t *testing.T, srv *httptest.Server, url string) string {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"url": url})
	resp, err := http.Post(srv.URL+"/api/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/links: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /api/links: код ответа = %d, ожидается %d (тело: %s)",
			resp.StatusCode, http.StatusCreated, data)
	}

	var payload struct {
		Code     string `json:"code"`
		URL      string `json:"url"`
		ShortURL string `json:"short_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("POST /api/links: ошибка декодирования JSON: %v", err)
	}
	if payload.Code == "" {
		t.Fatal("POST /api/links: пустой code в ответе")
	}
	if payload.URL != url {
		t.Errorf("POST /api/links: url = %q, ожидается %q", payload.URL, url)
	}
	return payload.Code
}

// getStats запрашивает GET /api/links/{code}/stats и возвращает
// число переходов.
func getStats(t *testing.T, srv *httptest.Server, code string) int64 {
	t.Helper()

	resp, err := http.Get(srv.URL + "/api/links/" + code + "/stats")
	if err != nil {
		t.Fatalf("GET /api/links/%s/stats: ошибка запроса: %v", code, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET stats: код ответа = %d, ожидается %d (тело: %s)",
			resp.StatusCode, http.StatusOK, data)
	}

	var payload struct {
		Hits int64 `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("GET stats: ошибка декодирования JSON: %v", err)
	}
	return payload.Hits
}

// isBase62Char показывает, является ли c символом base62-алфавита.
func isBase62Char(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// TestCreateLink проверяет, что POST /api/links возвращает 201
// и JSON с code, url и short_url.
func TestCreateLink(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	target := "https://example.com/very/long/path"
	body, _ := json.Marshal(map[string]string{"url": target})
	resp, err := http.Post(srv.URL+"/api/links", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/links: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/links: код ответа = %d, ожидается %d", resp.StatusCode, http.StatusCreated)
	}

	var payload struct {
		Code     string `json:"code"`
		URL      string `json:"url"`
		ShortURL string `json:"short_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("POST /api/links: ошибка декодирования JSON: %v", err)
	}
	if len(payload.Code) != 6 {
		t.Errorf("длина code = %d, ожидается 6", len(payload.Code))
	}
	for _, c := range payload.Code {
		if !isBase62Char(c) {
			t.Errorf("code содержит не-base62 символ %q", c)
		}
	}
	if payload.URL != target {
		t.Errorf("url = %q, ожидается %q", payload.URL, target)
	}
	wantShort := srv.URL + "/" + payload.Code
	if payload.ShortURL != wantShort {
		t.Errorf("short_url = %q, ожидается %q", payload.ShortURL, wantShort)
	}
}

// TestCreateLinkInvalidURL проверяет, что POST /api/links с
// некорректным URL или телом возвращает 400.
func TestCreateLinkInvalidURL(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	cases := map[string]string{
		"пустой url":          `{"url": ""}`,
		"нет поля url":        `{}`,
		"не URL":              `{"url": "just a string"}`,
		"нет схемы":           `{"url": "example.com"}`,
		"не та схема":         `{"url": "ftp://example.com"}`,
		"нет host":            `{"url": "https://"}`,
		"битый JSON":          `{"url": "https://ex`,
		"url не строкой":      `{"url": 42}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Post(srv.URL+"/api/links", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatalf("POST /api/links: ошибка запроса: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				data, _ := io.ReadAll(resp.Body)
				t.Errorf("POST /api/links: код ответа = %d, ожидается %d (тело: %s)",
					resp.StatusCode, http.StatusBadRequest, data)
			}
		})
	}
}

// TestCreateLinkUniqueCodes проверяет, что созданные ссылки
// получают уникальные code.
func TestCreateLinkUniqueCodes(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	seen := make(map[string]bool, 64)
	for i := 0; i < 64; i++ {
		code := createTestLink(t, srv, fmt.Sprintf("https://example.com/%d", i))
		if seen[code] {
			t.Fatalf("повторяющийся code %s", code)
		}
		seen[code] = true
	}
}

// TestRedirect проверяет, что GET /{code} возвращает 302,
// а заголовок Location содержит исходный URL.
func TestRedirect(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	target := "https://example.com/page?q=1"
	code := createTestLink(t, srv, target)

	client := noRedirectClient()
	resp, err := client.Get(srv.URL + "/" + code)
	if err != nil {
		t.Fatalf("GET /%s: ошибка запроса: %v", code, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("GET /%s: код ответа = %d, ожидается %d", code, resp.StatusCode, http.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != target {
		t.Errorf("Location = %q, ожидается %q", loc, target)
	}
}

// TestRedirectUnknownCode проверяет, что GET на неизвестный
// code возвращает 404.
func TestRedirectUnknownCode(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	client := noRedirectClient()
	resp, err := client.Get(srv.URL + "/unknown0")
	if err != nil {
		t.Fatalf("GET /unknown0: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /unknown0: код ответа = %d, ожидается %d", resp.StatusCode, http.StatusNotFound)
	}
}

// TestRedirectIncrementsHits проверяет, что каждый переход
// увеличивает счётчик, видимый через stats-эндпоинт.
func TestRedirectIncrementsHits(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	code := createTestLink(t, srv, "https://example.com")
	client := noRedirectClient()

	for i := 0; i < 3; i++ {
		resp, err := client.Get(srv.URL + "/" + code)
		if err != nil {
			t.Fatalf("GET /%s: ошибка запроса: %v", code, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("GET /%s: код ответа = %d, ожидается %d", code, resp.StatusCode, http.StatusFound)
		}
	}

	if hits := getStats(t, srv, code); hits != 3 {
		t.Errorf("hits = %d, ожидается 3", hits)
	}
}

// TestStats проверяет, что GET /api/links/{code}/stats возвращает
// 200 и JSON с корректным числом переходов.
func TestStats(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	target := "https://example.com/stats"
	code := createTestLink(t, srv, target)
	client := noRedirectClient()

	for i := 0; i < 2; i++ {
		resp, err := client.Get(srv.URL + "/" + code)
		if err != nil {
			t.Fatalf("GET /%s: ошибка запроса: %v", code, err)
		}
		resp.Body.Close()
	}

	resp, err := http.Get(srv.URL + "/api/links/" + code + "/stats")
	if err != nil {
		t.Fatalf("GET stats: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET stats: код ответа = %d, ожидается %d", resp.StatusCode, http.StatusOK)
	}

	var payload struct {
		Code  string `json:"code"`
		URL   string `json:"url"`
		Hits  int64  `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("GET stats: ошибка декодирования JSON: %v", err)
	}
	if payload.Code != code {
		t.Errorf("code = %q, ожидается %q", payload.Code, code)
	}
	if payload.URL != target {
		t.Errorf("url = %q, ожидается %q", payload.URL, target)
	}
	if payload.Hits != 2 {
		t.Errorf("hits = %d, ожидается 2", payload.Hits)
	}
}

// TestStatsUnknownCode проверяет, что запрос статистики
// для неизвестного code возвращает 404 с JSON-ошибкой.
func TestStatsUnknownCode(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/links/nope01/stats")
	if err != nil {
		t.Fatalf("GET stats: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET stats: код ответа = %d, ожидается %d", resp.StatusCode, http.StatusNotFound)
	}

	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("GET stats: ошибка декодирования JSON: %v", err)
	}
	if payload.Error == "" {
		t.Error("GET stats: пустое поле error в ответе")
	}
}

// TestStatsMethodNotAllowed проверяет, что методы отличные от GET
// на stats-эндпоинте возвращают 405.
func TestStatsMethodNotAllowed(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	code := createTestLink(t, srv, "https://example.com")

	resp, err := http.Post(srv.URL+"/api/links/"+code+"/stats", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST stats: ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST stats: код ответа = %d, ожидается %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

// TestConcurrentRedirects проверяет потокобезопасность хранилища:
// конкурентные переходы к одному code корректно увеличивают
// счётчик (при запуске с -race также ловит data race).
func TestConcurrentRedirects(t *testing.T) {
	srv := newLinksTestServer()
	defer srv.Close()

	code := createTestLink(t, srv, "https://example.com")
	client := noRedirectClient()

	const workers = 32
	const perWorker = 8
	var wg sync.WaitGroup
	errCh := make(chan error, workers*perWorker)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				resp, err := client.Get(srv.URL + "/" + code)
				if err != nil {
					errCh <- fmt.Errorf("GET /%s: %v", code, err)
					return
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusFound {
					errCh <- fmt.Errorf("код ответа = %d, ожидается %d", resp.StatusCode, http.StatusFound)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	expected := int64(workers * perWorker)
	if hits := getStats(t, srv, code); hits != expected {
		t.Errorf("hits = %d, ожидается %d", hits, expected)
	}
}
