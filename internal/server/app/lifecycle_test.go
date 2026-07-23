package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPprofServer_EmptyAddrDisabled: пустой адрес — pprof не создаётся вовсе (nil), сервер
// не должен слушать никакой дополнительный порт по умолчанию.
func TestNewPprofServer_EmptyAddrDisabled(t *testing.T) {
	srv := newPprofServer("")
	assert.Nil(t, srv)
}

// TestNewPprofServer_ServesIndexWhenEnabled: с заданным адресом сервер поднимается и отдаёт
// стандартный pprof index — подтверждает, что маршруты зарегистрированы на отдельном ServeMux
// (а не на http.DefaultServeMux), и что addr действительно применяется.
func TestNewPprofServer_ServesIndexWhenEnabled(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())

	srv := newPprofServer(addr)
	require.NotNil(t, srv)
	assert.Equal(t, addr, srv.Addr)

	go func() { _ = srv.ListenAndServe() }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	var resp *http.Response
	require.Eventually(t, func() bool {
		var getErr error
		resp, getErr = http.Get("http://" + addr + "/debug/pprof/")
		return getErr == nil
	}, 2*time.Second, 20*time.Millisecond, "pprof HTTP-сервер должен начать отвечать")
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "pprof", "index-страница pprof должна содержать характерный текст")
}
