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

// UserIDFromContext достаёт userID, положенный интерцептором Auth/StreamAuth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

// authenticatedStream оборачивает grpc.ServerStream, подменяя Context() на контекст с userID —
// нужно, так как ServerStream.Context() нельзя изменить напрямую.
type authenticatedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedStream) Context() context.Context { return s.ctx }

// StreamAuth — интерцептор для streaming RPC (client/server-streaming), проверяющий access-token
// так же, как Auth для unary. Нужен отдельно: grpc.UnaryInterceptor не применяется к стримам.
func StreamAuth(verifier TokenVerifier) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if methodsWithoutAuth[info.FullMethod] {
			return handler(srv, ss)
		}

		token, err := extractBearerToken(ss.Context())
		if err != nil {
			return err
		}
		userID, err := verifier.Verify(token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid or expired access token")
		}

		ctx := context.WithValue(ss.Context(), userIDContextKey, userID)
		return handler(srv, &authenticatedStream{ServerStream: ss, ctx: ctx})
	}
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
