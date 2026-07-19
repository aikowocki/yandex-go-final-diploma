// autorefresh.go — обёртка над TokenStore с проактивным обновлением access-токена.
// При Load() проверяет expiry JWT: если access-токен протухнет через < 2 мин — вызывает
// RefreshToken RPC и сохраняет новую пару. Прозрачно для вызывающего кода.
package keyring

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

const refreshThreshold = 2 * time.Minute

// Refresher — узкий интерфейс для вызова RefreshToken.
type Refresher interface {
	RefreshToken(ctx context.Context, refreshToken string) (contracts.LoginResult, error)
}

// AutoRefreshStore оборачивает TokenStore и добавляет проактивный refresh при Load().
type AutoRefreshStore struct {
	inner  contracts.TokenStore
	server Refresher
	mu     sync.Mutex
}

var _ contracts.TokenStore = (*AutoRefreshStore)(nil)

// NewAutoRefreshStore создаёт обёртку.
func NewAutoRefreshStore(inner contracts.TokenStore, server Refresher) *AutoRefreshStore {
	return &AutoRefreshStore{inner: inner, server: server}
}

// Save сохраняет токены через обёрнутый TokenStore.
func (s *AutoRefreshStore) Save(t contracts.Tokens) error {
	return s.inner.Save(t)
}

// Clear удаляет сохранённые токены через обёрнутый TokenStore.
func (s *AutoRefreshStore) Clear() error {
	return s.inner.Clear()
}

// Load загружает токены и проактивно обновляет их, если access token близок к истечению.
func (s *AutoRefreshStore) Load() (contracts.Tokens, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.inner.Load()
	if err != nil {
		return tokens, err
	}

	// Проверяем expiry access-токена.
	exp, ok := parseJWTExp(tokens.AccessToken)
	if !ok {
		// Не удалось распарсить — возвращаем как есть.
		return tokens, nil
	}

	if time.Until(exp) > refreshThreshold {
		// Токен ещё свежий — ничего не делаем.
		return tokens, nil
	}

	// Токен скоро протухнет или уже протух — обновляем проактивно.
	slog.Info("token: proactive refresh", "expires_in", time.Until(exp).Truncate(time.Second).String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := s.server.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		slog.Warn("token: proactive refresh failed, using existing token", "err", err)
		// Fallback: возвращаем старый — если протух, сервер ответит Unauthenticated,
		// но offline-операции всё равно продолжат работать.
		return tokens, nil
	}

	newTokens := res.Tokens
	if saveErr := s.inner.Save(newTokens); saveErr != nil {
		slog.Warn("token: save refreshed tokens failed", "err", saveErr)
	}
	return newTokens, nil
}

// parseJWTExp извлекает exp из JWT payload (без верификации подписи — только для проверки TTL).
func parseJWTExp(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
