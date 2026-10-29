package main

import (
	"crypto/rand"
	"errors"
	"sync"
	"time"
)

// Link представляет сокращённую ссылку и накопленную статистику переходов.
type Link struct {
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	Hits      int64     `json:"hits"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LinkStore — потокобезопасное in-memory хранилище ссылок.
// Все методы безопасны для конкурентного вызова: состояние
// защищается sync.RWMutex.
type LinkStore struct {
	mu    sync.RWMutex
	links map[string]*Link
}

// errCollision — ошибка, если после нескольких попыток не удалось
// сгенерировать уникальный code.
var errCollision = errors.New("не удалось сгенерировать уникальный code")

// NewLinkStore создаёт пустое хранилище ссылок.
func NewLinkStore() *LinkStore {
	return &LinkStore{links: make(map[string]*Link)}
}

const (
	codeAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength   = 6
	maxCodeTries = 32
)

// generateCode создаёт случайный code заданной длины из base62-алфавита.
func generateCode() (string, error) {
	b := make([]byte, codeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b), nil
}

// Create сохраняет новую ссылку и возвращает её.
// Если совпадение code возникает, генерация повторяется.
func (s *LinkStore) Create(url string) (*Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < maxCodeTries; i++ {
		code, err := generateCode()
		if err != nil {
			return nil, err
		}
		if _, exists := s.links[code]; exists {
			continue
		}
		now := time.Now().UTC()
		link := &Link{Code: code, URL: url, CreatedAt: now, UpdatedAt: now}
		s.links[code] = link
		return link, nil
	}
	return nil, errCollision
}

// Get возвращает ссылку по code. Второй результат показывает,
// найдена ли ссылка.
func (s *LinkStore) Get(code string) (*Link, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.links[code]
	return link, ok
}

// RecordHit увеличивает счётчик переходов ссылки по code.
// Возвращает false, если ссылка не найдена.
func (s *LinkStore) RecordHit(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[code]
	if !ok {
		return false
	}
	link.Hits++
	link.UpdatedAt = time.Now().UTC()
	return true
}

// Snapshot возвращает копию ссылки по code, чтобы вызывающий код
// мог читать состояние без блокировки хранилища.
func (s *LinkStore) Snapshot(code string) (Link, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.links[code]
	if !ok {
		return Link{}, false
	}
	return *link, true
}
