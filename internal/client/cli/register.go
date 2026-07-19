package cli

import (
	"context"
	"fmt"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// RegisterCmd — регистрация нового аккаунта: создаёт пользователя, затем настраивает
// шифрование (EncryptionPassphrase → MasterKey локально).
type RegisterCmd struct {
	Login string `arg:"" optional:"" help:"Account login. Prompted if omitted."`
}

// Run регистрирует новый аккаунт и опционально настраивает шифрование.
func (c *RegisterCmd) Run(uc *authuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()

	login := c.Login
	if login == "" {
		var err error
		login, err = promptLine(l.T("prompt_login"))
		if err != nil {
			return err
		}
	}

	credential, err := promptSecretConfirmed(l.T("prompt_login_credential"), l.T("prompt_login_credential_repeat"), l.T("err_entries_mismatch"))
	if err != nil {
		return err
	}

	if err := uc.Register(ctx, login, credential); err != nil {
		return err
	}
	fmt.Println(l.T("register_success"))

	// Настройка шифрования сразу после регистрации (с подтверждением). Можно пропустить
	// и выполнить позже командой setup-encryption или при следующем login.
	confirmed, err := promptConfirm(l.T("encryption_confirm"))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(l.T("encryption_skipped"))
		return nil
	}

	return runSetupEncryption(ctx, uc, l)
}

func runSetupEncryption(ctx context.Context, uc *authuc.UseCase, l *clienti18n.Localizer) error {
	fmt.Println(l.T("encryption_warning"))

	passphrase, err := promptSecretConfirmed(l.T("prompt_passphrase"), l.T("prompt_passphrase_repeat"), l.T("err_entries_mismatch"))
	if err != nil {
		return err
	}

	if err := uc.SetupEncryption(ctx, passphrase); err != nil {
		return err
	}
	fmt.Println(l.T("encryption_success"))

	// Генерируем и показываем recovery codes
	codes, err := uc.GenerateRecoveryCodes(ctx)
	if err != nil {
		// Шифрование уже настроено — не проваливаем всю команду, просто предупреждаем.
		fmt.Printf("%s: %v\n", l.T("recovery_codes_gen_failed"), err)
		return nil
	}
	printRecoveryCodes(l, codes)
	return nil
}

// printRecoveryCodes выводит сгенерированные recovery codes с предупреждением об их
// одноразовости и необходимости сохранить их в надёжном месте.
func printRecoveryCodes(l *clienti18n.Localizer, codes []string) {
	fmt.Println()
	fmt.Println(l.T("recovery_codes_warning"))
	fmt.Println(l.T("recovery_codes_header"))
	for i, code := range codes {
		fmt.Printf("  %d. %s\n", i+1, code)
	}
	fmt.Println(l.T("recovery_codes_hint"))
}
