package cli

import (
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// RenderError переводит ошибку команды в понятный локализованный текст для пользователя.
// Известные sentinel-ошибки маппятся на ключи локализации; всё остальное отдаётся как есть.
func RenderError(l *clienti18n.Localizer, err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, authuc.ErrEmptyLogin):
		return l.T("err_empty_login")
	case errors.Is(err, authuc.ErrEmptyCredential):
		return l.T("err_empty_credential")
	case errors.Is(err, authuc.ErrEmptyPassphrase):
		return l.T("err_empty_passphrase")
	case errors.Is(err, authuc.ErrEncryptionNotSetup):
		return l.T("err_encryption_not_setup")
	case errors.Is(err, errMismatch):
		return l.T("err_entries_mismatch")
	case errors.Is(err, grpcclient.ErrLoginTaken):
		return l.T("err_login_taken")
	case errors.Is(err, grpcclient.ErrInvalidCredentials):
		return l.T("err_invalid_credentials")
	case errors.Is(err, grpcclient.ErrInvalidArgument):
		return l.T("err_invalid_argument")
	case errors.Is(err, grpcclient.ErrNotFound):
		return l.T("err_not_found")
	case errors.Is(err, grpcclient.ErrUnavailable):
		return l.T("err_unavailable")
	case errors.Is(err, keyring.ErrNoToken):
		return l.T("err_no_token")
	case errors.Is(err, errVaultNotFound):
		return l.T("err_vault_not_found")
	case errors.Is(err, errVaultAmbiguous):
		return l.T("err_vault_ambiguous")
	case errors.Is(err, vaultuc.ErrLocked), errors.Is(err, secretuc.ErrVaultLocked):
		return l.T("err_locked")
	case errors.Is(err, secretuc.ErrEmptyTOTPSecret):
		return l.T("err_empty_totp_secret")
	case errors.Is(err, secretuc.ErrInvalidOTPAuthURI):
		return l.T("err_invalid_otpauth_uri")
	case errors.Is(err, secretuc.ErrIndexTooLarge):
		return l.T("err_index_too_large")
	case errors.Is(err, secretuc.ErrEmptyBinaryData):
		return l.T("err_empty_binary_data")
	case errors.Is(err, grpcclient.ErrInternal):
		return l.T("err_internal")
	default:
		return err.Error()
	}
}
