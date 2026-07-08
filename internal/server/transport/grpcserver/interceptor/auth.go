package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// contextKey — приватный тип ключа контекста, чтобы избежать коллизий с другими пакетами.
type contextKey string

const userIDContextKey contextKey = "userID"

type TokenVerifier interface {
	Verify(token string) (userID string, err error)
}

// methodsWithoutAuth — RPC-методы, не требующие access-токена.
var methodsWithoutAuth = map[string]bool{
	"/gophkeeper.v1.AuthService/Ping":         true,
	"/gophkeeper.v1.AuthService/Register":     true,
	"/gophkeeper.v1.AuthService/Login":        true,
	"/gophkeeper.v1.AuthService/RefreshToken": true,
}

// Auth — unary-интерцептор, проверяющий access-token в metadata запроса и кладущий
// userID в context.
func Auth(verifier TokenVerifier) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if methodsWithoutAuth[info.FullMethod] {
			return handler(ctx, req)
		}

		token, err := extractBearerToken(ctx)
		if err != nil {
			return nil, err
		}

		userID, err := verifier.Verify(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired access token")
		}

		ctx = context.WithValue(ctx, userIDContextKey, userID)
		return handler(ctx, req)
	}
}

// UserIDFromContext достаёт userID, положенный интерцептором Auth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

func extractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(values[0], prefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}

	return strings.TrimPrefix(values[0], prefix), nil
}
