package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tokenType различает access и refresh токены внутри claims, чтобы Verify мог
// отклонить refresh-токен, предъявленный как access (и наоборот) — иначе более
// долгоживущий refresh-токен расширял бы окно атаки при использовании как access.
type tokenType string

const (
	tokenTypeAccess  tokenType = "access"
	tokenTypeRefresh tokenType = "refresh"
)

var (
	// ErrInvalidToken — токен не прошёл проверку подписи/срока действия.
	ErrInvalidToken = errors.New("jwt: invalid token")
	// ErrWrongType — предъявлен токен не того типа (access вместо refresh или наоборот).
	ErrWrongType = errors.New("jwt: unexpected token type")
)

type claims struct {
	jwt.RegisteredClaims
	Type tokenType `json:"type"`
}

// TokenIssuer выпускает и проверяет JWT access/refresh токены.
type TokenIssuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// New создаёт TokenIssuer с секретом подписи и TTL для access/refresh токенов.
func New(secret []byte, accessTTL, refreshTTL time.Duration) *TokenIssuer {
	return &TokenIssuer{
		secret:     secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Issue выпускает новую пару access+refresh токенов для userID.
func (t *TokenIssuer) Issue(userID string) (access, refresh string, err error) {
	access, err = t.sign(userID, tokenTypeAccess, t.accessTTL)
	if err != nil {
		return "", "", fmt.Errorf("issue access token: %w", err)
	}

	refresh, err = t.sign(userID, tokenTypeRefresh, t.refreshTTL)
	if err != nil {
		return "", "", fmt.Errorf("issue refresh token: %w", err)
	}

	return access, refresh, nil
}

// Verify проверяет access-токен и возвращает userID.
func (t *TokenIssuer) Verify(token string) (userID string, err error) {
	return t.parse(token, tokenTypeAccess)
}

// VerifyRefresh проверяет refresh-токен и возвращает userID.
func (t *TokenIssuer) VerifyRefresh(token string) (userID string, err error) {
	return t.parse(token, tokenTypeRefresh)
}

// Refresh проверяет refresh-токен и выпускает новую пару.
func (t *TokenIssuer) Refresh(refreshToken string) (access, refresh string, err error) {
	userID, err := t.parse(refreshToken, tokenTypeRefresh)
	if err != nil {
		return "", "", err
	}

	return t.Issue(userID)
}

func (t *TokenIssuer) sign(userID string, typ tokenType, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Type: typ,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(t.secret)
}

func (t *TokenIssuer) parse(tokenStr string, expected tokenType) (string, error) {
	var c claims
	parsed, err := jwt.ParseWithClaims(tokenStr, &c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.secret, nil
	})
	if err != nil || !parsed.Valid {
		return "", ErrInvalidToken
	}

	if c.Type != expected {
		return "", ErrWrongType
	}

	return c.Subject, nil
}
