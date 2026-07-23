package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

func testLocalizer() *clienti18n.Localizer {
	return clienti18n.NewLocalizer(clienti18n.NewBundle(), "en")
}

func TestRenderError_MapsSentinels(t *testing.T) {
	l := testLocalizer()

	tests := []struct {
		name string
		err  error
		key  string
	}{
		{"empty login", authuc.ErrEmptyLogin, "err_empty_login"},
		{"empty credential", authuc.ErrEmptyCredential, "err_empty_credential"},
		{"empty passphrase", authuc.ErrEmptyPassphrase, "err_empty_passphrase"},
		{"encryption not setup", authuc.ErrEncryptionNotSetup, "err_encryption_not_setup"},
		{"entries mismatch", errMismatch, "err_entries_mismatch"},
		{"login taken", grpcclient.ErrLoginTaken, "err_login_taken"},
		{"invalid credentials", grpcclient.ErrInvalidCredentials, "err_invalid_credentials"},
		{"invalid argument", grpcclient.ErrInvalidArgument, "err_invalid_argument"},
		{"not found", grpcclient.ErrNotFound, "err_not_found"},
		{"unavailable", grpcclient.ErrUnavailable, "err_unavailable"},
		{"no token", keyring.ErrNoToken, "err_no_token"},
		{"internal", grpcclient.ErrInternal, "err_internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderError(l, tt.err)
			assert.Equal(t, l.T(tt.key), got)
			assert.NotEmpty(t, got)
		})
	}
}

func TestRenderError_WrappedSentinelStillMaps(t *testing.T) {
	l := testLocalizer()
	wrapped := fmt.Errorf("rpc failed: %w", grpcclient.ErrUnavailable)
	assert.Equal(t, l.T("err_unavailable"), RenderError(l, wrapped))
}

func TestRenderError_UnknownReturnsRawText(t *testing.T) {
	l := testLocalizer()
	assert.Equal(t, "some low-level failure", RenderError(l, errors.New("some low-level failure")))
}

func TestRenderError_NilReturnsEmpty(t *testing.T) {
	assert.Empty(t, RenderError(testLocalizer(), nil))
}
