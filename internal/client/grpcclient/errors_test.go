package grpcclient

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapErr(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want error
	}{
		{"already exists", codes.AlreadyExists, ErrLoginTaken},
		{"unauthenticated", codes.Unauthenticated, ErrInvalidCredentials},
		{"not found", codes.NotFound, ErrNotFound},
		{"unavailable", codes.Unavailable, ErrUnavailable},
		{"deadline exceeded", codes.DeadlineExceeded, ErrUnavailable},
		{"invalid argument", codes.InvalidArgument, ErrInvalidArgument},
		{"internal", codes.Internal, ErrInternal},
		{"unknown code falls back to internal", codes.PermissionDenied, ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapErr(status.Error(tt.code, "msg"))
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestMapErr_Nil(t *testing.T) {
	assert.NoError(t, mapErr(nil))
}

func TestMapErr_OKCode(t *testing.T) {
	assert.NoError(t, mapErr(status.Error(codes.OK, "")))
}

func TestMapErr_NonGRPCErrorPassthrough(t *testing.T) {
	// Ошибка без gRPC-статуса возвращается как есть.
	raw := errors.New("plain error")
	assert.ErrorIs(t, mapErr(raw), raw)
}
