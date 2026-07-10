package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// promptOTPCodes запрашивает одноразовые коды восстановления (по одному на строку, пустая
// строка = конец). Опциональное поле — можно сразу нажать Enter, чтобы пропустить.
func promptOTPCodes(l *clienti18n.Localizer) ([]secretcontent.OTPCode, error) {
	fmt.Println(l.T("prompt_otp_codes_intro"))
	var codes []secretcontent.OTPCode
	for {
		line, err := promptLine(l.T("prompt_otp_code_line"))
		if err != nil {
			return nil, err
		}
		code := strings.TrimSpace(line)
		if code == "" {
			break
		}
		codes = append(codes, secretcontent.OTPCode{Code: code})
	}
	return codes, nil
}

// printOTPCodes выводит список одноразовых кодов, помечая использованные как [x].
func printOTPCodes(l *clienti18n.Localizer, codes []secretcontent.OTPCode) {
	if len(codes) == 0 {
		return
	}
	fmt.Printf("%s:\n", l.T("label_otp_codes"))
	for i, c := range codes {
		mark := "[ ]"
		if c.Used {
			mark = "[x]"
		}
		fmt.Printf("  %d. %s %s\n", i+1, mark, c.Code)
	}
}

// OTPUseCmd — пометка одноразового кода восстановления как использованного.
// Работает для ЛЮБОГО типа секрета (otp_codes есть в payload каждого типа).
type OTPUseCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id."`
	Index int    `arg:"" help:"Code number (1-based, from 'secret show' output)."`
}

func (c *OTPUseCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	if err := secret.MarkOTPCodeUsed(ctx, vaultID, c.ID, c.Index-1); err != nil {
		return err
	}
	fmt.Println(l.T("otp_code_marked_used"))
	return nil
}
