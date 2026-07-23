package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeVerifier struct {
	userID string
	err    error
}

func (f fakeVerifier) Verify(token string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.userID, nil
}

func ctxWithAuth(header string) context.Context {
	if header == "" {
		return context.Background()
	}
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", header))
}

func okHandler(ctx context.Context, req interface{}) (interface{}, error) {
	userID, _ := UserIDFromContext(ctx)
	return userID, nil
}

func TestAuth_SkipsMethodsWithoutAuth(t *testing.T) {
	interceptor := Auth(fakeVerifier{err: assert.AnError})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.AuthService/Login"}

	resp, err := interceptor(context.Background(), nil, info, okHandler)
	require.NoError(t, err)
	assert.Equal(t, "", resp)
}

func TestAuth_MissingMetadata(t *testing.T) {
	interceptor := Auth(fakeVerifier{})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretService/CreateSecret"}

	_, err := interceptor(context.Background(), nil, info, okHandler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	interceptor := Auth(fakeVerifier{})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretService/CreateSecret"}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-other", "value"))

	_, err := interceptor(ctx, nil, info, okHandler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_InvalidHeaderFormat(t *testing.T) {
	interceptor := Auth(fakeVerifier{})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretService/CreateSecret"}
	ctx := ctxWithAuth("Basic abc123")

	_, err := interceptor(ctx, nil, info, okHandler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_VerifierError(t *testing.T) {
	interceptor := Auth(fakeVerifier{err: assert.AnError})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretService/CreateSecret"}
	ctx := ctxWithAuth("Bearer bad-token")

	_, err := interceptor(ctx, nil, info, okHandler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_Success(t *testing.T) {
	interceptor := Auth(fakeVerifier{userID: "user-1"})
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.SecretService/CreateSecret"}
	ctx := ctxWithAuth("Bearer good-token")

	resp, err := interceptor(ctx, nil, info, okHandler)
	require.NoError(t, err)
	assert.Equal(t, "user-1", resp)
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	_, ok := UserIDFromContext(context.Background())
	assert.False(t, ok)
}

// fakeServerStream — минимальная реализация grpc.ServerStream для тестов StreamAuth.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *fakeServerStream) Context() context.Context { return s.ctx }

func TestStreamAuth_SkipsMethodsWithoutAuth(t *testing.T) {
	interceptor := StreamAuth(fakeVerifier{err: assert.AnError})
	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.AuthService/Ping"}
	stream := &fakeServerStream{ctx: context.Background()}

	err := interceptor(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error { return nil })
	require.NoError(t, err)
}

func TestStreamAuth_MissingToken(t *testing.T) {
	interceptor := StreamAuth(fakeVerifier{})
	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.SecretService/Stream"}
	stream := &fakeServerStream{ctx: context.Background()}

	err := interceptor(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error { return nil })
	require.Error(t, err)
}

func TestStreamAuth_VerifierError(t *testing.T) {
	interceptor := StreamAuth(fakeVerifier{err: assert.AnError})
	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.SecretService/Stream"}
	stream := &fakeServerStream{ctx: ctxWithAuth("Bearer bad")}

	err := interceptor(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error { return nil })
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestStreamAuth_Success(t *testing.T) {
	interceptor := StreamAuth(fakeVerifier{userID: "user-2"})
	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.SecretService/Stream"}
	stream := &fakeServerStream{ctx: ctxWithAuth("Bearer good")}

	var gotUserID string
	err := interceptor(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error {
		gotUserID, _ = UserIDFromContext(ss.Context())
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "user-2", gotUserID)
}
